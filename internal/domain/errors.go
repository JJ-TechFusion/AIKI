package domain

import (
	"errors"
	"net/http"
)

// Domain errors
var (
	ErrUserNotFound             = errors.New("user not found")
	ErrUserAlreadyExists        = errors.New("user already exists")
	ErrInvalidCredentials       = errors.New("invalid credentials")
	ErrInvalidToken             = errors.New("invalid token")
	ErrTokenExpired             = errors.New("token expired")
	ErrUnauthorized             = errors.New("unauthorized")
	ErrInvalidInput             = errors.New("invalid input")
	ErrInternalServer           = errors.New("internal server error")
	ErrEmailAlreadyExists       = errors.New("email already exists")
	ErrWeakPassword             = errors.New("password is too weak")
	ErrUserProfileNotCreated    = errors.New("user profile not created")
	ErrUserProfileAlreadyExists = errors.New("user profile already exists")
	ErrFileSizeExceedsLimit     = errors.New("file size exceeds limit")
	ErrFailedToUpload           = errors.New("failed to upload file")
	ErrInvalidDateFormat        = errors.New("invalid date format")
	ErrFailedToCreateJob        = errors.New("failed to create job")
	ErrFailedToUpdateJob        = errors.New("failed to update job")
	ErrInvalidJobID             = errors.New("invalid job id")
)

// AppError represents an application error with HTTP status code
type AppError struct {
	Err        error
	Message    string
	StatusCode int
}

func (e *AppError) Error() string {
	if e.Message != "" {
		return e.Message
	}
	return e.Err.Error()
}

// NewAppError creates a new AppError
func NewAppError(err error, message string, statusCode int) *AppError {
	return &AppError{
		Err:        err,
		Message:    message,
		StatusCode: statusCode,
	}
}

// GetHTTPStatus returns the HTTP status code for an error
func GetHTTPStatus(err error) int {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr.StatusCode
	}

	switch {
	case errors.Is(err, ErrUserNotFound):
		return http.StatusNotFound
	case errors.Is(err, ErrUserAlreadyExists), errors.Is(err, ErrEmailAlreadyExists):
		return http.StatusConflict
	case errors.Is(err, ErrInvalidCredentials), errors.Is(err, ErrUnauthorized):
		return http.StatusUnauthorized
	case errors.Is(err, ErrInvalidToken), errors.Is(err, ErrTokenExpired):
		return http.StatusUnauthorized
	case errors.Is(err, ErrInvalidInput), errors.Is(err, ErrWeakPassword), errors.Is(err, ErrEmptyMessages):
		return http.StatusBadRequest
	case errors.Is(err, ErrProviderNotFound):
		return http.StatusUnprocessableEntity
	case errors.Is(err, ErrProviderNotConfigured):
		return http.StatusServiceUnavailable
	default:
		return http.StatusInternalServerError
	}
}
