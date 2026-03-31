package handler

import (
	"aiki/internal/domain"
	"aiki/internal/pkg/response"
	"aiki/internal/service"
	"net/http"

	"github.com/labstack/echo/v4"
)

type ChatHandler struct {
	chatService service.ChatService
}

func NewChatHandler(chatService service.ChatService) *ChatHandler {
	return &ChatHandler{chatService: chatService}
}

// Chat godoc
// @Summary      Send a chat message to an AI provider
// @Description  Delegates the conversation to the selected AI provider (e.g. openai, anthropic).
//
//	The caller chooses the provider and can optionally override the model and
//	tune generation parameters (temperature, max_tokens).
//
// @Tags         ai-chat
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body body domain.APIChatRequest true "Chat request"
// @Success      200  {object} response.Response{data=domain.ChatResponse}
// @Failure      400  {object} response.Response
// @Failure      401  {object} response.Response
// @Failure      422  {object} response.Response
// @Failure      500  {object} response.Response
// @Router       /chat [post]
func (h *ChatHandler) Chat(c echo.Context) error {
	_, ok := c.Get("user_id").(int32)
	if !ok {
		return response.Error(c, domain.ErrUnauthorized)
	}

	var req domain.APIChatRequest
	if err := c.Bind(&req); err != nil {
		return response.ValidationError(c, "invalid request body")
	}
	if err := c.Validate(req); err != nil {
		return response.ValidationError(c, err.Error())
	}

	resp, err := h.chatService.Chat(c.Request().Context(), req)
	if err != nil {
		return response.Error(c, err)
	}

	return response.Success(c, http.StatusOK, "chat completed", resp)
}

// GetProviders godoc
// @Summary      List available AI providers
// @Description  Returns the names of all configured AI providers that can be used in /chat.
// @Tags         ai-chat
// @Produce      json
// @Security     BearerAuth
// @Success      200 {object} response.Response{data=[]string}
// @Failure      401 {object} response.Response
// @Router       /chat/providers [get]
func (h *ChatHandler) GetProviders(c echo.Context) error {
	_, ok := c.Get("user_id").(int32)
	if !ok {
		return response.Error(c, domain.ErrUnauthorized)
	}

	return response.Success(c, http.StatusOK, "providers retrieved", h.chatService.AvailableProviders())
}
