package adapters

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/agentx/agentx-backend/internal/api/models"
	"github.com/agentx/agentx-backend/internal/providers"
)

// Adapter interface for normalizing provider-specific formats
type Adapter interface {
	// NormalizeRequest converts unified request to provider-specific format
	NormalizeRequest(req models.UnifiedChatRequest) (providers.CompletionRequest, error)
	
	// NormalizeResponse converts provider response to unified format
	NormalizeResponse(resp *providers.CompletionResponse) (*models.UnifiedChatResponse, error)
	
	// NormalizeStreamChunk converts provider stream chunk to unified format
	NormalizeStreamChunk(chunk providers.StreamChunk) (*models.UnifiedStreamChunk, error)
	
	// NormalizeError converts provider errors to unified format
	NormalizeError(err error, provider string) *models.UnifiedError
	
	// GetProviderName returns the provider this adapter is for
	GetProviderName() string
}

// BaseAdapter provides common functionality for all adapters
type BaseAdapter struct {
	providerName string
}

// NewBaseAdapter creates a new base adapter
func NewBaseAdapter(providerName string) BaseAdapter {
	return BaseAdapter{providerName: providerName}
}

// GetProviderName returns the provider name
func (a BaseAdapter) GetProviderName() string {
	return a.providerName
}

// NormalizeRequest provides base implementation
func (a BaseAdapter) NormalizeRequest(req models.UnifiedChatRequest) (providers.CompletionRequest, error) {
	providerReq := providers.CompletionRequest{
		Messages:     req.Messages,
		Temperature:  req.Temperature,
		MaxTokens:    req.MaxTokens,
		Functions:    req.Functions,
		Tools:        req.Tools,
	}
	
	// Handle response format conversion
	if req.ResponseFormat != "" {
		switch req.ResponseFormat {
		case "json":
			providerReq.ResponseFormat = &providers.ResponseFormat{Type: "json_object"}
		case "text", "markdown", "code":
			// Most providers handle these as regular text
			providerReq.ResponseFormat = &providers.ResponseFormat{Type: "text"}
		}
	}
	
	return providerReq, nil
}

// NormalizeResponse provides base implementation
func (a BaseAdapter) NormalizeResponse(resp *providers.CompletionResponse) (*models.UnifiedChatResponse, error) {
	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("no choices in response")
	}
	
	choice := resp.Choices[0]
	
	unifiedResp := &models.UnifiedChatResponse{
		ID:      resp.ID,
		Content: choice.Message.Content,
		Role:    choice.Message.Role,
		Usage: models.Usage{
			PromptTokens:     resp.Usage.PromptTokens,
			CompletionTokens: resp.Usage.CompletionTokens,
			TotalTokens:      resp.Usage.TotalTokens,
		},
	}
	
	// Convert function/tool calls
	if len(choice.Message.FunctionCall.Name) > 0 {
		unifiedResp.Functions = []models.FunctionResponse{
			{
				Name:      choice.Message.FunctionCall.Name,
				Arguments: choice.Message.FunctionCall.Arguments,
			},
		}
	}
	
	for _, toolCall := range choice.Message.ToolCalls {
		unifiedResp.Tools = append(unifiedResp.Tools, models.ToolResponse{
			ID:   toolCall.ID,
			Type: toolCall.Type,
			Function: models.FunctionResponse{
				Name:      toolCall.Function.Name,
				Arguments: toolCall.Function.Arguments,
			},
		})
	}
	
	return unifiedResp, nil
}

// NormalizeStreamChunk provides base implementation
func (a BaseAdapter) NormalizeStreamChunk(chunk providers.StreamChunk) (*models.UnifiedStreamChunk, error) {
	unified := &models.UnifiedStreamChunk{}
	
	// Determine chunk type
	if chunk.Error != "" {
		unified.Type = "error"
		unified.Error = &models.UnifiedError{
			Message: chunk.Error,
			Type:    models.ErrorTypeProvider,
		}
	} else if chunk.FinishReason != "" {
		unified.Type = "done"
	} else if len(chunk.ToolCalls) > 0 {
		unified.Type = "tool_use"
		// Convert first tool call (streaming usually sends one at a time)
		if len(chunk.ToolCalls) > 0 {
			tc := chunk.ToolCalls[0]
			unified.Tool = &models.ToolResponse{
				ID:   tc.ID,
				Type: tc.Type,
				Function: models.FunctionResponse{
					Name:      tc.Function.Name,
					Arguments: tc.Function.Arguments,
				},
			}
		}
	} else if chunk.FunctionCall != nil && chunk.FunctionCall.Name != "" {
		unified.Type = "function_call"
		unified.Function = &models.FunctionResponse{
			Name:      chunk.FunctionCall.Name,
			Arguments: chunk.FunctionCall.Arguments,
		}
	} else if chunk.Delta != "" {
		unified.Type = "content"
		unified.Content = chunk.Delta
	} else if chunk.Role != "" || chunk.Model != "" {
		unified.Type = "meta"
		unified.Metadata = &models.ChunkMetadata{
			Provider: a.providerName,
			Model:    chunk.Model,
		}
	}
	
	return unified, nil
}

// NormalizeError provides base implementation
func (a BaseAdapter) NormalizeError(err error, provider string) *models.UnifiedError {
	errStr := err.Error()
	
	// Determine error type based on error message patterns
	errorType := models.ErrorTypeProvider
	retry := false
	
	if strings.Contains(strings.ToLower(errStr), "rate limit") ||
	   strings.Contains(strings.ToLower(errStr), "too many requests") {
		errorType = models.ErrorTypeRateLimit
		retry = true
	} else if strings.Contains(strings.ToLower(errStr), "unauthorized") ||
	          strings.Contains(strings.ToLower(errStr), "authentication") ||
	          strings.Contains(strings.ToLower(errStr), "api key") {
		errorType = models.ErrorTypeAuth
	} else if strings.Contains(strings.ToLower(errStr), "context length") ||
	          strings.Contains(strings.ToLower(errStr), "maximum context") ||
	          strings.Contains(strings.ToLower(errStr), "token limit") {
		errorType = models.ErrorTypeModelLimit
		retry = true // Can retry with smaller context
	} else if strings.Contains(strings.ToLower(errStr), "connection") ||
	          strings.Contains(strings.ToLower(errStr), "timeout") ||
	          strings.Contains(strings.ToLower(errStr), "network") {
		errorType = models.ErrorTypeNetwork
		retry = true
	} else if strings.Contains(strings.ToLower(errStr), "invalid") ||
	          strings.Contains(strings.ToLower(errStr), "bad request") {
		errorType = models.ErrorTypeInvalid
	}
	
	return &models.UnifiedError{
		Code:    string(errorType),
		Message: errStr,
		Type:    errorType,
		Retry:   retry,
	}
}

// ExtractErrorCode attempts to extract error code from provider error
func ExtractErrorCode(err error) string {
	// Try to parse as JSON error
	var jsonErr struct {
		Error struct {
			Code string `json:"code"`
			Type string `json:"type"`
		} `json:"error"`
	}
	
	if json.Unmarshal([]byte(err.Error()), &jsonErr) == nil {
		if jsonErr.Error.Code != "" {
			return jsonErr.Error.Code
		}
		if jsonErr.Error.Type != "" {
			return jsonErr.Error.Type
		}
	}
	
	// Fallback to simple string matching
	errStr := strings.ToLower(err.Error())
	switch {
	case strings.Contains(errStr, "rate_limit"):
		return "rate_limit_exceeded"
	case strings.Contains(errStr, "invalid_api_key"):
		return "invalid_api_key"
	case strings.Contains(errStr, "model_not_found"):
		return "model_not_found"
	case strings.Contains(errStr, "context_length_exceeded"):
		return "context_length_exceeded"
	default:
		return "unknown_error"
	}
}

// EstimateCost calculates estimated cost based on usage
func EstimateCost(usage models.Usage, provider, model string) float64 {
	// Cost per 1K tokens (rough estimates)
	costs := map[string]map[string]struct{ input, output float64 }{
		"openai": {
			"gpt-4":           {0.03, 0.06},
			"gpt-4-turbo":     {0.01, 0.03},
			"gpt-3.5-turbo":   {0.0005, 0.0015},
		},
		"anthropic": {
			"claude-3-opus":   {0.015, 0.075},
			"claude-3-sonnet": {0.003, 0.015},
			"claude-3-haiku":  {0.00025, 0.00125},
		},
	}
	
	if providerCosts, ok := costs[provider]; ok {
		if modelCosts, ok := providerCosts[model]; ok {
			inputCost := float64(usage.PromptTokens) / 1000 * modelCosts.input
			outputCost := float64(usage.CompletionTokens) / 1000 * modelCosts.output
			return inputCost + outputCost
		}
	}
	
	return 0
}