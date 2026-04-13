package handler

import (
	"aiki/internal/domain"
	"aiki/internal/pkg/response"
	"aiki/internal/service"
	"net/http"

	"github.com/labstack/echo/v4"
)

type UserHandler struct {
	userService service.UserService
	validator   echo.Validator
}

func NewUserHandler(userService service.UserService, validator echo.Validator) *UserHandler {
	return &UserHandler{
		userService: userService,
		validator:   validator,
	}
}

// GetMe godoc
// @Summary      Get current user
// @Description  Get the currently authenticated user's profile
// @Tags         users
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Success      200 {object} response.Response{data=domain.User}
// @Failure      401 {object} response.Response
// @Failure      404 {object} response.Response
// @Router       /users/me [get]
func (h *UserHandler) GetMe(c echo.Context) error {
	userID, ok := c.Get("user_id").(int32)
	if !ok {
		return response.Error(c, domain.ErrUnauthorized)
	}

	user, err := h.userService.GetByID(c.Request().Context(), userID)
	if err != nil {
		return response.Error(c, err)
	}

	return response.Success(c, http.StatusOK, "user retrieved successfully", user)
}

// UpdateMe godoc
// @Summary      Update current user
// @Description  Update the currently authenticated user's profile
// @Tags         users
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request body domain.UpdateUserRequest true "Update user details"
// @Success      200 {object} response.Response{data=domain.User}
// @Failure      400 {object} response.Response
// @Failure      401 {object} response.Response
// @Failure      404 {object} response.Response
// @Router       /users/me [put]
func (h *UserHandler) UpdateMe(c echo.Context) error {
	userID, ok := c.Get("user_id").(int32)
	if !ok {
		return response.Error(c, domain.ErrUnauthorized)
	}

	var req domain.UpdateUserRequest
	if err := c.Bind(&req); err != nil {
		return response.ValidationError(c, "invalid request body")
	}

	if err := h.validator.Validate(&req); err != nil {
		return response.ValidationError(c, err.Error())
	}

	user, err := h.userService.Update(c.Request().Context(), userID, &req)
	if err != nil {
		return response.Error(c, err)
	}

	return response.Success(c, http.StatusOK, "user updated successfully", user)
}

// CreateProfile godoc
// @Summary      Create user profile
// @Description  Create a profile for the currently authenticated user
// @Tags         users
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request body domain.UserProfileRequest true "Profile details"
// @Success      200 {object} response.Response{data=domain.UserProfile}
// @Failure      400 {object} response.Response
// @Failure      401 {object} response.Response
// @Failure      409 {object} response.Response
// @Router       /users/profile [post]
func (h *UserHandler) CreateProfile(c echo.Context) error {
	id, ok := c.Get("user_id").(int32)
	if !ok {
		return response.Error(c, domain.ErrUnauthorized)
	}
	var req domain.UserProfileRequest
	if err := c.Bind(&req); err != nil {
		return response.ValidationError(c, "invalid request body")
	}
	if err := h.validator.Validate(&req); err != nil {
		return response.ValidationError(c, err.Error())
	}
	pf := domain.UserProfile{
		UserId:          id,
		FullName:        req.FullName,
		CurrentJob:      req.CurrentJob,
		ExperienceLevel: req.ExperienceLevel,
		Goals:           req.Goals,
	}

	user, err := h.userService.CreateUserProfile(c.Request().Context(), pf)
	if err != nil {
		return response.Error(c, err)
	}
	return response.Success(c, http.StatusOK, "user profile successfully", user)
}

// UpdateProfile godoc
// @Summary      Update user profile
// @Description  Update the currently authenticated user's profile details
// @Tags         users
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request body domain.UserProfileRequest true "Updated profile details"
// @Success      200 {object} response.Response{data=domain.UserProfile}
// @Failure      400 {object} response.Response
// @Failure      401 {object} response.Response
// @Failure      404 {object} response.Response
// @Router       /users/profile [patch]
func (h *UserHandler) UpdateProfile(c echo.Context) error {
	id, ok := c.Get("user_id").(int32)
	if !ok {
		return response.Error(c, domain.ErrUnauthorized)
	}
	var req domain.UserProfile
	if err := c.Bind(&req); err != nil {
		c.Logger().Errorf("error body data: %v", err)
		return response.ValidationError(c, "invalid request body")
	}
	if err := h.validator.Validate(&req); err != nil {
		c.Logger().Errorf("failed to validate user input: %v", err)
		return response.ValidationError(c, err.Error())
	}
	req.UserId = id
	profile, err := h.userService.UpdateUserProfile(c.Request().Context(), req)
	if err != nil {
		c.Logger().Errorf("failed to update user profile: %v", err)
		return response.Error(c, err)
	}
	return response.Success(c, http.StatusOK, "user profile successfully", profile)
}

// GetProfile godoc
// @Summary      Get user profile
// @Description  Get the currently authenticated user's profile for onboarding tracking
// @Tags         users
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Success      200 {object} response.Response{data=domain.UserProfile}
// @Failure      401 {object} response.Response
// @Failure      404 {object} response.Response
// @Router       /users/profile [get]
func (h *UserHandler) GetProfile(c echo.Context) error {
	id, ok := c.Get("user_id").(int32)
	if !ok {
		return response.Error(c, domain.ErrUnauthorized)
	}

	profile, err := h.userService.GetUserProfile(c.Request().Context(), id)
	if err != nil {
		return response.Error(c, err)
	}

	return response.Success(c, http.StatusOK, "user profile retrieved successfully", profile)
}

// UploadCV godoc
// @Summary      Upload CV
// @Description  Upload a CV file (PDF, max 5MB) for the currently authenticated user
// @Tags         users
// @Accept       multipart/form-data
// @Produce      json
// @Security     BearerAuth
// @Param        cv formData file true "CV file"
// @Success      200 {object} response.Response
// @Failure      400 {object} response.Response
// @Failure      401 {object} response.Response
// @Router       /users/upload/cv [post]
func (h *UserHandler) UploadCV(c echo.Context) error {
	id, ok := c.Get("user_id").(int32)
	if !ok {
		return response.Error(c, domain.ErrUnauthorized)
	}

	fileHeader, err := c.FormFile("cv")
	if err != nil {
		c.Logger().Errorf("failed to get file from form: %v", err)
		return response.Error(c, domain.ErrInvalidInput)
	}

	file, err := fileHeader.Open()
	if err != nil {
		c.Logger().Errorf("failed to open file: %v", err)
		return response.Error(c, domain.ErrInvalidInput)
	}
	defer file.Close()

	data := make([]byte, fileHeader.Size)
	_, err = file.Read(data)
	if err != nil {
		c.Logger().Errorf("failed to read file: %v", err)
		return response.Error(c, domain.ErrInvalidInput)
	}

	err = h.userService.UploadUserCV(c.Request().Context(), id, data)
	if err != nil {
		c.Logger().Errorf("failed to upload CV: %v", err)
		return response.Error(c, err)
	}

	return response.Success(c, http.StatusOK, "file uploaded successfully", nil)
}

// GetCV godoc
// @Summary      Get user CV
// @Description  Download the currently authenticated user's uploaded CV
// @Tags         users
// @Produce      application/pdf
// @Security     BearerAuth
// @Success      200 {file} file "CV contents"
// @Failure      401 {object} response.Response
// @Failure      404 {object} response.Response
// @Router       /users/cv [get]
func (h *UserHandler) GetCV(c echo.Context) error {
	id, ok := c.Get("user_id").(int32)
	if !ok {
		return response.Error(c, domain.ErrUnauthorized)
	}

	cvBytes, err := h.userService.GetUserCV(c.Request().Context(), id)
	if err != nil {
		return response.Error(c, err)
	}

	if len(cvBytes) == 0 {
		return response.Error(c, domain.ErrCVNotFound)
	}

	return c.Blob(http.StatusOK, "application/pdf", cvBytes)
}
