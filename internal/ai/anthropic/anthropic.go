// Package anthropic implements the ai.Provider interface for the Anthropic Messages API.
package anthropic

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
	apiURL           = "https://api.anthropic.com/v1/messages"
	anthropicVersion = "2023-06-01"
	providerName     = "anthropic"
	defaultMaxTokens = 1024
)

// Provider satisfies ai.Provider using the Anthropic Messages API.
type Provider struct {
	apiKey       string
	defaultModel string
	httpClient   *http.Client
}

// New creates an Anthropic provider.
// defaultModel is used when the caller does not specify a model (e.g. "claude-haiku-4-5-20251001").
func New(apiKey, defaultModel string) *Provider {
	return &Provider{
		apiKey:       apiKey,
		defaultModel: defaultModel,
		httpClient:   &http.Client{Timeout: 60 * time.Second},
	}
}

func (p *Provider) Name() string         { return providerName }
func (p *Provider) DefaultModel() string { return p.defaultModel }

// Chat sends the conversation to Anthropic and returns the assistant reply.
func (p *Provider) Chat(ctx context.Context, req ai.ChatRequest) (*ai.ChatResponse, error) {
	if p.apiKey == "" {
		return nil, fmt.Errorf("anthropic: api key not configured")
	}

	model := req.Model
	if model == "" {
		model = p.defaultModel
	}

	// Anthropic separates system messages from the conversation turns.
	var systemPrompt string
	userMessages := make([]anthropicMessage, 0, len(req.Messages))
	for _, m := range req.Messages {
		if m.Role == "system" {
			systemPrompt = m.Content
		} else {
			userMessages = append(userMessages, anthropicMessage{
				Role:    m.Role,
				Content: m.Content,
			})
		}
	}

	maxTokens := defaultMaxTokens
	if req.Config.MaxTokens != nil {
		maxTokens = *req.Config.MaxTokens
	}

	body := anthropicRequest{
		Model:     model,
		MaxTokens: maxTokens,
		Messages:  userMessages,
	}
	if systemPrompt != "" {
		body.System = systemPrompt
	}
	if req.Config.Temperature != nil {
		body.Temperature = req.Config.Temperature
	}

	payload, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("anthropic: failed to encode request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("anthropic: failed to build request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", p.apiKey)
	httpReq.Header.Set("anthropic-version", anthropicVersion)

	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("anthropic: request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var apiErr anthropicError
		_ = json.NewDecoder(resp.Body).Decode(&apiErr)
		return nil, fmt.Errorf("anthropic: api error %d: %s", resp.StatusCode, apiErr.Error.Message)
	}

	var result anthropicResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("anthropic: failed to decode response: %w", err)
	}
	if len(result.Content) == 0 {
		return nil, fmt.Errorf("anthropic: empty content in response")
	}

	return &ai.ChatResponse{
		Provider: providerName,
		Model:    result.Model,
		Message: ai.Message{
			Role:    result.Role,
			Content: result.Content[0].Text,
		},
		Usage: &ai.Usage{
			PromptTokens:     result.Usage.InputTokens,
			CompletionTokens: result.Usage.OutputTokens,
			TotalTokens:      result.Usage.InputTokens + result.Usage.OutputTokens,
		},
	}, nil
}

// ── internal wire types ───────────────────────────────────────────────────────

type anthropicMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type anthropicRequest struct {
	Model       string             `json:"model"`
	MaxTokens   int                `json:"max_tokens"`
	Messages    []anthropicMessage `json:"messages"`
	System      string             `json:"system,omitempty"`
	Temperature *float64           `json:"temperature,omitempty"`
}

type anthropicResponse struct {
	Model   string `json:"model"`
	Role    string `json:"role"`
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	Usage struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
}

type anthropicError struct {
	Error struct {
		Type    string `json:"type"`
		Message string `json:"message"`
	} `json:"error"`
}
