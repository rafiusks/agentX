package llm

import (
	"time"
)

// Message represents a single message in a conversation
type Message struct {
	Role       string                 `json:"role"`
	Content    string                 `json:"content"`
	Name       string                 `json:"name,omitempty"`
	ToolCalls  []ToolCall            `json:"tool_calls,omitempty"`
	ToolCallID string                `json:"tool_call_id,omitempty"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

// ToolCall represents a tool invocation
type ToolCall struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}

// Tool represents an available tool/function
type Tool struct {
	Type     string      `json:"type"`
	Function ToolFunction `json:"function"`
}

// ToolFunction describes a callable function
type ToolFunction struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
}

// Preferences for routing and model selection
type Preferences struct {
	Provider     string   `json:"provider,omitempty"`
	Model        string   `json:"model,omitempty"`
	ConnectionID string   `json:"connection_id,omitempty"`
	Capabilities []string `json:"capabilities,omitempty"`
}

// Requirements for the request
type Requirements struct {
	MaxLatency   time.Duration `json:"max_latency,omitempty"`
	MinQuality   string        `json:"min_quality,omitempty"`
	MaxCost      float64       `json:"max_cost,omitempty"`
	RequireTools bool          `json:"require_tools,omitempty"`
}

// Usage statistics
type Usage struct {
	PromptTokens     int     `json:"prompt_tokens"`
	CompletionTokens int     `json:"completion_tokens"`
	TotalTokens      int     `json:"total_tokens"`
	EstimatedCost    float64 `json:"estimated_cost,omitempty"`
}

// Metadata about the request/response
type Metadata struct {
	Provider      string        `json:"provider"`
	Model         string        `json:"model"`
	ConnectionID  string        `json:"connection_id,omitempty"`
	LatencyMs     int64         `json:"latency_ms,omitempty"`
	Retries       int           `json:"retries,omitempty"`
	FallbackUsed  bool          `json:"fallback_used,omitempty"`
	CircuitBreaker string       `json:"circuit_breaker_status,omitempty"`
	Extra         map[string]interface{} `json:"extra,omitempty"`
}