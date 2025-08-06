package models

import (
	"github.com/agentx/agentx-backend/internal/providers"
)

// UnifiedChatRequest represents a provider-agnostic chat request
type UnifiedChatRequest struct {
	// Session ID for context continuity
	SessionID string `json:"session_id,omitempty"`
	
	// Messages for the conversation
	Messages []providers.Message `json:"messages"`
	
	// User preferences instead of specific model/provider
	Preferences Preferences `json:"preferences,omitempty"`
	
	// Required capabilities
	Requirements providers.Requirements `json:"requirements,omitempty"`
	
	// Universal parameters
	Temperature    *float32 `json:"temperature,omitempty"`
	MaxTokens      *int     `json:"max_tokens,omitempty"`
	ResponseFormat string   `json:"response_format,omitempty"` // text, json, markdown, code
	
	// Advanced features (auto-detected and routed)
	Functions []providers.Function `json:"functions,omitempty"`
	Tools     []providers.Tool     `json:"tools,omitempty"`
	Images    []Image              `json:"images,omitempty"`
	Audio     *Audio               `json:"audio,omitempty"`
}

// Preferences for routing decisions
type Preferences struct {
	Speed        string `json:"speed,omitempty"`         // fast, balanced, quality
	Cost         string `json:"cost,omitempty"`          // economy, standard, premium
	Privacy      string `json:"privacy,omitempty"`       // local, cloud
	Provider     string `json:"provider,omitempty"`      // optional override (legacy)
	Model        string `json:"model,omitempty"`         // optional override
	ConnectionID string `json:"connection_id,omitempty"` // specific connection to use
}

// Image for multimodal requests
type Image struct {
	URL    string `json:"url,omitempty"`
	Base64 string `json:"base64,omitempty"`
	Type   string `json:"type,omitempty"` // image/jpeg, image/png, etc.
}

// Audio for audio input
type Audio struct {
	URL    string `json:"url,omitempty"`
	Base64 string `json:"base64,omitempty"`
	Type   string `json:"type,omitempty"` // audio/mp3, audio/wav, etc.
}

// UnifiedChatResponse represents a provider-agnostic response
type UnifiedChatResponse struct {
	ID        string              `json:"id"`
	Content   string              `json:"content"`
	Role      string              `json:"role"`
	Functions []FunctionResponse  `json:"functions,omitempty"`
	Tools     []ToolResponse      `json:"tools,omitempty"`
	Metadata  ResponseMetadata    `json:"metadata"`
	Usage     Usage               `json:"usage"`
}

// FunctionResponse for function calls
type FunctionResponse struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// ToolResponse for tool use
type ToolResponse struct {
	ID       string           `json:"id"`
	Type     string           `json:"type"`
	Function FunctionResponse `json:"function"`
}

// ResponseMetadata provides transparency about routing
type ResponseMetadata struct {
	Provider      string  `json:"provider"`
	Model         string  `json:"model"`
	LatencyMs     int64   `json:"latency_ms"`
	Confidence    float32 `json:"confidence,omitempty"`
	RoutingReason string  `json:"routing_reason,omitempty"`
}

// Usage information
type Usage struct {
	PromptTokens     int     `json:"prompt_tokens"`
	CompletionTokens int     `json:"completion_tokens"`
	TotalTokens      int     `json:"total_tokens"`
	EstimatedCost    float64 `json:"estimated_cost,omitempty"`
}

// UnifiedStreamChunk for streaming responses
type UnifiedStreamChunk struct {
	Type     string           `json:"type"` // content, function_call, tool_use, error, meta, done
	Content  string           `json:"content,omitempty"`
	Function *FunctionResponse `json:"function,omitempty"`
	Tool     *ToolResponse     `json:"tool,omitempty"`
	Error    *UnifiedError     `json:"error,omitempty"`
	Metadata *ChunkMetadata    `json:"metadata,omitempty"`
}

// ChunkMetadata for streaming
type ChunkMetadata struct {
	Provider   string `json:"provider,omitempty"`
	Model      string `json:"model,omitempty"`
	TokenCount int    `json:"token_count,omitempty"`
	LatencyMs  int64  `json:"latency_ms,omitempty"`
}

// UnifiedError represents normalized errors
type UnifiedError struct {
	Code     string    `json:"code"`
	Message  string    `json:"message"`
	Type     ErrorType `json:"type"`
	Retry    bool      `json:"retry"`
	Fallback *Fallback `json:"fallback,omitempty"`
}

// ErrorType categorizes errors
type ErrorType string

const (
	ErrorTypeRateLimit  ErrorType = "rate_limit"
	ErrorTypeModelLimit ErrorType = "model_limit"
	ErrorTypeProvider   ErrorType = "provider_error"
	ErrorTypeCapability ErrorType = "capability_missing"
	ErrorTypeAuth       ErrorType = "authentication"
	ErrorTypeNetwork    ErrorType = "network"
	ErrorTypeInvalid    ErrorType = "invalid_request"
)

// Fallback information
type Fallback struct {
	Provider string `json:"provider"`
	Model    string `json:"model"`
	Reason   string `json:"reason"`
}

// UnifiedModelsResponse for listing available models
type UnifiedModelsResponse struct {
	Models []ModelInfo `json:"models"`
	Total  int         `json:"total"`
}

// ModelInfo provides model details
type ModelInfo struct {
	ID           string                    `json:"id"`
	Provider     string                    `json:"provider"`
	DisplayName  string                    `json:"display_name"`
	Description  string                    `json:"description"`
	Capabilities providers.Capabilities    `json:"capabilities"`
	Pricing      PricingInfo               `json:"pricing"`
	Status       ModelStatus               `json:"status"`
}

// PricingInfo for cost estimation
type PricingInfo struct {
	Tier           string  `json:"tier"` // free, economy, standard, premium
	InputPer1K     float64 `json:"input_per_1k,omitempty"`
	OutputPer1K    float64 `json:"output_per_1k,omitempty"`
	Currency       string  `json:"currency,omitempty"`
}

// ModelStatus indicates availability
type ModelStatus struct {
	Available bool   `json:"available"`
	Health    string `json:"health"` // healthy, degraded, unavailable
	Message   string `json:"message,omitempty"`
}