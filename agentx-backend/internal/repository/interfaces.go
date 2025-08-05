package repository

import (
	"context"
	"database/sql"
	"time"
)

// Session represents a chat session
type Session struct {
	ID        string         `db:"id"`
	Title     string         `db:"title"`
	Provider  sql.NullString `db:"provider"`
	Model     sql.NullString `db:"model"`
	CreatedAt time.Time      `db:"created_at"`
	UpdatedAt time.Time      `db:"updated_at"`
	Metadata  []byte         `db:"metadata"`
}

// Message represents a chat message
type Message struct {
	ID           string         `db:"id"`
	SessionID    string         `db:"session_id"`
	Role         string         `db:"role"`
	Content      string         `db:"content"`
	FunctionCall sql.NullString `db:"function_call"`
	ToolCalls    sql.NullString `db:"tool_calls"`
	ToolCallID   sql.NullString `db:"tool_call_id"`
	CreatedAt    time.Time      `db:"created_at"`
	Metadata     []byte         `db:"metadata"`
}

// SessionRepository defines session storage operations
type SessionRepository interface {
	Create(ctx context.Context, session Session) (string, error)
	Get(ctx context.Context, id string) (*Session, error)
	List(ctx context.Context) ([]*Session, error)
	Update(ctx context.Context, id string, updates map[string]interface{}) error
	Delete(ctx context.Context, id string) error
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
	Create(ctx context.Context, providerID, name string, config map[string]interface{}) (string, error)
	GetByID(ctx context.Context, id string) (*ProviderConnection, error)
	GetByProviderID(ctx context.Context, providerID string) ([]*ProviderConnection, error)
	List(ctx context.Context) ([]*ProviderConnection, error)
	Update(ctx context.Context, id string, updates map[string]interface{}) error
	Delete(ctx context.Context, id string) error
	SetDefault(ctx context.Context, providerID string, connectionID string) error
	GetDefault(ctx context.Context, providerID string) (*ProviderConnection, error)
}

// ProviderConnection represents a named connection to a provider
type ProviderConnection struct {
	ID         string                 `db:"id"`
	ProviderID string                 `db:"provider_id"`
	Name       string                 `db:"name"`
	Enabled    bool                   `db:"enabled"`
	Config     map[string]interface{} `db:"config"`
	Metadata   map[string]interface{} `db:"metadata"`
	CreatedAt  time.Time              `db:"created_at"`
	UpdatedAt  time.Time              `db:"updated_at"`
}