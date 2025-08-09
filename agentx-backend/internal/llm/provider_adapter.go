package llm

import (
	"context"
	"fmt"
	"time"
	
	"github.com/agentx/agentx-backend/internal/providers"
)

// Complete performs a non-streaming completion using the underlying provider
func (a *ProviderAdapter) Complete(ctx context.Context, req *Request) (*Response, error) {
	// Convert unified request to provider request
	providerReq := a.convertRequest(req)
	
	// Call underlying provider
	providerResp, err := a.provider.Complete(ctx, providerReq)
	if err != nil {
		return nil, err
	}
	
	// Convert provider response to unified response
	return a.convertResponse(providerResp), nil
}

// StreamComplete performs a streaming completion using the underlying provider
func (a *ProviderAdapter) StreamComplete(ctx context.Context, req *Request) (<-chan *StreamChunk, error) {
	// Convert unified request to provider request
	providerReq := a.convertRequest(req)
	providerReq.Stream = true
	
	// Call underlying provider
	providerStream, err := a.provider.StreamComplete(ctx, providerReq)
	if err != nil {
		return nil, err
	}
	
	// Create output channel
	out := make(chan *StreamChunk)
	
	// Convert stream
	go func() {
		defer close(out)
		
		for chunk := range providerStream {
			out <- a.convertStreamChunk(chunk)
		}
	}()
	
	return out, nil
}

// GetModels returns available models from the underlying provider
func (a *ProviderAdapter) GetModels(ctx context.Context) ([]ModelInfo, error) {
	providerModels, err := a.provider.GetModels(ctx)
	if err != nil {
		return nil, err
	}
	
	models := make([]ModelInfo, len(providerModels))
	for i, pm := range providerModels {
		models[i] = ModelInfo{
			ID:          pm.ID,
			Provider:    a.config.Type,
			DisplayName: pm.ID, // Use ID as display name since Name doesn't exist
			Description: fmt.Sprintf("%s model", pm.ID),
			MaxTokens:   4096, // Default, since ContextLength doesn't exist
			Status: ModelStatus{
				Available: true,
				Health:    "healthy",
				LastCheck: time.Now(),
			},
		}
	}
	
	return models, nil
}

// HealthCheck checks if the underlying provider is healthy
func (a *ProviderAdapter) HealthCheck(ctx context.Context) error {
	// Try to get models as a health check
	_, err := a.provider.GetModels(ctx)
	return err
}

// GetCapabilities returns provider capabilities
func (a *ProviderAdapter) GetCapabilities() ProviderCapabilities {
	// Determine capabilities based on provider type
	caps := ProviderCapabilities{
		Streaming:       true,
		FunctionCalling: false,
		Vision:          false,
		AudioInput:      false,
		AudioOutput:     false,
		MaxTokens:       4096,
	}
	
	switch a.config.Type {
	case "openai":
		caps.FunctionCalling = true
		caps.Vision = true
		caps.MaxTokens = 128000 // GPT-4 Turbo
		caps.SupportedModels = []string{
			"gpt-4-turbo-preview",
			"gpt-4",
			"gpt-3.5-turbo",
			"gpt-4-vision-preview",
		}
	case "anthropic":
		caps.FunctionCalling = true
		caps.Vision = true
		caps.MaxTokens = 200000 // Claude 3
		caps.SupportedModels = []string{
			"claude-3-opus-20240229",
			"claude-3-sonnet-20240229",
			"claude-3-haiku-20240307",
		}
	case "local":
		// Local provider capabilities depend on the model
		caps.MaxTokens = 8192
		caps.SupportedModels = []string{
			"llama2",
			"codellama",
			"mistral",
			"mixtral",
		}
	}
	
	return caps
}

// Close closes the underlying provider connection
func (a *ProviderAdapter) Close() error {
	// Most providers don't need explicit closing
	// but we can add cleanup here if needed
	return nil
}

// convertRequest converts a unified request to a provider request
func (a *ProviderAdapter) convertRequest(req *Request) providers.CompletionRequest {
	// Convert messages
	messages := make([]providers.Message, len(req.Messages))
	for i, msg := range req.Messages {
		messages[i] = providers.Message{
			Role:    msg.Role,
			Content: msg.Content,
			// Name field doesn't exist in providers.Message
		}
		
		// Convert tool calls if any
		if len(msg.ToolCalls) > 0 {
			messages[i].ToolCalls = make([]providers.ToolCall, len(msg.ToolCalls))
			for j, tc := range msg.ToolCalls {
				messages[i].ToolCalls[j] = providers.ToolCall{
					ID:   tc.ID,
					Type: tc.Type,
					Function: providers.FunctionCall{
						Name:      tc.Function.Name,
						Arguments: tc.Function.Arguments,
					},
				}
			}
		}
	}
	
	providerReq := providers.CompletionRequest{
		Messages: messages,
		Model:    req.Model,
		Stream:   req.Stream,
	}
	
	// Copy generation parameters that exist in providers.CompletionRequest
	if req.Temperature != nil {
		providerReq.Temperature = req.Temperature
	}
	if req.MaxTokens != nil {
		providerReq.MaxTokens = req.MaxTokens
	}
	// Note: TopP, FrequencyPenalty, PresencePenalty, Stop, N don't exist in providers.CompletionRequest
	// These would need to be added to the providers interface if needed
	
	// Convert tools
	if len(req.Tools) > 0 {
		providerReq.Tools = make([]providers.Tool, len(req.Tools))
		for i, tool := range req.Tools {
			providerReq.Tools[i] = providers.Tool{
				Type: tool.Type,
				Function: providers.Function{
					Name:        tool.Function.Name,
					Description: tool.Function.Description,
					Parameters:  tool.Function.Parameters,
				},
			}
		}
	}
	
	return providerReq
}

// convertResponse converts a provider response to a unified response
func (a *ProviderAdapter) convertResponse(resp *providers.CompletionResponse) *Response {
	// Convert choices
	choices := make([]Choice, len(resp.Choices))
	for i, c := range resp.Choices {
		msg := Message{
			Role:    c.Message.Role,
			Content: c.Message.Content,
		}
		
		// Convert tool calls if any
		if len(c.Message.ToolCalls) > 0 {
			msg.ToolCalls = make([]ToolCall, len(c.Message.ToolCalls))
			for j, tc := range c.Message.ToolCalls {
				msg.ToolCalls[j] = ToolCall{
					ID:   tc.ID,
					Type: tc.Type,
					Function: struct {
						Name      string `json:"name"`
						Arguments string `json:"arguments"`
					}{
						Name:      tc.Function.Name,
						Arguments: tc.Function.Arguments,
					},
				}
			}
		}
		
		choices[i] = Choice{
			Index:        c.Index,
			Message:      msg,
			FinishReason: c.FinishReason,
		}
	}
	
	return &Response{
		ID:      resp.ID,
		Object:  resp.Object,
		Created: time.Unix(resp.Created, 0),
		Model:   resp.Model,
		Choices: choices,
		Usage: Usage{
			PromptTokens:     resp.Usage.PromptTokens,
			CompletionTokens: resp.Usage.CompletionTokens,
			TotalTokens:      resp.Usage.TotalTokens,
		},
		Metadata: Metadata{
			Provider: a.config.Type,
			Model:    resp.Model,
		},
	}
}

// convertStreamChunk converts a provider stream chunk to a unified stream chunk
func (a *ProviderAdapter) convertStreamChunk(chunk providers.StreamChunk) *StreamChunk {
	// The provider StreamChunk is simpler - convert to our unified format
	delta := MessageDelta{
		Role:    chunk.Role,
		Content: chunk.Delta,
	}
	
	// Convert tool calls if any
	if len(chunk.ToolCalls) > 0 {
		delta.ToolCalls = make([]ToolCall, len(chunk.ToolCalls))
		for i, tc := range chunk.ToolCalls {
			delta.ToolCalls[i] = ToolCall{
				ID:   tc.ID,
				Type: tc.Type,
				Function: struct {
					Name      string `json:"name"`
					Arguments string `json:"arguments"`
				}{
					Name:      tc.Function.Name,
					Arguments: tc.Function.Arguments,
				},
			}
		}
	}
	
	streamChunk := &StreamChunk{
		ID:      chunk.ID,
		Object:  chunk.Object,
		Created: time.Unix(chunk.Created, 0),
		Model:   chunk.Model,
		Type:    "content", // Set the type for proper filtering
		Content: chunk.Delta, // Set the top-level content
		Choices: []StreamChoice{
			{
				Index:        0,
				Delta:        delta,
				FinishReason: chunk.FinishReason,
			},
		},
	}
	
	// Note: provider StreamChunk doesn't have Usage field
	
	return streamChunk
}