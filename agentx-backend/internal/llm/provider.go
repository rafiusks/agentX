package llm

import (
	"context"
	"time"
	
	"github.com/agentx/agentx-backend/internal/config"
)

// Provider is the interface that all LLM providers must implement
type Provider interface {
	// Complete performs a non-streaming completion
	Complete(ctx context.Context, req *Request) (*Response, error)
	
	// StreamComplete performs a streaming completion
	StreamComplete(ctx context.Context, req *Request) (<-chan *StreamChunk, error)
	
	// GetModels returns available models for this provider
	GetModels(ctx context.Context) ([]ModelInfo, error)
	
	// HealthCheck checks if the provider is healthy
	HealthCheck(ctx context.Context) error
	
	// GetCapabilities returns provider capabilities
	GetCapabilities() ProviderCapabilities
	
	// Close closes the provider connection
	Close() error
}

// ProviderConfig contains configuration for a provider
type ProviderConfig struct {
	Type         string                 `json:"type"`         // openai, anthropic, local, etc.
	Name         string                 `json:"name"`         // User-friendly name
	APIKey       string                 `json:"api_key,omitempty"`
	BaseURL      string                 `json:"base_url,omitempty"`
	Organization string                 `json:"organization,omitempty"`
	Headers      map[string]string      `json:"headers,omitempty"`
	Settings     map[string]interface{} `json:"settings,omitempty"`
	
	// Rate limiting
	RateLimit    int           `json:"rate_limit,omitempty"`     // Requests per minute
	Timeout      time.Duration `json:"timeout,omitempty"`
	MaxRetries   int           `json:"max_retries,omitempty"`
	
	// Cost tracking
	PricePerToken map[string]float64 `json:"price_per_token,omitempty"`
}

// ToProviderConfig converts to the providers package config format
func (c ProviderConfig) ToProviderConfig() config.ProviderConfig {
	return config.ProviderConfig{
		Type:    c.Type,
		Name:    c.Name,
		APIKey:  c.APIKey,
		BaseURL: c.BaseURL,
		Extra:   c.Settings,
	}
}

// ProviderCapabilities describes what a provider can do
type ProviderCapabilities struct {
	Streaming        bool     `json:"streaming"`
	FunctionCalling  bool     `json:"function_calling"`
	Vision           bool     `json:"vision"`
	AudioInput       bool     `json:"audio_input"`
	AudioOutput      bool     `json:"audio_output"`
	MaxTokens        int      `json:"max_tokens"`
	SupportedModels  []string `json:"supported_models"`
}

// ModelInfo contains information about a model
type ModelInfo struct {
	ID           string               `json:"id"`
	Provider     string               `json:"provider"`
	DisplayName  string               `json:"display_name"`
	Description  string               `json:"description"`
	Capabilities []string             `json:"capabilities"`
	MaxTokens    int                  `json:"max_tokens"`
	PricingTier  string               `json:"pricing_tier"`
	Status       ModelStatus          `json:"status"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// ModelStatus represents the status of a model
type ModelStatus struct {
	Available bool      `json:"available"`
	Health    string    `json:"health"` // healthy, degraded, unhealthy
	LastCheck time.Time `json:"last_check"`
	Message   string    `json:"message,omitempty"`
}

// HealthStatus represents provider health
type HealthStatus struct {
	Provider    string        `json:"provider"`
	Status      string        `json:"status"` // healthy, degraded, unhealthy
	Latency     time.Duration `json:"latency"`
	LastCheck   time.Time     `json:"last_check"`
	Error       string        `json:"error,omitempty"`
	Models      int           `json:"models_available"`
}