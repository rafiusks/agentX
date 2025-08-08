package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
)

// Session represents a chat session
type Session struct {
	ID        string         `db:"id" json:"ID"`
	UserID    uuid.UUID      `db:"user_id" json:"UserID"`
	Title     string         `db:"title" json:"Title"`
	Provider  sql.NullString `db:"provider" json:"Provider,omitempty"`
	Model     sql.NullString `db:"model" json:"Model,omitempty"`
	CreatedAt time.Time      `db:"created_at" json:"CreatedAt"`
	UpdatedAt time.Time      `db:"updated_at" json:"UpdatedAt"`
	Metadata  []byte         `db:"metadata" json:"Metadata,omitempty"`
}

// Message represents a chat message
type Message struct {
	ID           string         `db:"id" json:"id"`
	SessionID    string         `db:"session_id" json:"chat_id"`
	Role         string         `db:"role" json:"role"`
	Content      string         `db:"content" json:"content"`
	FunctionCall sql.NullString `db:"function_call" json:"function_call,omitempty"`
	ToolCalls    sql.NullString `db:"tool_calls" json:"tool_calls,omitempty"`
	ToolCallID   sql.NullString `db:"tool_call_id" json:"tool_call_id,omitempty"`
	CreatedAt    time.Time      `db:"created_at" json:"created_at"`
	Metadata     []byte         `db:"metadata" json:"metadata,omitempty"`
}

// SessionRepository defines session storage operations
type SessionRepository interface {
	Create(ctx context.Context, userID uuid.UUID, session Session) (string, error)
	Get(ctx context.Context, userID uuid.UUID, id string) (*Session, error)
	List(ctx context.Context, userID uuid.UUID) ([]*Session, error)
	Update(ctx context.Context, userID uuid.UUID, id string, updates map[string]interface{}) error
	Delete(ctx context.Context, userID uuid.UUID, id string) error
}

// MessageRepository defines message storage operations
type MessageRepository interface {
	Create(ctx context.Context, message Message) (string, error)
	ListBySession(ctx context.Context, sessionID string) ([]Message, error)
	Delete(ctx context.Context, id string) error
}

// ConfigRepository defines configuration storage operations
type ConfigRepository interface {
	Get(ctx context.Context, key string) (interface{}, error)
	Set(ctx context.Context, key string, value interface{}) error
	GetAll(ctx context.Context) (map[string]interface{}, error)
	Delete(ctx context.Context, key string) error
}

// ProviderConfig represents a provider configuration
type ProviderConfig struct {
	ID       string                 `db:"id"`
	Type     string                 `db:"type"`
	Name     string                 `db:"name"`
	Enabled  bool                   `db:"enabled"`
	Settings map[string]interface{} `db:"settings"`
	CreatedAt time.Time             `db:"created_at"`
	UpdatedAt time.Time             `db:"updated_at"`
}

// ConnectionRepository defines provider connection storage operations
type ConnectionRepository interface {
	Create(ctx context.Context, userID uuid.UUID, providerID, name string, config map[string]interface{}) (string, error)
	GetByID(ctx context.Context, userID uuid.UUID, id string) (*ProviderConnection, error)
	GetByProviderID(ctx context.Context, userID uuid.UUID, providerID string) ([]*ProviderConnection, error)
	List(ctx context.Context, userID uuid.UUID) ([]*ProviderConnection, error)
	Update(ctx context.Context, userID uuid.UUID, id string, updates map[string]interface{}) error
	Delete(ctx context.Context, userID uuid.UUID, id string) error
	SetDefault(ctx context.Context, userID uuid.UUID, providerID string, connectionID string) error
	GetDefault(ctx context.Context, userID uuid.UUID, providerID string) (*ProviderConnection, error)
}

// ProviderConnection represents a named connection to a provider
type ProviderConnection struct {
	ID         string                 `db:"id"`
	UserID     uuid.UUID              `db:"user_id"`
	ProviderID string                 `db:"provider_id"`
	Name       string                 `db:"name"`
	Enabled    bool                   `db:"enabled"`
	Config     map[string]interface{} `db:"config"`
	Metadata   map[string]interface{} `db:"metadata"`
	CreatedAt  time.Time              `db:"created_at"`
	UpdatedAt  time.Time              `db:"updated_at"`
}