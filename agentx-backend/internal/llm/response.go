package llm

import (
	"time"
)

// Response represents a unified LLM response
type Response struct {
	ID        string    `json:"id"`
	Object    string    `json:"object"`
	Created   time.Time `json:"created"`
	Model     string    `json:"model"`
	
	// Response content
	Choices []Choice `json:"choices"`
	
	// Simplified access
	Content  string `json:"content"`  // Direct access to first choice content
	Role     string `json:"role"`     // Direct access to first choice role
	Provider string `json:"provider"` // Provider that handled the request
	
	// Usage statistics
	Usage Usage `json:"usage"`
	
	// Response metadata
	Metadata Metadata `json:"metadata"`
	
	// System fingerprint (for reproducibility)
	SystemFingerprint string `json:"system_fingerprint,omitempty"`
}

// Choice represents a single response choice
type Choice struct {
	Index        int      `json:"index"`
	Message      Message  `json:"message"`
	FinishReason string   `json:"finish_reason"`
	LogProbs     *LogProbs `json:"logprobs,omitempty"`
}

// LogProbs contains log probability information
type LogProbs struct {
	Content []TokenLogProb `json:"content"`
}

// TokenLogProb represents log probability for a token
type TokenLogProb struct {
	Token       string  `json:"token"`
	LogProb     float64 `json:"logprob"`
	Bytes       []byte  `json:"bytes,omitempty"`
	TopLogProbs []struct {
		Token   string  `json:"token"`
		LogProb float64 `json:"logprob"`
	} `json:"top_logprobs,omitempty"`
}

// StreamChunk represents a chunk in a streaming response
type StreamChunk struct {
	ID       string         `json:"id"`
	Object   string         `json:"object"`
	Created  time.Time      `json:"created"`
	Model    string         `json:"model"`
	Provider string         `json:"provider"`
	Type     string         `json:"type"` // "content", "error", "done"
	Content  string         `json:"content,omitempty"`
	Choices  []StreamChoice `json:"choices"`
	Usage    *Usage         `json:"usage,omitempty"`
	Error    error          `json:"error,omitempty"`
}

// StreamChoice represents a choice in a streaming chunk
type StreamChoice struct {
	Index        int          `json:"index"`
	Delta        MessageDelta `json:"delta"`
	FinishReason string       `json:"finish_reason,omitempty"`
	LogProbs     *LogProbs    `json:"logprobs,omitempty"`
}

// MessageDelta represents incremental message content
type MessageDelta struct {
	Role      string     `json:"role,omitempty"`
	Content   string     `json:"content,omitempty"`
	ToolCalls []ToolCall `json:"tool_calls,omitempty"`
}

// GetContent returns the content from the first choice
func (r *Response) GetContent() string {
	if len(r.Choices) > 0 {
		return r.Choices[0].Message.Content
	}
	return ""
}

// GetRole returns the role from the first choice
func (r *Response) GetRole() string {
	if len(r.Choices) > 0 {
		return r.Choices[0].Message.Role
	}
	return ""
}

// HasToolCalls checks if the response contains tool calls
func (r *Response) HasToolCalls() bool {
	if len(r.Choices) > 0 {
		return len(r.Choices[0].Message.ToolCalls) > 0
	}
	return false
}

// GetToolCalls returns tool calls from the first choice
func (r *Response) GetToolCalls() []ToolCall {
	if len(r.Choices) > 0 {
		return r.Choices[0].Message.ToolCalls
	}
	return nil
}

// IsComplete checks if the response is complete
func (r *Response) IsComplete() bool {
	if len(r.Choices) > 0 {
		return r.Choices[0].FinishReason != ""
	}
	return false
}

// GetFinishReason returns the finish reason from the first choice
func (r *Response) GetFinishReason() string {
	if len(r.Choices) > 0 {
		return r.Choices[0].FinishReason
	}
	return ""
}