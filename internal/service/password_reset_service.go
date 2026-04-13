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

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

const (
	passwordResetTTL      = 10 * time.Minute
	passwordResetCooldown = 60 * time.Second
)

type PasswordResetService interface {
	SendResetCode(ctx context.Context, email string) (*domain.ForgotPasswordResponse, error)
	VerifyResetCode(ctx context.Context, sessionID, otp string) error
	GetVerifiedEmail(ctx context.Context, sessionID string) (string, error)
	ClearSession(ctx context.Context, sessionID string) error
}

type passwordResetService struct {
	userRepo repository.UserRepository
	redis    *redis.Client
	sender   mailer.Sender
	env      string
}

type passwordResetPayload struct {
	Email    string    `json:"email,omitempty"`
	Otp      string    `json:"otp,omitempty"`
	SentAt   time.Time `json:"sent_at"`
	Verified bool      `json:"verified"`
}

func NewPasswordResetService(
	userRepo repository.UserRepository,
	redisClient *redis.Client,
	sender mailer.Sender,
	env string,
) PasswordResetService {
	return &passwordResetService{
		userRepo: userRepo,
		redis:    redisClient,
		sender:   sender,
		env:      strings.ToLower(strings.TrimSpace(env)),
	}
}

func (s *passwordResetService) SendResetCode(ctx context.Context, email string) (*domain.ForgotPasswordResponse, error) {
	sessionID := uuid.NewString()
	normalizedEmail := strings.TrimSpace(strings.ToLower(email))

	user, err := s.userRepo.GetByEmail(ctx, normalizedEmail)
	if err != nil {
		if err == domain.ErrUserNotFound {
			return &domain.ForgotPasswordResponse{SessionID: sessionID}, nil
		}
		return nil, err
	}

	if throttled, err := s.isThrottled(ctx, user.Email); err != nil {
		return nil, err
	} else if throttled {
		return nil, domain.ErrVerificationCodeThrottled
	}

	otp, err := otp_token.OTPGenerator()
	if err != nil {
		return nil, domain.ErrInternalServer
	}

	payload := passwordResetPayload{
		Email:    user.Email,
		Otp:      otp,
		SentAt:   time.Now().UTC(),
		Verified: false,
	}

	if err := s.persistPayload(ctx, sessionID, payload, passwordResetTTL); err != nil {
		return nil, err
	}

	if err := s.redis.Set(ctx, passwordResetEmailKey(user.Email), sessionID, passwordResetTTL).Err(); err != nil {
		_ = s.redis.Del(ctx, passwordResetSessionKey(sessionID)).Err()
		return nil, domain.ErrInternalServer
	}

	if err := s.deliverEmail(ctx, user.Email, otp); err != nil {
		_ = s.redis.Del(ctx, passwordResetSessionKey(sessionID), passwordResetEmailKey(user.Email)).Err()
		return nil, err
	}

	return &domain.ForgotPasswordResponse{SessionID: sessionID}, nil
}

func (s *passwordResetService) VerifyResetCode(ctx context.Context, sessionID, otp string) error {
	payload, err := s.getPayload(ctx, sessionID)
	if err != nil {
		return err
	}

	if subtle.ConstantTimeCompare([]byte(strings.TrimSpace(payload.Otp)), []byte(strings.TrimSpace(otp))) != 1 {
		return domain.ErrInvalidVerificationCode
	}

	payload.Verified = true

	ttl, err := s.redis.TTL(ctx, passwordResetSessionKey(sessionID)).Result()
	if err != nil {
		return domain.ErrInternalServer
	}
	if ttl <= 0 {
		ttl = passwordResetTTL
	}

	return s.persistPayload(ctx, sessionID, *payload, ttl)
}

func (s *passwordResetService) GetVerifiedEmail(ctx context.Context, sessionID string) (string, error) {
	payload, err := s.getPayload(ctx, sessionID)
	if err != nil {
		return "", err
	}
	if !payload.Verified || strings.TrimSpace(payload.Email) == "" {
		return "", domain.ErrUnauthorized
	}
	return payload.Email, nil
}

func (s *passwordResetService) ClearSession(ctx context.Context, sessionID string) error {
	payload, err := s.getPayload(ctx, sessionID)
	if err != nil && err != domain.ErrVerificationCodeExpired {
		return err
	}

	keys := []string{passwordResetSessionKey(sessionID)}
	if payload != nil && strings.TrimSpace(payload.Email) != "" {
		keys = append(keys, passwordResetEmailKey(payload.Email))
	}

	if err := s.redis.Del(ctx, keys...).Err(); err != nil {
		return domain.ErrInternalServer
	}

	return nil
}

func (s *passwordResetService) isThrottled(ctx context.Context, email string) (bool, error) {
	sessionID, err := s.redis.Get(ctx, passwordResetEmailKey(email)).Result()
	if err != nil {
		if err == redis.Nil {
			return false, nil
		}
		return false, domain.ErrInternalServer
	}

	payload, err := s.getPayload(ctx, sessionID)
	if err != nil {
		if err == domain.ErrVerificationCodeExpired {
			return false, nil
		}
		return false, err
	}

	return time.Since(payload.SentAt) < passwordResetCooldown, nil
}

func (s *passwordResetService) getPayload(ctx context.Context, sessionID string) (*passwordResetPayload, error) {
	value, err := s.redis.Get(ctx, passwordResetSessionKey(sessionID)).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, domain.ErrVerificationCodeExpired
		}
		return nil, domain.ErrInternalServer
	}

	var payload passwordResetPayload
	if err := json.Unmarshal(value, &payload); err != nil {
		return nil, domain.ErrInternalServer
	}

	return &payload, nil
}

func (s *passwordResetService) persistPayload(ctx context.Context, sessionID string, payload passwordResetPayload, ttl time.Duration) error {
	value, err := json.Marshal(payload)
	if err != nil {
		return domain.ErrInternalServer
	}

	if err := s.redis.Set(ctx, passwordResetSessionKey(sessionID), value, ttl).Err(); err != nil {
		return domain.ErrInternalServer
	}

	return nil
}

func (s *passwordResetService) deliverEmail(ctx context.Context, recipient, otp string) error {
	subject := "Your Aiki password reset code"
	textBody := fmt.Sprintf(
		"Your Aiki password reset code is %s. It expires in 10 minutes.",
		otp,
	)
	htmlBody := fmt.Sprintf(
		"<p>Your Aiki password reset code is <strong>%s</strong>.</p><p>It expires in 10 minutes.</p>",
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
		log.Printf("password reset email fallback for %s with otp %s: %v", recipient, otp, err)
		return nil
	}

	return domain.ErrEmailDispatchFailed
}

func (s *passwordResetService) isNonProduction() bool {
	return s.env == "" || s.env == "development" || s.env == "dev" || s.env == "local"
}

func passwordResetSessionKey(sessionID string) string {
	return fmt.Sprintf("password-reset-session-%s", sessionID)
}

func passwordResetEmailKey(email string) string {
	return fmt.Sprintf("password-reset-email-%s", strings.ToLower(strings.TrimSpace(email)))
}
