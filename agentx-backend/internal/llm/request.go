package llm

import (
	"context"
	"fmt"
)

// Request represents a unified LLM request
type Request struct {
	// Core fields
	Messages []Message `json:"messages"`
	Model    string    `json:"model,omitempty"`
	Stream   bool      `json:"stream,omitempty"`

	// User context
	UserID       string `json:"user_id"`
	SessionID    string `json:"session_id,omitempty"`
	ConnectionID string `json:"connection_id,omitempty"`

	// Generation parameters
	Temperature      *float32 `json:"temperature,omitempty"`
	MaxTokens        *int     `json:"max_tokens,omitempty"`
	TopP             *float32 `json:"top_p,omitempty"`
	FrequencyPenalty *float32 `json:"frequency_penalty,omitempty"`
	PresencePenalty  *float32 `json:"presence_penalty,omitempty"`
	Stop             []string `json:"stop,omitempty"`
	N                *int     `json:"n,omitempty"`

	// Advanced features
	Tools         []Tool                 `json:"tools,omitempty"`
	ToolChoice    interface{}            `json:"tool_choice,omitempty"`
	ResponseFormat map[string]interface{} `json:"response_format,omitempty"`

	// Routing and requirements
	Preferences  Preferences `json:"preferences,omitempty"`
	Requirements Requirements `json:"requirements,omitempty"`

	// Request metadata
	Metadata map[string]interface{} `json:"metadata,omitempty"`

	// Internal context (not serialized)
	ctx context.Context
}

// NewRequest creates a new LLM request
func NewRequest(userID string, messages []Message) *Request {
	return &Request{
		UserID:   userID,
		Messages: messages,
		Metadata: make(map[string]interface{}),
	}
}

// WithContext sets the context for the request
func (r *Request) WithContext(ctx context.Context) *Request {
	r.ctx = ctx
	return r
}

// Context returns the request context
func (r *Request) Context() context.Context {
	if r.ctx == nil {
		return context.Background()
	}
	return r.ctx
}

// WithSession sets the session ID
func (r *Request) WithSession(sessionID string) *Request {
	r.SessionID = sessionID
	return r
}

// WithConnection sets the connection ID
func (r *Request) WithConnection(connectionID string) *Request {
	r.ConnectionID = connectionID
	return r
}

// WithModel sets the model preference
func (r *Request) WithModel(model string) *Request {
	r.Model = model
	return r
}

// WithProvider sets the provider preference
func (r *Request) WithProvider(provider string) *Request {
	r.Preferences.Provider = provider
	return r
}

// WithTemperature sets the temperature parameter
func (r *Request) WithTemperature(temp float32) *Request {
	r.Temperature = &temp
	return r
}

// WithMaxTokens sets the max tokens parameter
func (r *Request) WithMaxTokens(tokens int) *Request {
	r.MaxTokens = &tokens
	return r
}

// WithStreaming enables streaming mode
func (r *Request) WithStreaming() *Request {
	r.Stream = true
	return r
}

// Validate checks if the request is valid
func (r *Request) Validate() error {
	if r.UserID == "" {
		return fmt.Errorf("user_id is required")
	}
	if len(r.Messages) == 0 {
		return fmt.Errorf("at least one message is required")
	}
	
	// Validate temperature range
	if r.Temperature != nil && (*r.Temperature < 0 || *r.Temperature > 2) {
		return fmt.Errorf("temperature must be between 0 and 2")
	}
	
	// Validate top_p range
	if r.TopP != nil && (*r.TopP < 0 || *r.TopP > 1) {
		return fmt.Errorf("top_p must be between 0 and 1")
	}
	
	return nil
}

// Clone creates a deep copy of the request
func (r *Request) Clone() *Request {
	clone := &Request{
		Messages:     make([]Message, len(r.Messages)),
		Model:        r.Model,
		Stream:       r.Stream,
		UserID:       r.UserID,
		SessionID:    r.SessionID,
		ConnectionID: r.ConnectionID,
		Preferences:  r.Preferences,
		Requirements: r.Requirements,
		ctx:          r.ctx,
	}
	
	// Copy messages
	copy(clone.Messages, r.Messages)
	
	// Copy generation parameters
	if r.Temperature != nil {
		temp := *r.Temperature
		clone.Temperature = &temp
	}
	if r.MaxTokens != nil {
		tokens := *r.MaxTokens
		clone.MaxTokens = &tokens
	}
	if r.TopP != nil {
		topP := *r.TopP
		clone.TopP = &topP
	}
	
	// Copy metadata
	if r.Metadata != nil {
		clone.Metadata = make(map[string]interface{})
		for k, v := range r.Metadata {
			clone.Metadata[k] = v
		}
	}
	
	return clone
}