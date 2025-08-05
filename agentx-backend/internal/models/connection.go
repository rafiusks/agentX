package models

import (
	"time"

	"github.com/google/uuid"
)

// ProviderConnection represents a named connection to a provider
type ProviderConnection struct {
	ID         uuid.UUID              `json:"id" db:"id"`
	ProviderID string                 `json:"provider_id" db:"provider_id"`
	Name       string                 `json:"name" db:"name"`
	Enabled    bool                   `json:"enabled" db:"enabled"`
	Config     map[string]interface{} `json:"config" db:"config"`
	Metadata   map[string]interface{} `json:"metadata" db:"metadata"`
	CreatedAt  time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt  time.Time              `json:"updated_at" db:"updated_at"`
}

// ConnectionConfig represents provider-specific configuration
type ConnectionConfig struct {
	APIKey       string   `json:"api_key,omitempty"`
	BaseURL      string   `json:"base_url,omitempty"`
	Models       []string `json:"models,omitempty"`
	DefaultModel string   `json:"default_model,omitempty"`
}

// ConnectionMetadata represents additional connection metadata
type ConnectionMetadata struct {
	LastTested   *time.Time `json:"last_tested,omitempty"`
	TestStatus   string     `json:"test_status,omitempty"`
	TestMessage  string     `json:"test_message,omitempty"`
	LastUsed     *time.Time `json:"last_used,omitempty"`
	RequestCount int64      `json:"request_count,omitempty"`
}

// ConnectionWithStatus includes real-time status information
type ConnectionWithStatus struct {
	*ProviderConnection
	Status       string `json:"status"` // connected, connecting, error, disconnected
	StatusDetail string `json:"status_detail,omitempty"`
}

// CreateConnectionRequest represents a request to create a new connection
type CreateConnectionRequest struct {
	ProviderID string                 `json:"provider_id" validate:"required"`
	Name       string                 `json:"name" validate:"required"`
	Config     map[string]interface{} `json:"config" validate:"required"`
}

// UpdateConnectionRequest represents a request to update a connection
type UpdateConnectionRequest struct {
	Name     *string                 `json:"name,omitempty"`
	Enabled  *bool                   `json:"enabled,omitempty"`
	Config   *map[string]interface{} `json:"config,omitempty"`
	Metadata *map[string]interface{} `json:"metadata,omitempty"`
}

// TestConnectionResponse represents the result of testing a connection
type TestConnectionResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Details map[string]interface{} `json:"details,omitempty"`
}