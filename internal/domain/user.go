package domain

import (
	"time"
)

// User represents a user in the system
type User struct {
	ID           int32     `json:"id"`
	FirstName    *string   `json:"first_name"`
	LastName     *string   `json:"last_name"`
	Email        string    `json:"email"`
	PhoneNumber  *string   `json:"phone_number,omitempty"`
	PasswordHash *string   `json:"-"`                     // Never expose password hash, now nullable
	LinkedInID   *string   `json:"linkedin_id,omitempty"` // Nullable LinkedIn ID
	IsActive     bool      `json:"is_active"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// UserProfile Details
type UserProfile struct {
	UserId          int32     `json:"user_id"`
	FullName        string    `json:"full_name"`
	CurrentJob      string    `json:"current_job"`
	ExperienceLevel string    `json:"experience_level"`
	Goals           []string  `json:"goals"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type UserProfileRequest struct {
	FullName        string   `json:"full_name" validate:"required,min=7,max=200"`
	CurrentJob      string   `json:"current_job" validate:"required,min=5,max=200"`
	ExperienceLevel string   `json:"experience_level" validate:"required,min=5,max=200"`
	Goals           []string `json:"goals,omitempty" validate:"omitempty"`
}

// RegisterRequest represents the request to register a new user
type RegisterRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=8,max=100"`
}

// LoginRequest represents the request to login
type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

// AuthResponse represents the response after successful authentication
type AuthResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	User         *User  `json:"user"`
}

// RefreshTokenRequest represents the request to refresh access token
type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

// UpdateUserRequest represents the request to update user information
type UpdateUserRequest struct {
	FirstName   *string `json:"first_name,omitempty" validate:"omitempty,min=2,max=100"`
	LastName    *string `json:"last_name,omitempty" validate:"omitempty,min=2,max=100"`
	PhoneNumber *string `json:"phone_number,omitempty" validate:"omitempty,min=10,max=20"`
}

type ForgotPasswordRequest struct {
	Email string `json:"email" validate:"required,email"`
}

type ValidateForgotPasswordOTP struct {
	SessionId string `json:"session_id" validate:"required"`
	Otp       string `json:"otp" validate:"required"`
}

type ResetPasswordRequest struct {
	NewPassword string `json:"new_password" validate:"required,min=8,max=100"`
	SessionId   string `json:"session_id" validate:"required"`
}
