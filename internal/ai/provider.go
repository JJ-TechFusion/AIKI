// Package ai defines the Provider interface that every AI backend must satisfy,
// along with the shared request/response types used across providers.
package ai

import "context"

// Message is a single turn in a conversation.
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ChatConfig holds optional parameters that tune model behaviour.
// All fields are optional; unset fields use the provider's defaults.
type ChatConfig struct {
	Temperature *float64
	MaxTokens   *int
}

// ChatRequest is what a Provider receives.
type ChatRequest struct {
	Model    string
	Messages []Message
	Config   ChatConfig
}

// Usage reports token consumption.
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// ChatResponse is what every Provider returns on success.
type ChatResponse struct {
	Message  Message
	Model    string
	Provider string
	Usage    *Usage
}

// Provider is the interface every AI backend must implement.
// Adding a new AI provider means implementing this interface and
// registering the implementation with the Registry.
type Provider interface {
	// Name returns the unique identifier for this provider (e.g. "openai").
	Name() string
	// DefaultModel returns the model used when the caller does not specify one.
	DefaultModel() string
	// Chat sends the conversation to the provider and returns the reply.
	Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error)
}
