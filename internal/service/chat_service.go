package service

import (
	"aiki/internal/ai"
	"aiki/internal/domain"
	"context"
	"fmt"
)

// ChatService dispatches chat requests to the appropriate AI provider.
type ChatService interface {
	Chat(ctx context.Context, req domain.APIChatRequest) (*domain.ChatResponse, error)
	AvailableProviders() []string
}

type chatService struct {
	registry *ai.Registry
}

func NewChatService(registry *ai.Registry) ChatService {
	return &chatService{registry: registry}
}

func (s *chatService) Chat(ctx context.Context, req domain.APIChatRequest) (*domain.ChatResponse, error) {
	if len(req.Messages) == 0 {
		return nil, domain.ErrEmptyMessages
	}

	provider, ok := s.registry.Get(req.Provider)
	if !ok {
		return nil, fmt.Errorf("%w: %q (available: %v)", domain.ErrProviderNotFound, req.Provider, s.registry.Names())
	}

	// Map domain messages → ai layer messages
	msgs := make([]ai.Message, len(req.Messages))
	for i, m := range req.Messages {
		msgs[i] = ai.Message{Role: m.Role, Content: m.Content}
	}

	aiReq := ai.ChatRequest{
		Model:    req.Model,
		Messages: msgs,
		Config: ai.ChatConfig{
			Temperature: req.Config.Temperature,
			MaxTokens:   req.Config.MaxTokens,
		},
	}

	aiResp, err := provider.Chat(ctx, aiReq)
	if err != nil {
		return nil, err
	}

	resp := &domain.ChatResponse{
		Provider: aiResp.Provider,
		Model:    aiResp.Model,
		Message: domain.ChatMessage{
			Role:    aiResp.Message.Role,
			Content: aiResp.Message.Content,
		},
	}
	if aiResp.Usage != nil {
		resp.Usage = &domain.ChatUsage{
			PromptTokens:     aiResp.Usage.PromptTokens,
			CompletionTokens: aiResp.Usage.CompletionTokens,
			TotalTokens:      aiResp.Usage.TotalTokens,
		}
	}

	return resp, nil
}

func (s *chatService) AvailableProviders() []string {
	return s.registry.Names()
}
