package service

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"html"
	"log"
	"strings"
	"time"

	"aiki/internal/domain"
	"aiki/internal/pkg/mailer"
	"aiki/internal/pkg/otp_token"
	"aiki/internal/repository"

	"github.com/redis/go-redis/v9"
)

const (
	emailVerificationTTL      = 10 * time.Minute
	emailVerificationCooldown = 60 * time.Second
)

type EmailVerificationService interface {
	SendVerification(ctx context.Context, userID int32) (*domain.EmailVerificationResponse, error)
	VerifyEmail(ctx context.Context, userID int32, otp string) error
}

type emailVerificationService struct {
	userRepo repository.UserRepository
	redis    *redis.Client
	sender   mailer.Sender
	env      string
}

type emailVerificationPayload struct {
	Otp    string    `json:"otp"`
	Email  string    `json:"email"`
	SentAt time.Time `json:"sent_at"`
}

func NewEmailVerificationService(
	userRepo repository.UserRepository,
	redisClient *redis.Client,
	sender mailer.Sender,
	env string,
) EmailVerificationService {
	return &emailVerificationService{
		userRepo: userRepo,
		redis:    redisClient,
		sender:   sender,
		env:      strings.ToLower(strings.TrimSpace(env)),
	}
}

func (s *emailVerificationService) SendVerification(
	ctx context.Context,
	userID int32,
) (*domain.EmailVerificationResponse, error) {
	isVerified, err := s.userRepo.IsEmailVerified(ctx, userID)
	if err != nil {
		return nil, err
	}
	if isVerified {
		return nil, domain.ErrEmailAlreadyVerified
	}

	if throttled, err := s.isThrottled(ctx, userID); err != nil {
		return nil, err
	} else if throttled {
		return nil, domain.ErrVerificationCodeThrottled
	}

	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	otp, err := otp_token.OTPGenerator()
	if err != nil {
		return nil, domain.ErrInternalServer
	}

	payload := emailVerificationPayload{
		Otp:    otp,
		Email:  user.Email,
		SentAt: time.Now().UTC(),
	}

	value, err := json.Marshal(payload)
	if err != nil {
		return nil, domain.ErrInternalServer
	}

	key := emailVerificationKey(userID)
	if err := s.redis.Set(ctx, key, value, emailVerificationTTL).Err(); err != nil {
		return nil, domain.ErrInternalServer
	}

	if err := s.deliverEmail(ctx, user.Email, otp); err != nil {
		return nil, err
	}

	response := &domain.EmailVerificationResponse{
		ExpiresInSeconds: int(emailVerificationTTL.Seconds()),
		ResendInSeconds:  int(emailVerificationCooldown.Seconds()),
	}
	if s.isNonProduction() {
		response.DebugOtp = otp
	}

	return response, nil
}

func (s *emailVerificationService) VerifyEmail(
	ctx context.Context,
	userID int32,
	otp string,
) error {
	isVerified, err := s.userRepo.IsEmailVerified(ctx, userID)
	if err != nil {
		return err
	}
	if isVerified {
		return nil
	}

	payload, err := s.getPayload(ctx, userID)
	if err != nil {
		return err
	}

	if subtle.ConstantTimeCompare([]byte(strings.TrimSpace(payload.Otp)), []byte(strings.TrimSpace(otp))) != 1 {
		return domain.ErrInvalidVerificationCode
	}

	if err := s.userRepo.MarkEmailVerified(ctx, userID); err != nil {
		return err
	}

	if err := s.redis.Del(ctx, emailVerificationKey(userID)).Err(); err != nil {
		log.Printf("failed to delete email verification key for user %d: %v", userID, err)
	}

	return nil
}

func (s *emailVerificationService) isThrottled(ctx context.Context, userID int32) (bool, error) {
	payload, err := s.getPayload(ctx, userID)
	if err != nil {
		if err == domain.ErrVerificationCodeExpired {
			return false, nil
		}
		return false, err
	}

	return time.Since(payload.SentAt) < emailVerificationCooldown, nil
}

func (s *emailVerificationService) getPayload(ctx context.Context, userID int32) (*emailVerificationPayload, error) {
	value, err := s.redis.Get(ctx, emailVerificationKey(userID)).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, domain.ErrVerificationCodeExpired
		}
		return nil, domain.ErrInternalServer
	}

	var payload emailVerificationPayload
	if err := json.Unmarshal(value, &payload); err != nil {
		return nil, domain.ErrInternalServer
	}

	return &payload, nil
}

func (s *emailVerificationService) deliverEmail(ctx context.Context, recipient, otp string) error {
	subject := "Verify your Aiki email"
	textBody := fmt.Sprintf(
		"Your Aiki verification code is %s. It expires in 10 minutes.",
		otp,
	)
	htmlBody := fmt.Sprintf(
		"<p>Your Aiki verification code is <strong>%s</strong>.</p><p>It expires in 10 minutes.</p>",
		html.EscapeString(otp),
	)

	err := s.sender.Send(ctx, mailer.SendRequest{
		To:      []string{recipient},
		Subject: subject,
		HTML:    htmlBody,
		Text:    textBody,
	})
	if err == nil {
		return nil
	}

	if s.isNonProduction() {
		log.Printf("email verification fallback for %s with otp %s: %v", recipient, otp, err)
		return nil
	}

	return domain.ErrEmailDispatchFailed
}

func (s *emailVerificationService) isNonProduction() bool {
	return s.env == "" || s.env == "development" || s.env == "dev" || s.env == "local"
}

func emailVerificationKey(userID int32) string {
	return fmt.Sprintf("email-verification-%d", userID)
}
