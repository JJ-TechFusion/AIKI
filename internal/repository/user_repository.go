package repository

import (
	"aiki/internal/database/db"
	"context"
	"errors"
	"fmt"
	"time"

	"aiki/internal/domain"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

//go:generate mockgen -source=user_repository.go -destination=mocks/mock_user_repository.go -package=mocks

type UserRepository interface {
	Create(ctx context.Context, user *domain.User, password string) (*domain.User, error)
	GetByID(ctx context.Context, id int32) (*domain.User, error)
	GetByEmail(ctx context.Context, email string) (*domain.User, error)
	Update(ctx context.Context, id int32, firstName, lastName *string, phoneNumber *string) (*domain.User, error)
	EmailExists(ctx context.Context, email string) (bool, error)
	CreateRefreshToken(ctx context.Context, userID int32, token string, expiresAt time.Time) error
	GetRefreshToken(ctx context.Context, token string) (int32, error)
	DeleteRefreshToken(ctx context.Context, token string) error
	DeleteUserRefreshTokens(ctx context.Context, userID int32) error
	UpdateUserPassword(ctx context.Context, userID int32, newPasswordHash string) error
	CreateUserProfile(ctx context.Context, userId int32, fullName, currentJob, experienceLevel *string) (*domain.UserProfile, error)
	UpdateUserProfile(ctx context.Context, userId int32, fullName, currentJob, experienceLevel *string) (*domain.UserProfile, error)
	GetUserProfileByID(ctx context.Context, userId int32) (*domain.UserProfile, error)
	UploadCV(ctx context.Context, userId int32, data []byte) error
}

type userRepository struct {
	db      *pgxpool.Pool
	queries *db.Queries
}

func NewUserRepository(dbPool *pgxpool.Pool) UserRepository {
	return &userRepository{db: dbPool, queries: db.New(dbPool)}
}

func (r *userRepository) Create(ctx context.Context, user *domain.User, passwordHash string) (*domain.User, error) {
	query := `
		INSERT INTO users (email, password_hash)
		VALUES ($1, $2)
		RETURNING id, first_name, last_name, email, phone_number, is_active, created_at, updated_at
	`

	var createdUser domain.User
	err := r.db.QueryRow(ctx, query,
		user.Email,
		passwordHash,
	).Scan(
		&createdUser.ID,
		&createdUser.FirstName,
		&createdUser.LastName,
		&createdUser.Email,
		&createdUser.PhoneNumber,
		&createdUser.IsActive,
		&createdUser.CreatedAt,
		&createdUser.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrUserAlreadyExists
		}
		return nil, err
	}
	return &createdUser, nil
}

func (r *userRepository) GetByID(ctx context.Context, id int32) (*domain.User, error) {
	query := `
		SELECT id, first_name, last_name, email, phone_number, password_hash, is_active, created_at, updated_at
		FROM users
		WHERE id = $1 AND is_active = TRUE
	`

	var user domain.User
	err := r.db.QueryRow(ctx, query, id).Scan(
		&user.ID,
		&user.FirstName,
		&user.LastName,
		&user.Email,
		&user.PhoneNumber,
		&user.PasswordHash,
		&user.IsActive,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrUserNotFound
		}
		return nil, err
	}

	return &user, nil
}

func (r *userRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	query := `
		SELECT id, first_name, last_name, email, phone_number, password_hash, is_active, created_at, updated_at
		FROM users
		WHERE email = $1 AND is_active = TRUE
	`

	var user domain.User
	err := r.db.QueryRow(ctx, query, email).Scan(
		&user.ID,
		&user.FirstName,
		&user.LastName,
		&user.Email,
		&user.PhoneNumber,
		&user.PasswordHash,
		&user.IsActive,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrUserNotFound
		}
		return nil, err
	}

	return &user, nil
}

func (r *userRepository) Update(ctx context.Context, id int32, firstName, lastName *string, phoneNumber *string) (*domain.User, error) {
	query := `
		UPDATE users
		SET
			first_name = COALESCE($2, first_name),
			last_name = COALESCE($3, last_name),
			phone_number = COALESCE($4, phone_number),
			updated_at = NOW()
		WHERE id = $1 AND is_active = TRUE
		RETURNING id, first_name, last_name, email, phone_number, is_active, created_at, updated_at
	`

	var user domain.User
	err := r.db.QueryRow(ctx, query, id, firstName, lastName, phoneNumber).Scan(
		&user.ID,
		&user.FirstName,
		&user.LastName,
		&user.Email,
		&user.PhoneNumber,
		&user.IsActive,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrUserNotFound
		}
		return nil, err
	}

	return &user, nil
}

func (r *userRepository) EmailExists(ctx context.Context, email string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM users WHERE email = $1)`

	var exists bool
	err := r.db.QueryRow(ctx, query, email).Scan(&exists)
	if err != nil {
		return false, err
	}

	return exists, nil
}

func (r *userRepository) CreateRefreshToken(ctx context.Context, userID int32, token string, expiresAt time.Time) error {
	query := `
		INSERT INTO refresh_tokens (user_id, token, expires_at)
		VALUES ($1, $2, $3)
	`

	_, err := r.db.Exec(ctx, query, userID, token, expiresAt)
	return err
}

func (r *userRepository) GetRefreshToken(ctx context.Context, token string) (int32, error) {
	query := `
		SELECT user_id FROM refresh_tokens
		WHERE token = $1 AND expires_at > NOW()
	`

	var userID int32
	err := r.db.QueryRow(ctx, query, token).Scan(&userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, domain.ErrInvalidToken
		}
		return 0, err
	}

	return userID, nil
}

func (r *userRepository) DeleteRefreshToken(ctx context.Context, token string) error {
	query := `DELETE FROM refresh_tokens WHERE token = $1`
	_, err := r.db.Exec(ctx, query, token)
	return err
}

func (r *userRepository) DeleteUserRefreshTokens(ctx context.Context, userID int32) error {
	query := `DELETE FROM refresh_tokens WHERE user_id = $1`
	_, err := r.db.Exec(ctx, query, userID)
	return err
}

func (r *userRepository) UpdateUserPassword(ctx context.Context, userID int32, newPasswordHash string) error {
	query := `
		UPDATE users
		SET password_hash = $2, updated_at = NOW()
		WHERE id = $1 AND is_active = TRUE
	`

	_, err := r.db.Exec(ctx, query, userID, newPasswordHash)
	return err
}

func (r *userRepository) CreateUserProfile(ctx context.Context, userId int32, fullName, currentJob, experienceLevel *string) (*domain.UserProfile, error) {
	profile, err := r.queries.CreateUserProfile(ctx, db.CreateUserProfileParams{
		UserID:          userId,
		FullName:        fullName,
		CurrentJob:      currentJob,
		ExperienceLevel: experienceLevel,
	})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			switch pgErr.Code {
			case "23505": // unique_violation
				return nil, domain.ErrUserProfileAlreadyExists
			case "23503": // foreign_key_violation
				return nil, domain.ErrUserProfileAlreadyExists
			}
		}
		fmt.Println("Error creating user profile:", err)
		return nil, domain.ErrUserProfileNotCreated
	}
	fmt.Println("User profile created successfully")
	return &domain.UserProfile{
		UserId:          profile.UserID,
		FullName:        *profile.FullName,
		CurrentJob:      *profile.CurrentJob,
		ExperienceLevel: *profile.ExperienceLevel,
		UpdatedAt:       profile.UpdatedAt.Time,
	}, nil
}

func (r *userRepository) GetUserProfileByID(ctx context.Context, userId int32) (*domain.UserProfile, error) {
	profile, err := r.queries.GetUserProfileByUserID(ctx, userId)
	if err != nil {
		fmt.Println("Error getting user profile:", err)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrUserNotFound
		}
		return nil, err
	}
	return &domain.UserProfile{
		UserId:          profile.UserID,
		FullName:        *profile.FullName,
		CurrentJob:      *profile.CurrentJob,
		ExperienceLevel: *profile.ExperienceLevel,
		UpdatedAt:       profile.UpdatedAt.Time,
	}, nil
}

func (r *userRepository) UpdateUserProfile(ctx context.Context, userId int32, fullName, currentJob, experienceLevel *string) (*domain.UserProfile, error) {
	profile, err := r.queries.UpdateUserProfile(ctx, db.UpdateUserProfileParams{
		UserID:          userId,
		FullName:        fullName,
		CurrentJob:      currentJob,
		ExperienceLevel: experienceLevel,
	})
	if err != nil {
		return nil, err
	}

	return &domain.UserProfile{
		UserId:          profile.UserID,
		FullName:        *profile.FullName,
		CurrentJob:      *profile.CurrentJob,
		ExperienceLevel: *profile.ExperienceLevel,
		UpdatedAt:       profile.UpdatedAt.Time,
	}, nil
}

func (r *userRepository) UploadCV(ctx context.Context, userId int32, data []byte) error {
	const maxSize = 5 * 1024 * 1024
	if len(data) > maxSize {
		return domain.ErrFileSizeExceedsLimit
	}
	_, err := r.queries.UploadUserCV(ctx, db.UploadUserCVParams{
		UserID: userId,
		Cv:     data,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.ErrFailedToUpload
		}
		fmt.Println("Error uploading CV:", err)
		return domain.ErrInternalServer
	}
	return nil
}

//func (r *userRepository) GetCV(ctx context.Context, userId int32) (byt)
