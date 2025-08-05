package openai

import (
	"context"
	"errors"
	"io"

	"github.com/sashabaranov/go-openai"
	"github.com/agentx/agentx-backend/internal/config"
	"github.com/agentx/agentx-backend/internal/providers"
)

// Provider implements the OpenAI provider
type Provider struct {
	id     string
	config config.ProviderConfig
	client *openai.Client
}

// NewProvider creates a new OpenAI provider
func NewProvider(id string, cfg config.ProviderConfig) (*Provider, error) {
	if cfg.APIKey == "" {
		return nil, errors.New("OpenAI API key is required")
	}

	client := openai.NewClient(cfg.APIKey)
	
	return &Provider{
		id:     id,
		config: cfg,
		client: client,
	}, nil
}

// Name returns the provider name
func (p *Provider) Name() string {
	return p.config.Name
}

// Complete performs a non-streaming completion
func (p *Provider) Complete(ctx context.Context, req providers.CompletionRequest) (*providers.CompletionResponse, error) {
	openAIReq := p.convertRequest(req)
	openAIReq.Stream = false

	resp, err := p.client.CreateChatCompletion(ctx, openAIReq)
	if err != nil {
		return nil, err
	}

	return p.convertResponse(&resp), nil
}

// StreamComplete performs a streaming completion
func (p *Provider) StreamComplete(ctx context.Context, req providers.CompletionRequest) (<-chan providers.StreamChunk, error) {
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

// GetModels returns available models
func (p *Provider) GetModels(ctx context.Context) ([]providers.Model, error) {
	modelList, err := p.client.ListModels(ctx)
	if err != nil {
		return nil, err
	}

	models := make([]providers.Model, len(modelList.Models))
	for i, m := range modelList.Models {
		models[i] = providers.Model{
			ID:      m.ID,
			Object:  m.Object,
			Created: 0, // OpenAI Go client doesn't expose Created field
			OwnedBy: m.OwnedBy,
		}
	}

	return models, nil
}

// ValidateConfig validates the provider configuration
func (p *Provider) ValidateConfig() error {
	if p.config.APIKey == "" {
		return errors.New("API key is required")
	}
	return nil
}

// convertRequest converts internal request to OpenAI request
func (p *Provider) convertRequest(req providers.CompletionRequest) openai.ChatCompletionRequest {
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

	// Convert functions to tools format (OpenAI prefers tools over functions now)
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
func (p *Provider) convertResponse(resp *openai.ChatCompletionResponse) *providers.CompletionResponse {
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
func (p *Provider) convertToolCalls(toolCalls []openai.ToolCall) []providers.ToolCall {
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