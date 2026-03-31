// Package openai implements the ai.Provider interface for the OpenAI Chat Completions API.
package openai

import (
	"aiki/internal/ai"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

const (
	apiURL       = "https://api.openai.com/v1/chat/completions"
	providerName = "openai"
)

// Provider satisfies ai.Provider using the OpenAI Chat Completions API.
type Provider struct {
	apiKey       string
	defaultModel string
	httpClient   *http.Client
}

// New creates an OpenAI provider.
// defaultModel is used when the caller does not specify a model (e.g. "gpt-4o-mini").
func New(apiKey, defaultModel string) *Provider {
	return &Provider{
		apiKey:       apiKey,
		defaultModel: defaultModel,
		httpClient:   &http.Client{Timeout: 60 * time.Second},
	}
}

func (p *Provider) Name() string         { return providerName }
func (p *Provider) DefaultModel() string { return p.defaultModel }

// Chat sends the conversation to OpenAI and returns the assistant reply.
func (p *Provider) Chat(ctx context.Context, req ai.ChatRequest) (*ai.ChatResponse, error) {
	if p.apiKey == "" {
		return nil, fmt.Errorf("openai: api key not configured")
	}

	model := req.Model
	if model == "" {
		model = p.defaultModel
	}

	// Build request body
	body := openAIRequest{
		Model:    model,
		Messages: toOpenAIMessages(req.Messages),
	}
	if req.Config.Temperature != nil {
		body.Temperature = req.Config.Temperature
	}
	if req.Config.MaxTokens != nil {
		body.MaxTokens = req.Config.MaxTokens
	}

	payload, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("openai: failed to encode request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("openai: failed to build request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)

	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("openai: request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var apiErr openAIError
		_ = json.NewDecoder(resp.Body).Decode(&apiErr)
		return nil, fmt.Errorf("openai: api error %d: %s", resp.StatusCode, apiErr.Error.Message)
	}

	var result openAIResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("openai: failed to decode response: %w", err)
	}
	if len(result.Choices) == 0 {
		return nil, fmt.Errorf("openai: no choices returned")
	}

	choice := result.Choices[0]
	return &ai.ChatResponse{
		Provider: providerName,
		Model:    result.Model,
		Message: ai.Message{
			Role:    choice.Message.Role,
			Content: choice.Message.Content,
		},
		Usage: &ai.Usage{
			PromptTokens:     result.Usage.PromptTokens,
			CompletionTokens: result.Usage.CompletionTokens,
			TotalTokens:      result.Usage.TotalTokens,
		},
	}, nil
}

// ── internal wire types ───────────────────────────────────────────────────────

type openAIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openAIRequest struct {
	Model       string          `json:"model"`
	Messages    []openAIMessage `json:"messages"`
	Temperature *float64        `json:"temperature,omitempty"`
	MaxTokens   *int            `json:"max_tokens,omitempty"`
}

type openAIResponse struct {
	Model   string `json:"model"`
	Choices []struct {
		Message openAIMessage `json:"message"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

type openAIError struct {
	Error struct {
		Message string `json:"message"`
	} `json:"error"`
}

func toOpenAIMessages(msgs []ai.Message) []openAIMessage {
	out := make([]openAIMessage, len(msgs))
	for i, m := range msgs {
		out[i] = openAIMessage{Role: m.Role, Content: m.Content}
	}
	return out
}
