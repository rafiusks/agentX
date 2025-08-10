package models

import (
	"database/sql/driver"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

// MCPServer represents an MCP server configuration
type MCPServer struct {
	ID             uuid.UUID         `json:"id" db:"id"`
	UserID         uuid.UUID         `json:"user_id" db:"user_id"`
	Name           string            `json:"name" db:"name"`
	Description    string            `json:"description" db:"description"`
	Command        string            `json:"command" db:"command"`
	Args           pq.StringArray    `json:"args" db:"args"`
	Env            json.RawMessage   `json:"env" db:"env"`
	Enabled        bool              `json:"enabled" db:"enabled"`
	Status         string            `json:"status" db:"status"`
	LastConnectedAt *time.Time       `json:"last_connected_at" db:"last_connected_at"`
	Capabilities   json.RawMessage   `json:"capabilities" db:"capabilities"`
	Metadata       json.RawMessage   `json:"metadata" db:"metadata"`
	CreatedAt      time.Time         `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time         `json:"updated_at" db:"updated_at"`
	
	// Relationships
	Tools     []MCPTool     `json:"tools,omitempty"`
	Resources []MCPResource `json:"resources,omitempty"`
}

// MCPTool represents a tool exposed by an MCP server
type MCPTool struct {
	ID          uuid.UUID       `json:"id" db:"id"`
	ServerID    uuid.UUID       `json:"server_id" db:"server_id"`
	Name        string          `json:"name" db:"name"`
	Description string          `json:"description" db:"description"`
	InputSchema json.RawMessage `json:"input_schema" db:"input_schema"`
	Enabled     bool            `json:"enabled" db:"enabled"`
	UsageCount  int             `json:"usage_count" db:"usage_count"`
	LastUsedAt  *time.Time      `json:"last_used_at" db:"last_used_at"`
	CreatedAt   time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at" db:"updated_at"`
}

// MCPResource represents a resource exposed by an MCP server
type MCPResource struct {
	ID          uuid.UUID       `json:"id" db:"id"`
	ServerID    uuid.UUID       `json:"server_id" db:"server_id"`
	URI         string          `json:"uri" db:"uri"`
	Name        string          `json:"name" db:"name"`
	Description string          `json:"description" db:"description"`
	MimeType    string          `json:"mime_type" db:"mime_type"`
	Metadata    json.RawMessage `json:"metadata" db:"metadata"`
	CreatedAt   time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at" db:"updated_at"`
}

// MCPServerStatus represents the connection status of an MCP server
type MCPServerStatus string

const (
	MCPServerStatusConnected    MCPServerStatus = "connected"
	MCPServerStatusDisconnected MCPServerStatus = "disconnected"
	MCPServerStatusError        MCPServerStatus = "error"
	MCPServerStatusConnecting   MCPServerStatus = "connecting"
)

// MCPServerCreateRequest represents a request to create a new MCP server
type MCPServerCreateRequest struct {
	Name        string            `json:"name" validate:"required,min=1,max=255"`
	Description string            `json:"description"`
	Command     string            `json:"command" validate:"required,min=1,max=500"`
	Args        []string          `json:"args"`
	Env         map[string]string `json:"env"`
	Enabled     bool              `json:"enabled"`
}

// MCPServerUpdateRequest represents a request to update an MCP server
type MCPServerUpdateRequest struct {
	Name        string            `json:"name,omitempty" validate:"omitempty,min=1,max=255"`
	Description string            `json:"description,omitempty"`
	Command     string            `json:"command,omitempty" validate:"omitempty,min=1,max=500"`
	Args        []string          `json:"args,omitempty"`
	Env         map[string]string `json:"env,omitempty"`
	Enabled     *bool             `json:"enabled,omitempty"`
}

// MCPToolCallRequest represents a request to call an MCP tool
type MCPToolCallRequest struct {
	ServerID  uuid.UUID              `json:"server_id" validate:"required"`
	ToolName  string                 `json:"tool_name" validate:"required"`
	Arguments map[string]interface{} `json:"arguments"`
}

// MCPToolCallResponse represents the response from an MCP tool call
type MCPToolCallResponse struct {
	Result interface{} `json:"result"`
	Error  *string     `json:"error,omitempty"`
}

// MCPResourceReadRequest represents a request to read an MCP resource
type MCPResourceReadRequest struct {
	ServerID uuid.UUID `json:"server_id" validate:"required"`
	URI      string    `json:"uri" validate:"required"`
}

// MCPResourceReadResponse represents the response from reading an MCP resource
type MCPResourceReadResponse struct {
	Content  interface{} `json:"content"`
	MimeType string      `json:"mime_type"`
	Error    *string     `json:"error,omitempty"`
}

// StringMap is a custom type for map[string]string that can be stored as JSONB
type StringMap map[string]string

// Value implements the driver.Valuer interface
func (m StringMap) Value() (driver.Value, error) {
	return json.Marshal(m)
}

// Scan implements the sql.Scanner interface
func (m *StringMap) Scan(value interface{}) error {
	if value == nil {
		*m = make(StringMap)
		return nil
	}
	
	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}
	
	return json.Unmarshal(bytes, m)
}