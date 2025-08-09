package llm

import (
	"context"
	"time"
)

// TaskType defines the type of LLM task
type TaskType string

const (
	TaskGenerateTitle  TaskType = "generate_title"
	TaskSummarize      TaskType = "summarize"
	TaskTranslate      TaskType = "translate"
	TaskCodeGeneration TaskType = "code_generation"
	TaskCustom         TaskType = "custom"
)

// CompletionRequest represents a general LLM completion request
type CompletionRequest struct {
	Task         TaskType               `json:"task" validate:"required"`
	Context      map[string]interface{} `json:"context,omitempty"`
	Parameters   Parameters             `json:"parameters,omitempty"`
	ConnectionID string                 `json:"connection_id,omitempty"`
	ProviderHints ProviderHints         `json:"provider_hints,omitempty"`
}

// Parameters for LLM requests
type Parameters struct {
	MaxTokens    *int     `json:"max_tokens,omitempty"`
	Temperature  *float32 `json:"temperature,omitempty"`
	TopP         *float32 `json:"top_p,omitempty"`
	SystemPrompt string   `json:"system_prompt,omitempty"`
	UserPrompt   string   `json:"user_prompt,omitempty"`
}

// ProviderHints for routing logic
type ProviderHints struct {
	Preferred []string `json:"preferred,omitempty"`
	Exclude   []string `json:"exclude,omitempty"`
}

// CompletionResponse represents the LLM response
type CompletionResponse struct {
	Result       string                 `json:"result"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
	Provider     string                 `json:"provider"`
	ConnectionID string                 `json:"connection_id"`
	Usage        *TaskUsage             `json:"usage,omitempty"`
	Duration     time.Duration          `json:"duration,omitempty"`
}

// TaskUsage tracks token usage for tasks
type TaskUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// TaskHandler defines the interface for task-specific handlers
type TaskHandler interface {
	Handle(ctx context.Context, req CompletionRequest) (*CompletionResponse, error)
	ValidateRequest(req CompletionRequest) error
}

// Validate validates a completion request
func (r CompletionRequest) Validate() error {
	if r.Task == "" {
		return ErrTaskRequired
	}
	return nil
}

// Common errors
var (
	ErrTaskRequired       = &LLMError{Code: "TASK_REQUIRED", Message: "task is required"}
	ErrSessionIDRequired  = &LLMError{Code: "SESSION_ID_REQUIRED", Message: "session_id is required"}
	ErrNoConnectionFound  = &LLMError{Code: "NO_CONNECTION_FOUND", Message: "no suitable connection found"}
	ErrInvalidConnection  = &LLMError{Code: "INVALID_CONNECTION", Message: "invalid connection specified"}
	ErrPromptRequired     = &LLMError{Code: "PROMPT_REQUIRED", Message: "at least one prompt is required"}
)

// LLMError represents a structured error for LLM operations
type LLMError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (e *LLMError) Error() string {
	return e.Message
}