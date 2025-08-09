package llm

import (
	"context"
	"fmt"
	"time"
)

// StubProvider is a temporary stub implementation for testing
type StubProvider struct {
	providerType string
	config       ProviderConfig
}

// Complete performs a non-streaming completion
func (s *StubProvider) Complete(ctx context.Context, req *Request) (*Response, error) {
	return &Response{
		ID:      fmt.Sprintf("stub-%d", time.Now().Unix()),
		Object:  "chat.completion",
		Created: time.Now(),
		Model:   req.Model,
		Choices: []Choice{
			{
				Index: 0,
				Message: Message{
					Role:    "assistant",
					Content: "This is a stub response from the " + s.providerType + " provider",
				},
				FinishReason: "stop",
			},
		},
		Usage: Usage{
			PromptTokens:     10,
			CompletionTokens: 10,
			TotalTokens:      20,
		},
		Metadata: Metadata{
			Provider: s.providerType,
			Model:    req.Model,
		},
	}, nil
}

// StreamComplete performs a streaming completion
func (s *StubProvider) StreamComplete(ctx context.Context, req *Request) (<-chan *StreamChunk, error) {
	out := make(chan *StreamChunk)
	
	go func() {
		defer close(out)
		
		// Send a few chunks
		chunks := []string{"This ", "is ", "a ", "stub ", "streaming ", "response."}
		for i, chunk := range chunks {
			select {
			case out <- &StreamChunk{
				ID:      fmt.Sprintf("stub-%d", time.Now().Unix()),
				Object:  "chat.completion.chunk",
				Created: time.Now(),
				Model:   req.Model,
				Choices: []StreamChoice{
					{
						Index: i,
						Delta: MessageDelta{
							Content: chunk,
						},
					},
				},
			}:
			case <-ctx.Done():
				return
			}
			time.Sleep(100 * time.Millisecond)
		}
		
		// Send final chunk with finish reason
		out <- &StreamChunk{
			ID:      fmt.Sprintf("stub-%d", time.Now().Unix()),
			Object:  "chat.completion.chunk",
			Created: time.Now(),
			Model:   req.Model,
			Choices: []StreamChoice{
				{
					Index:        0,
					Delta:        MessageDelta{},
					FinishReason: "stop",
				},
			},
		}
	}()
	
	return out, nil
}

// GetModels returns available models
func (s *StubProvider) GetModels(ctx context.Context) ([]ModelInfo, error) {
	models := []ModelInfo{}
	
	switch s.providerType {
	case "openai":
		models = []ModelInfo{
			{ID: "gpt-3.5-turbo", DisplayName: "GPT-3.5 Turbo"},
			{ID: "gpt-4", DisplayName: "GPT-4"},
		}
	case "anthropic":
		models = []ModelInfo{
			{ID: "claude-3-opus", DisplayName: "Claude 3 Opus"},
			{ID: "claude-3-sonnet", DisplayName: "Claude 3 Sonnet"},
		}
	case "local":
		models = []ModelInfo{
			{ID: "llama2", DisplayName: "Llama 2"},
			{ID: "mistral", DisplayName: "Mistral"},
		}
	}
	
	for i := range models {
		models[i].Provider = s.providerType
		models[i].Status = ModelStatus{
			Available: true,
			Health:    "healthy",
			LastCheck: time.Now(),
		}
	}
	
	return models, nil
}

// HealthCheck checks if the provider is healthy
func (s *StubProvider) HealthCheck(ctx context.Context) error {
	// Always healthy for stub
	return nil
}

// GetCapabilities returns provider capabilities
func (s *StubProvider) GetCapabilities() ProviderCapabilities {
	return ProviderCapabilities{
		Streaming:       true,
		FunctionCalling: s.providerType != "local",
		Vision:          s.providerType != "local",
		MaxTokens:       4096,
		SupportedModels: []string{},
	}
}

// Close closes the provider connection
func (s *StubProvider) Close() error {
	// Nothing to close for stub
	return nil
}