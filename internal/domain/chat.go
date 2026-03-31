package domain

import "errors"

var (
	ErrProviderNotFound      = errors.New("ai provider not found")
	ErrProviderNotConfigured = errors.New("ai provider is not configured (missing api key)")
	ErrEmptyMessages         = errors.New("messages cannot be empty")
)

// ChatMessage represents a single message in a conversation.
type ChatMessage struct {
	Role    string `json:"role"    validate:"required,oneof=user assistant system"`
	Content string `json:"content" validate:"required"`
}

// ChatModelConfig holds optional per-request model tuning parameters.
type ChatModelConfig struct {
	Temperature *float64 `json:"temperature,omitempty"`
	MaxTokens   *int     `json:"max_tokens,omitempty"`
}

// APIChatRequest is the inbound HTTP request body for POST /chat.
type APIChatRequest struct {
	// Provider selects which AI backend to use (e.g. "openai", "anthropic").
	Provider string `json:"provider" validate:"required"`
	// Model overrides the provider's default model (optional).
	Model    string          `json:"model,omitempty"`
	Messages []ChatMessage   `json:"messages" validate:"required,min=1,dive"`
	Config   ChatModelConfig `json:"config,omitempty"`
}

// ChatUsage reports token consumption for a request.
type ChatUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// ChatResponse is returned by the chat service and handler.
type ChatResponse struct {
	Provider string      `json:"provider"`
	Model    string      `json:"model"`
	Message  ChatMessage `json:"message"`
	Usage    *ChatUsage  `json:"usage,omitempty"`
}
