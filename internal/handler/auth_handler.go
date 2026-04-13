package handler

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"aiki/internal/config"
	"aiki/internal/domain"
	"aiki/internal/pkg/response"
	"aiki/internal/service"

	"github.com/labstack/echo/v4"
	"github.com/redis/go-redis/v9"
)

type AuthHandler struct {
	authService              service.AuthService
	emailVerificationService service.EmailVerificationService
	passwordResetService     service.PasswordResetService
	validator                echo.Validator
	config                   config.Config
	redis                    *redis.Client
}

const (
	ExpireInMinute = 5 * time.Minute
	ExpireInHours  = 1 * time.Hour
)

// NewAuthHandler Update the constructor to accept config
func NewAuthHandler(
	authService service.AuthService,
	emailVerificationService service.EmailVerificationService,
	passwordResetService service.PasswordResetService,
	validator echo.Validator,
	redis *redis.Client,
	cfg config.Config,
) *AuthHandler {
	return &AuthHandler{
		authService:              authService,
		emailVerificationService: emailVerificationService,
		passwordResetService:     passwordResetService,
		validator:                validator,
		redis:                    redis,
		config:                   cfg,
	}
}

// Register godoc
// @Summary Register a new user
// @Description Create a new user account
// @Tags auth
// @Accept json
// @Produce json
// @Param request body domain.RegisterRequest true "Registration details"
// @Success 201 {object} response.Response{data=domain.AuthResponse}
// @Failure 400 {object} response.Response
// @Failure 409 {object} response.Response
// @Router /auth/register [post]
func (h *AuthHandler) Register(c echo.Context) error {
	var req domain.RegisterRequest
	if err := c.Bind(&req); err != nil {
		return response.ValidationError(c, "invalid request body")
	}

	if err := h.validator.Validate(&req); err != nil {
		return response.ValidationError(c, err.Error())
	}

	resp, err := h.authService.Register(c.Request().Context(), &req)
	if err != nil {
		return response.Error(c, err)
	}

	if _, err := h.emailVerificationService.SendVerification(c.Request().Context(), resp.User.ID); err != nil {
		c.Logger().Warnf("failed to send verification email after register for user %d: %v", resp.User.ID, err)
	}

	return response.Success(c, http.StatusCreated, "user registered successfully", resp)
}

// Login godoc
// @Summary Login
// @Description Authenticate user and return tokens
// @Tags auth
// @Accept json
// @Produce json
// @Param request body domain.LoginRequest true "Login credentials"
// @Success 200 {object} response.Response{data=domain.AuthResponse}
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Router /auth/login [post]
func (h *AuthHandler) Login(c echo.Context) error {
	var req domain.LoginRequest
	if err := c.Bind(&req); err != nil {
		return response.ValidationError(c, "invalid request body")
	}

	if err := h.validator.Validate(&req); err != nil {
		return response.ValidationError(c, err.Error())
	}

	resp, err := h.authService.Login(c.Request().Context(), &req)
	if err != nil {
		return response.Error(c, err)
	}

	return response.Success(c, http.StatusOK, "login successful", resp)
}

// RefreshToken godoc
// @Summary Refresh access token
// @Description Get a new access token using refresh token
// @Tags auth
// @Accept json
// @Produce json
// @Param request body domain.RefreshTokenRequest true "Refresh token"
// @Success 200 {object} response.Response{data=domain.AuthResponse}
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Router /auth/refresh [post]
func (h *AuthHandler) RefreshToken(c echo.Context) error {
	var req domain.RefreshTokenRequest
	if err := c.Bind(&req); err != nil {
		return response.ValidationError(c, "invalid request body")
	}

	if err := h.validator.Validate(&req); err != nil {
		return response.ValidationError(c, err.Error())
	}

	resp, err := h.authService.RefreshToken(c.Request().Context(), req.RefreshToken)
	if err != nil {
		return response.Error(c, err)
	}

	return response.Success(c, http.StatusOK, "token refreshed successfully", resp)
}

// LinkedInLogin godoc
// @Summary Initiate LinkedIn login
// @Description Redirects user to LinkedIn for authentication
// @Tags auth
// @Produce html
// @Success 302 "Redirects to LinkedIn"
// @Failure 500 {object} response.Response "Internal server error"
// @Router /auth/linkedin/login [get]
func (h *AuthHandler) LinkedInLogin(c echo.Context) error {
	// Generate a random state to protect against CSRF attacks
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		return response.Error(c, domain.ErrInternalServer)
	}
	state := base64.URLEncoding.EncodeToString(b)

	// Set the state in a cookie for later validation
	cookie := &http.Cookie{
		Name:     "linkedin_oauth_state",
		Value:    state,
		Path:     "/",
		Expires:  time.Now().Add(5 * time.Minute), // State valid for 5 minutes
		HttpOnly: true,
		Secure:   h.config.Server.Env == "production", // Only secure in production
		SameSite: http.SameSiteLaxMode,
	}
	c.SetCookie(cookie)

	// Construct the LinkedIn authorization URL
	// Note: LinkedIn's OAuth 2.0 endpoints typically start with https://www.linkedin.com/oauth/v2/
	// You need to register your redirect URI with LinkedIn.
	// For this example, let's assume the redirect URI is /auth/linkedin/callback on your server.
	linkedInAuthURL := "https://www.linkedin.com/oauth/v2/authorization"
	params := url.Values{}
	params.Add("response_type", "code")
	params.Add("client_id", h.config.LinkedIn.ClientID)
	params.Add("redirect_uri", h.config.LinkedIn.ClientCallbackUrl)
	params.Add("state", state)
	params.Add("scope", "openid profile email") // LinkedIn OpenID Connect scopes

	fullURL := linkedInAuthURL + "?" + params.Encode()

	return c.Redirect(http.StatusFound, fullURL)
}

// LinkedInCallback godoc
// @Summary Handle LinkedIn OAuth callback
// @Description Exchanges authorization code for tokens and logs in/registers the user
// @Tags auth
// @Produce json
// @Param code query string true "Authorization code from LinkedIn"
// @Param state query string true "CSRF state parameter"
// @Success 200 {object} response.Response{data=domain.AuthResponse}
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /auth/linkedin/callback [get]
func (h *AuthHandler) LinkedInCallback(c echo.Context) error {
	// Debug logging
	c.Logger().Infof("LinkedIn Callback Hit! Raw Query: %s", c.Request().URL.RawQuery)
	c.Logger().Infof("Query params: %v", c.QueryParams())

	// 1. Validate CSRF state
	code := c.QueryParam("code")
	state := c.QueryParam("state")

	if code == "" || state == "" {
		c.Logger().Errorf("Missing code (%v) or state (%v)", code != "", state != "")
		return response.ValidationError(c, "missing code or state parameter")
	}

	cookie, err := c.Cookie("linkedin_oauth_state")
	if err != nil || cookie.Value != state {
		return response.Error(c, domain.ErrUnauthorized)
	}

	// Clear the state cookie
	c.SetCookie(&http.Cookie{
		Name:     "linkedin_oauth_state",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
	})

	// 2. Exchange authorization code for access token
	tokenURL := "https://www.linkedin.com/oauth/v2/accessToken"
	formData := url.Values{}
	formData.Set("grant_type", "authorization_code")
	formData.Set("code", code)
	formData.Set("redirect_uri", h.config.LinkedIn.ClientCallbackUrl)
	formData.Set("client_id", h.config.LinkedIn.ClientID)
	formData.Set("client_secret", h.config.LinkedIn.ClientSecret)

	tokenReq, err := http.NewRequest(http.MethodPost, tokenURL, strings.NewReader(formData.Encode()))
	if err != nil {
		c.Logger().Errorf("failed to create token request: %v", err)
		return response.Error(c, domain.ErrInternalServer)
	}
	tokenReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{Timeout: 10 * time.Second}
	tokenResp, err := client.Do(tokenReq)
	if err != nil {
		c.Logger().Errorf("failed to exchange code for token: %v", err)
		return response.Error(c, domain.ErrInternalServer)
	}
	defer tokenResp.Body.Close()

	if tokenResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(tokenResp.Body)
		c.Logger().Errorf("linkedin token exchange failed: status=%d body=%s", tokenResp.StatusCode, string(body))
		return response.Error(c, domain.ErrInternalServer)
	}

	var tokenData struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.NewDecoder(tokenResp.Body).Decode(&tokenData); err != nil {
		c.Logger().Errorf("failed to decode token response: %v", err)
		return response.Error(c, domain.ErrInternalServer)
	}

	// 3. Fetch user info from LinkedIn using OpenID Connect userinfo endpoint
	userInfoReq, err := http.NewRequest(http.MethodGet, "https://api.linkedin.com/v2/userinfo", nil)
	if err != nil {
		c.Logger().Errorf("failed to create userinfo request: %v", err)
		return response.Error(c, domain.ErrInternalServer)
	}
	userInfoReq.Header.Set("Authorization", "Bearer "+tokenData.AccessToken)

	userInfoResp, err := client.Do(userInfoReq)
	if err != nil {
		c.Logger().Errorf("failed to fetch linkedin user info: %v", err)
		return response.Error(c, domain.ErrInternalServer)
	}
	defer userInfoResp.Body.Close()

	if userInfoResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(userInfoResp.Body)
		c.Logger().Errorf("linkedin userinfo failed: status=%d body=%s", userInfoResp.StatusCode, string(body))
		return response.Error(c, domain.ErrInternalServer)
	}

	var linkedInUser struct {
		Sub        string `json:"sub"` // LinkedIn unique user ID
		Email      string `json:"email"`
		GivenName  string `json:"given_name"`
		FamilyName string `json:"family_name"`
	}
	if err := json.NewDecoder(userInfoResp.Body).Decode(&linkedInUser); err != nil {
		c.Logger().Errorf("failed to decode userinfo response: %v", err)
		return response.Error(c, domain.ErrInternalServer)
	}

	if linkedInUser.Sub == "" || linkedInUser.Email == "" {
		c.Logger().Error("linkedin returned empty sub or email")
		return response.Error(c, domain.ErrInternalServer)
	}

	// 4. Login or register via service layer
	authResp, err := h.authService.LinkedInLogin(c.Request().Context(), linkedInUser.Sub, linkedInUser.Email, linkedInUser.GivenName, linkedInUser.FamilyName)
	if err != nil {
		return response.Error(c, err)
	}

	return response.Success(c, http.StatusOK, "linkedin login successful", authResp)
}

// Logout godoc
// @Summary Logout
// @Description Invalidate refresh token
// @Tags auth
// @Accept json
// @Produce json
// @Param request body domain.RefreshTokenRequest true "Refresh token"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Router /auth/logout [post]
func (h *AuthHandler) Logout(c echo.Context) error {
	var req domain.RefreshTokenRequest
	if err := c.Bind(&req); err != nil {
		return response.ValidationError(c, "invalid request body")
	}

	if err := h.validator.Validate(&req); err != nil {
		return response.ValidationError(c, err.Error())
	}

	if err := h.authService.Logout(c.Request().Context(), req.RefreshToken); err != nil {
		return response.Error(c, err)
	}

	return response.Success(c, http.StatusOK, "logout successful", nil)
}

func (h *AuthHandler) ForgottenPassword(c echo.Context) error {
	var req domain.ForgotPasswordRequest
	if err := c.Bind(&req); err != nil {
		return response.ValidationError(c, "invalid request body")
	}

	if err := h.validator.Validate(&req); err != nil {
		return response.ValidationError(c, err.Error())
	}

	data, err := h.passwordResetService.SendResetCode(c.Request().Context(), req.Email)
	if err != nil {
		return response.Error(c, err)
	}

	return response.Success(c, http.StatusOK, "if the email exists, a password reset code has been sent", data)
}

func (h *AuthHandler) ValidateForgottenPasswordOTP(c echo.Context) error {
	var req domain.ValidateForgotPasswordOTP
	if err := c.Bind(&req); err != nil {
		return response.ValidationError(c, "invalid request body")
	}
	if err := h.validator.Validate(&req); err != nil {
		return response.ValidationError(c, err.Error())
	}

	if err := h.passwordResetService.VerifyResetCode(c.Request().Context(), req.SessionId, req.Otp); err != nil {
		return response.Error(c, err)
	}

	return response.Success(c, http.StatusOK, "user validation successful", nil)
}

func (h *AuthHandler) ResetPassword(c echo.Context) error {
	var req domain.ResetPasswordRequest
	if err := c.Bind(&req); err != nil {
		return response.ValidationError(c, "invalid request body")
	}

	if err := h.validator.Validate(&req); err != nil {
		return response.ValidationError(c, err.Error())
	}

	email, err := h.passwordResetService.GetVerifiedEmail(c.Request().Context(), req.SessionId)
	if err != nil {
		return response.Error(c, err)
	}

	if err := h.authService.ResetPassword(c.Request().Context(), email, req.NewPassword); err != nil {
		return response.Error(c, err)
	}

	if err := h.passwordResetService.ClearSession(c.Request().Context(), req.SessionId); err != nil {
		c.Logger().Errorf("failed to clear password reset session: %v", err)
	}
	return response.Success(c, http.StatusOK, "password has been reset successfully", nil)
}

// VerifyEmail godoc
// @Summary Verify email
// @Description Verify the authenticated user's email with a one-time code
// @Tags auth
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body domain.VerifyEmailRequest true "Verification code"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 410 {object} response.Response
// @Router /auth/verify-email [post]
func (h *AuthHandler) VerifyEmail(c echo.Context) error {
	userID, ok := c.Get("user_id").(int32)
	if !ok {
		return response.Error(c, domain.ErrUnauthorized)
	}

	var req domain.VerifyEmailRequest
	if err := c.Bind(&req); err != nil {
		return response.ValidationError(c, "invalid request body")
	}
	if err := h.validator.Validate(&req); err != nil {
		return response.ValidationError(c, err.Error())
	}

	if err := h.emailVerificationService.VerifyEmail(c.Request().Context(), userID, req.Otp); err != nil {
		return response.Error(c, err)
	}

	return response.Success(c, http.StatusOK, "email verified successfully", nil)
}

// ResendEmailVerification godoc
// @Summary Resend verification email
// @Description Send a new verification code to the authenticated user's email
// @Tags auth
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.Response{data=domain.EmailVerificationResponse}
// @Failure 401 {object} response.Response
// @Failure 409 {object} response.Response
// @Failure 429 {object} response.Response
// @Router /auth/verify-email/resend [post]
func (h *AuthHandler) ResendEmailVerification(c echo.Context) error {
	userID, ok := c.Get("user_id").(int32)
	if !ok {
		return response.Error(c, domain.ErrUnauthorized)
	}

	data, err := h.emailVerificationService.SendVerification(c.Request().Context(), userID)
	if err != nil {
		return response.Error(c, err)
	}

	return response.Success(c, http.StatusOK, "verification email sent successfully", data)
}
