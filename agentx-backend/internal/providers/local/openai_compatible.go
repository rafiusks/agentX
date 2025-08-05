package local

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/sashabaranov/go-openai"
	"github.com/agentx/agentx-backend/internal/config"
	"github.com/agentx/agentx-backend/internal/providers"
)

// OpenAICompatibleProvider implements a provider for OpenAI-compatible APIs
type OpenAICompatibleProvider struct {
	id     string
	config config.ProviderConfig
	client *openai.Client
}

// NewOpenAICompatibleProvider creates a new OpenAI-compatible provider
func NewOpenAICompatibleProvider(id string, cfg config.ProviderConfig) (*OpenAICompatibleProvider, error) {
	if cfg.BaseURL == "" {
		return nil, errors.New("base URL is required for OpenAI-compatible provider")
	}

	// Create custom config for the OpenAI client
	apiKey := "dummy-key" // Default for local providers
	if cfg.APIKey != "" {
		apiKey = cfg.APIKey
	}
	
	clientConfig := openai.DefaultConfig(apiKey)
	clientConfig.BaseURL = strings.TrimSuffix(cfg.BaseURL, "/") + "/v1"

	client := openai.NewClientWithConfig(clientConfig)

	return &OpenAICompatibleProvider{
		id:     id,
		config: cfg,
		client: client,
	}, nil
}

// Name returns the provider name
func (p *OpenAICompatibleProvider) Name() string {
	return p.config.Name
}

// Complete performs a non-streaming completion
func (p *OpenAICompatibleProvider) Complete(ctx context.Context, req providers.CompletionRequest) (*providers.CompletionResponse, error) {
	openAIReq := p.convertRequest(req)
	openAIReq.Stream = false

	resp, err := p.client.CreateChatCompletion(ctx, openAIReq)
	if err != nil {
		return nil, err
	}

	return p.convertResponse(&resp), nil
}

// StreamComplete performs a streaming completion
func (p *OpenAICompatibleProvider) StreamComplete(ctx context.Context, req providers.CompletionRequest) (<-chan providers.StreamChunk, error) {
	chunks := make(chan providers.StreamChunk)

	go func() {
		defer close(chunks)

		openAIReq := p.convertRequest(req)
		openAIReq.Stream = true

		stream, err := p.client.CreateChatCompletionStream(ctx, openAIReq)
		if err != nil {
			chunks <- providers.StreamChunk{Error: err.Error()}
			return
		}
		defer stream.Close()

		for {
			response, err := stream.Recv()
			if errors.Is(err, io.EOF) {
				chunks <- providers.StreamChunk{FinishReason: "stop"}
				return
			}
			if err != nil {
				chunks <- providers.StreamChunk{Error: err.Error()}
				return
			}

			if len(response.Choices) > 0 {
				choice := response.Choices[0]
				chunk := providers.StreamChunk{
					ID:      response.ID,
					Object:  response.Object,
					Created: response.Created,
					Model:   response.Model,
				}

				if choice.Delta.Content != "" {
					chunk.Delta = choice.Delta.Content
				}

				if choice.Delta.Role != "" {
					chunk.Role = choice.Delta.Role
				}

				if choice.Delta.FunctionCall != nil {
					chunk.FunctionCall = &providers.FunctionCall{
						Name:      choice.Delta.FunctionCall.Name,
						Arguments: choice.Delta.FunctionCall.Arguments,
					}
				}

				if len(choice.Delta.ToolCalls) > 0 {
					chunk.ToolCalls = p.convertToolCalls(choice.Delta.ToolCalls)
				}

				if choice.FinishReason != "" {
					chunk.FinishReason = string(choice.FinishReason)
				}

				chunks <- chunk
			}
		}
	}()

	return chunks, nil
}

// GetModels returns available models from the provider
func (p *OpenAICompatibleProvider) GetModels(ctx context.Context) ([]providers.Model, error) {
	// Try to get models from the /v1/models endpoint
	url := strings.TrimSuffix(p.config.BaseURL, "/") + "/v1/models"
	
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	if p.config.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+p.config.APIKey)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		// If we can't reach the models endpoint, return configured models
		if len(p.config.Models) > 0 {
			models := make([]providers.Model, len(p.config.Models))
			for i, modelID := range p.config.Models {
				models[i] = providers.Model{
					ID:      modelID,
					Object:  "model",
					OwnedBy: p.config.Name,
				}
			}
			return models, nil
		}
		return nil, fmt.Errorf("failed to discover models: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// Return configured models as fallback
		if len(p.config.Models) > 0 {
			models := make([]providers.Model, len(p.config.Models))
			for i, modelID := range p.config.Models {
				models[i] = providers.Model{
					ID:      modelID,
					Object:  "model",
					OwnedBy: p.config.Name,
				}
			}
			return models, nil
		}
		return nil, fmt.Errorf("models endpoint returned status %d", resp.StatusCode)
	}

	var modelsResp struct {
		Data []providers.Model `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&modelsResp); err != nil {
		return nil, err
	}

	return modelsResp.Data, nil
}

// ValidateConfig validates the provider configuration
func (p *OpenAICompatibleProvider) ValidateConfig() error {
	if p.config.BaseURL == "" {
		return errors.New("base URL is required")
	}
	return nil
}

// convertRequest converts internal request to OpenAI request
func (p *OpenAICompatibleProvider) convertRequest(req providers.CompletionRequest) openai.ChatCompletionRequest {
	messages := make([]openai.ChatCompletionMessage, len(req.Messages))
	for i, msg := range req.Messages {
		messages[i] = openai.ChatCompletionMessage{
			Role:    msg.Role,
			Content: msg.Content,
		}

		if msg.FunctionCall != nil {
			messages[i].FunctionCall = &openai.FunctionCall{
				Name:      msg.FunctionCall.Name,
				Arguments: msg.FunctionCall.Arguments,
			}
		}

		if len(msg.ToolCalls) > 0 {
			messages[i].ToolCalls = make([]openai.ToolCall, len(msg.ToolCalls))
			for j, tc := range msg.ToolCalls {
				messages[i].ToolCalls[j] = openai.ToolCall{
					ID:   tc.ID,
					Type: openai.ToolType(tc.Type),
					Function: openai.FunctionCall{
						Name:      tc.Function.Name,
						Arguments: tc.Function.Arguments,
					},
				}
			}
		}

		if msg.ToolCallID != "" {
			messages[i].ToolCallID = msg.ToolCallID
		}
	}

	openAIReq := openai.ChatCompletionRequest{
		Model:    req.Model,
		Messages: messages,
		Stream:   req.Stream,
	}

	if req.Temperature != nil {
		openAIReq.Temperature = *req.Temperature
	}

	if req.MaxTokens != nil {
		openAIReq.MaxTokens = *req.MaxTokens
	}

	// Convert functions to tools format
	if len(req.Functions) > 0 {
		openAIReq.Tools = make([]openai.Tool, len(req.Functions))
		for i, fn := range req.Functions {
			openAIReq.Tools[i] = openai.Tool{
				Type: openai.ToolTypeFunction,
				Function: &openai.FunctionDefinition{
					Name:        fn.Name,
					Description: fn.Description,
					Parameters:  fn.Parameters,
				},
			}
		}
	}

	// Add tools if provided
	if len(req.Tools) > 0 {
		for _, tool := range req.Tools {
			openAIReq.Tools = append(openAIReq.Tools, openai.Tool{
				Type: openai.ToolTypeFunction,
				Function: &openai.FunctionDefinition{
					Name:        tool.Function.Name,
					Description: tool.Function.Description,
					Parameters:  tool.Function.Parameters,
				},
			})
		}
	}

	// Convert tool choice
	if req.ToolChoice != nil {
		switch req.ToolChoice.Type {
		case "auto":
			openAIReq.ToolChoice = "auto"
		case "none":
			openAIReq.ToolChoice = "none"
		case "function":
			if req.ToolChoice.Function != nil {
				openAIReq.ToolChoice = openai.ToolChoice{
					Type: openai.ToolTypeFunction,
					Function: openai.ToolFunction{
						Name: req.ToolChoice.Function.Name,
					},
				}
			}
		}
	}

	return openAIReq
}

// convertResponse converts OpenAI response to internal response
func (p *OpenAICompatibleProvider) convertResponse(resp *openai.ChatCompletionResponse) *providers.CompletionResponse {
	choices := make([]providers.Choice, len(resp.Choices))
	for i, choice := range resp.Choices {
		msg := providers.Message{
			Role:    choice.Message.Role,
			Content: choice.Message.Content,
		}

		if choice.Message.FunctionCall != nil {
			msg.FunctionCall = &providers.FunctionCall{
				Name:      choice.Message.FunctionCall.Name,
				Arguments: choice.Message.FunctionCall.Arguments,
			}
		}

		if len(choice.Message.ToolCalls) > 0 {
			msg.ToolCalls = p.convertToolCalls(choice.Message.ToolCalls)
		}

		choices[i] = providers.Choice{
			Index:        choice.Index,
			Message:      msg,
			FinishReason: string(choice.FinishReason),
		}
	}

	return &providers.CompletionResponse{
		ID:      resp.ID,
		Object:  resp.Object,
		Created: resp.Created,
		Model:   resp.Model,
		Choices: choices,
		Usage: providers.Usage{
			PromptTokens:     resp.Usage.PromptTokens,
			CompletionTokens: resp.Usage.CompletionTokens,
			TotalTokens:      resp.Usage.TotalTokens,
		},
		SystemFingerprint: resp.SystemFingerprint,
	}
}

// convertToolCalls converts OpenAI tool calls to internal format
func (p *OpenAICompatibleProvider) convertToolCalls(toolCalls []openai.ToolCall) []providers.ToolCall {
	result := make([]providers.ToolCall, len(toolCalls))
	for i, tc := range toolCalls {
		result[i] = providers.ToolCall{
			ID:   tc.ID,
			Type: string(tc.Type),
			Function: providers.FunctionCall{
				Name:      tc.Function.Name,
				Arguments: tc.Function.Arguments,
			},
		}
	}
	return result
}