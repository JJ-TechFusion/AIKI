package domain

import (
	"errors"
	"net/http"
)

// Domain errors
var (
	ErrUserNotFound              = errors.New("user not found")
	ErrUserAlreadyExists         = errors.New("user already exists")
	ErrInvalidCredentials        = errors.New("invalid credentials")
	ErrInvalidToken              = errors.New("invalid token")
	ErrTokenExpired              = errors.New("token expired")
	ErrUnauthorized              = errors.New("unauthorized")
	ErrInvalidInput              = errors.New("invalid input")
	ErrInternalServer            = errors.New("internal server error")
	ErrEmailAlreadyExists        = errors.New("email already exists")
	ErrEmailAlreadyVerified      = errors.New("email already verified")
	ErrWeakPassword              = errors.New("password is too weak")
	ErrUserProfileNotCreated     = errors.New("user profile not created")
	ErrUserProfileAlreadyExists  = errors.New("user profile already exists")
	ErrInvalidVerificationCode   = errors.New("invalid verification code")
	ErrVerificationCodeExpired   = errors.New("verification code expired")
	ErrVerificationCodeThrottled = errors.New("verification code recently sent")
	ErrEmailDispatchFailed       = errors.New("failed to send verification email")
	ErrFileSizeExceedsLimit      = errors.New("file size exceeds limit")
	ErrFailedToUpload            = errors.New("failed to upload file")
	ErrInvalidDateFormat         = errors.New("invalid date format")
	ErrFailedToCreateJob         = errors.New("failed to create job")
	ErrFailedToUpdateJob         = errors.New("failed to update job")
	ErrInvalidJobID              = errors.New("invalid job id")
	ErrJobAlreadyTracked         = errors.New("job already saved to tracker")
	ErrJobAlreadyApplied         = errors.New("job already applied")
	ErrNoApplyLink               = errors.New("this listing has no apply link")
	ErrCVNotFound                = errors.New("cv not found")
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
	case errors.Is(err, ErrUserNotFound), errors.Is(err, ErrCVNotFound):
		return http.StatusNotFound
	case errors.Is(err, ErrUserAlreadyExists), errors.Is(err, ErrEmailAlreadyExists), errors.Is(err, ErrUserProfileAlreadyExists):
		return http.StatusConflict
	case errors.Is(err, ErrEmailAlreadyVerified):
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
	case errors.Is(err, ErrInvalidInput), errors.Is(err, ErrWeakPassword), errors.Is(err, ErrNoApplyLink), errors.Is(err, ErrInvalidVerificationCode):
		return http.StatusBadRequest
	case errors.Is(err, ErrVerificationCodeExpired):
		return http.StatusGone
	case errors.Is(err, ErrVerificationCodeThrottled):
		return http.StatusTooManyRequests
	case errors.Is(err, ErrEmailDispatchFailed):
		return http.StatusServiceUnavailable
	case errors.Is(err, ErrInvalidJobID):
		return http.StatusNotFound
	case errors.Is(err, ErrJobAlreadyTracked), errors.Is(err, ErrJobAlreadyApplied):
		return http.StatusConflict
	default:
		return http.StatusInternalServerError
	}
}
