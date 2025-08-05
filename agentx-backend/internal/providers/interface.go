package providers

import (
	"context"
)

// Provider defines the interface for all LLM providers
type Provider interface {
	// Name returns the provider name
	Name() string
	
	// Complete performs a non-streaming completion
	Complete(ctx context.Context, req CompletionRequest) (*CompletionResponse, error)
	
	// StreamComplete performs a streaming completion
	StreamComplete(ctx context.Context, req CompletionRequest) (<-chan StreamChunk, error)
	
	// GetModels returns available models
	GetModels(ctx context.Context) ([]Model, error)
	
	// ValidateConfig validates the provider configuration
	ValidateConfig() error
}

// CompletionRequest represents a chat completion request
type CompletionRequest struct {
	Messages        []Message        `json:"messages"`
	Model           string           `json:"model"`
	Temperature     *float32         `json:"temperature,omitempty"`
	MaxTokens       *int             `json:"max_tokens,omitempty"`
	Stream          bool             `json:"stream"`
	Functions       []Function       `json:"functions,omitempty"`
	ToolChoice      *ToolChoice      `json:"tool_choice,omitempty"`
	Tools           []Tool           `json:"tools,omitempty"`
	ResponseFormat  *ResponseFormat  `json:"response_format,omitempty"`
}

// Message represents a chat message
type Message struct {
	Role         string         `json:"role"`
	Content      string         `json:"content"`
	ContentArray []interface{}  `json:"content_array,omitempty"` // For multimodal content
	FunctionCall *FunctionCall  `json:"function_call,omitempty"`
	ToolCalls    []ToolCall     `json:"tool_calls,omitempty"`
	ToolCallID   string         `json:"tool_call_id,omitempty"`
}

// Function represents a function that can be called
type Function struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
}

// Tool represents a tool that can be used
type Tool struct {
	Type     string   `json:"type"`
	Function Function `json:"function"`
}

// FunctionCall represents a function call in a message
type FunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// ToolCall represents a tool call in a message
type ToolCall struct {
	ID       string       `json:"id"`
	Type     string       `json:"type"`
	Function FunctionCall `json:"function"`
}

// ToolChoice represents tool selection preference
type ToolChoice struct {
	Type string `json:"type"` // "auto", "none", or "function"
	Function *struct {
		Name string `json:"name"`
	} `json:"function,omitempty"`
}

// CompletionResponse represents a non-streaming response
type CompletionResponse struct {
	ID                string   `json:"id"`
	Object            string   `json:"object"`
	Created           int64    `json:"created"`
	Model             string   `json:"model"`
	Choices           []Choice `json:"choices"`
	Usage             Usage    `json:"usage"`
	SystemFingerprint string   `json:"system_fingerprint,omitempty"`
}

// Choice represents a completion choice
type Choice struct {
	Index        int          `json:"index"`
	Message      Message      `json:"message"`
	FinishReason string       `json:"finish_reason"`
}

// Usage represents token usage information
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// StreamChunk represents a chunk in a streaming response
type StreamChunk struct {
	ID           string        `json:"id,omitempty"`
	Object       string        `json:"object,omitempty"`
	Created      int64         `json:"created,omitempty"`
	Model        string        `json:"model,omitempty"`
	Delta        string        `json:"delta,omitempty"`
	Role         string        `json:"role,omitempty"`
	FunctionCall *FunctionCall `json:"function_call,omitempty"`
	ToolCalls    []ToolCall    `json:"tool_calls,omitempty"`
	FinishReason string        `json:"finish_reason,omitempty"`
	Error        string        `json:"error,omitempty"`
}

// ResponseFormat represents the desired response format
type ResponseFormat struct {
	Type string `json:"type"` // "text", "json_object"
}

// Model represents an available model
type Model struct {
	ID          string                 `json:"id"`
	Object      string                 `json:"object"`
	Created     int64                  `json:"created"`
	OwnedBy     string                 `json:"owned_by"`
	Permissions []interface{}         `json:"permissions,omitempty"`
	Extra       map[string]interface{} `json:"extra,omitempty"`
}