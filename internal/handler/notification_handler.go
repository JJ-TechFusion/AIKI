package handler

import (
	"aiki/internal/domain"
	"aiki/internal/pkg/response"
	"aiki/internal/service"
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"
)

type NotificationHandler struct {
	notifService service.NotificationService
}

func NewNotificationHandler(notifService service.NotificationService) *NotificationHandler {
	return &NotificationHandler{notifService: notifService}
}

// GetNotifications godoc
// @Summary      Get notifications
// @Description  Returns paginated notifications and unread count for the authenticated user
// @Tags         notifications
// @Produce      json
// @Security     BearerAuth
// @Param        limit  query int false "Limit (default 20)"
// @Param        offset query int false "Offset (default 0)"
// @Success      200 {object} response.Response{data=domain.NotificationSummary}
// @Failure      401 {object} response.Response
// @Router       /notifications [get]
func (h *NotificationHandler) GetNotifications(c echo.Context) error {
	userID, ok := c.Get("user_id").(int32)
	if !ok {
		return response.Error(c, domain.ErrUnauthorized)
	}

	limit := int32(20)
	offset := int32(0)
	if l := c.QueryParam("limit"); l != "" {
		if v, err := strconv.Atoi(l); err == nil {
			limit = int32(v)
		}
	}
	if o := c.QueryParam("offset"); o != "" {
		if v, err := strconv.Atoi(o); err == nil {
			offset = int32(v)
		}
	}

	summary, err := h.notifService.GetNotifications(c.Request().Context(), userID, limit, offset)
	if err != nil {
		return response.Error(c, err)
	}

	return response.Success(c, http.StatusOK, "notifications retrieved", summary)
}

// GetUnreadCount godoc
// @Summary      Get unread notification count
// @Description  Returns the number of unread notifications for the home screen badge
// @Tags         notifications
// @Produce      json
// @Security     BearerAuth
// @Success      200 {object} response.Response{data=int32}
// @Failure      401 {object} response.Response
// @Router       /notifications/unread-count [get]
func (h *NotificationHandler) GetUnreadCount(c echo.Context) error {
	userID, ok := c.Get("user_id").(int32)
	if !ok {
		return response.Error(c, domain.ErrUnauthorized)
	}

	count, err := h.notifService.GetUnreadCount(c.Request().Context(), userID)
	if err != nil {
		return response.Error(c, err)
	}

	return response.Success(c, http.StatusOK, "unread count retrieved", count)
}

// MarkRead godoc
// @Summary      Mark a notification as read
// @Tags         notifications
// @Produce      json
// @Security     BearerAuth
// @Param        id path int true "Notification ID"
// @Success      200 {object} response.Response
// @Failure      401 {object} response.Response
// @Router       /notifications/{id}/read [patch]
func (h *NotificationHandler) MarkRead(c echo.Context) error {
	userID, ok := c.Get("user_id").(int32)
	if !ok {
		return response.Error(c, domain.ErrUnauthorized)
	}

	notifID, err := parseIDParam(c, "id")
	if err != nil {
		return response.ValidationError(c, "invalid notification id")
	}

	if err := h.notifService.MarkRead(c.Request().Context(), notifID, userID); err != nil {
		return response.Error(c, err)
	}

	return response.Success(c, http.StatusOK, "notification marked as read", nil)
}

// MarkAllRead godoc
// @Summary      Mark all notifications as read
// @Tags         notifications
// @Produce      json
// @Security     BearerAuth
// @Success      200 {object} response.Response
// @Failure      401 {object} response.Response
// @Router       /notifications/read-all [patch]
func (h *NotificationHandler) MarkAllRead(c echo.Context) error {
	userID, ok := c.Get("user_id").(int32)
	if !ok {
		return response.Error(c, domain.ErrUnauthorized)
	}

	if err := h.notifService.MarkAllRead(c.Request().Context(), userID); err != nil {
		return response.Error(c, err)
	}

	return response.Success(c, http.StatusOK, "all notifications marked as read", nil)
}

// DeleteNotification godoc
// @Summary      Delete a notification
// @Tags         notifications
// @Produce      json
// @Security     BearerAuth
// @Param        id path int true "Notification ID"
// @Success      200 {object} response.Response
// @Failure      401 {object} response.Response
// @Router       /notifications/{id} [delete]
func (h *NotificationHandler) DeleteNotification(c echo.Context) error {
	userID, ok := c.Get("user_id").(int32)
	if !ok {
		return response.Error(c, domain.ErrUnauthorized)
	}

	notifID, err := parseIDParam(c, "id")
	if err != nil {
		return response.ValidationError(c, "invalid notification id")
	}

	if err := h.notifService.Delete(c.Request().Context(), notifID, userID); err != nil {
		return response.Error(c, err)
	}

	return response.Success(c, http.StatusOK, "notification deleted", nil)
}

// GetPreferences godoc
// @Summary      Get notification preferences
// @Description  Returns notification channel and category preferences for the authenticated user
// @Tags         notifications
// @Produce      json
// @Security     BearerAuth
// @Success      200 {object} response.Response{data=domain.NotificationPreferences}
// @Failure      401 {object} response.Response
// @Router       /notifications/preferences [get]
func (h *NotificationHandler) GetPreferences(c echo.Context) error {
	userID, ok := c.Get("user_id").(int32)
	if !ok {
		return response.Error(c, domain.ErrUnauthorized)
	}

	prefs, err := h.notifService.GetPreferences(c.Request().Context(), userID)
	if err != nil {
		return response.Error(c, err)
	}

	return response.Success(c, http.StatusOK, "notification preferences retrieved", prefs)
}

// UpdatePreferences godoc
// @Summary      Update notification preferences
// @Description  Replaces notification channel and category preferences for the authenticated user
// @Tags         notifications
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request body domain.UpdateNotificationPreferencesRequest true "Notification preferences"
// @Success      200 {object} response.Response{data=domain.NotificationPreferences}
// @Failure      400 {object} response.Response
// @Failure      401 {object} response.Response
// @Router       /notifications/preferences [put]
func (h *NotificationHandler) UpdatePreferences(c echo.Context) error {
	userID, ok := c.Get("user_id").(int32)
	if !ok {
		return response.Error(c, domain.ErrUnauthorized)
	}

	var req domain.UpdateNotificationPreferencesRequest
	if err := c.Bind(&req); err != nil {
		return response.ValidationError(c, "invalid request body")
	}

	prefs, err := h.notifService.UpdatePreferences(c.Request().Context(), userID, &req)
	if err != nil {
		return response.Error(c, err)
	}

	return response.Success(c, http.StatusOK, "notification preferences updated", prefs)
}
