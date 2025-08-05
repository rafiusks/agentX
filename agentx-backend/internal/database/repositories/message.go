package repositories

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/agentx/agentx-backend/internal/providers"
)

// Message represents a message in the database
type Message struct {
	ID           string          `db:"id"`
	SessionID    string          `db:"session_id"`
	Role         string          `db:"role"`
	Content      string          `db:"content"`
	FunctionCall json.RawMessage `db:"function_call"`
	ToolCalls    json.RawMessage `db:"tool_calls"`
	ToolCallID   sql.NullString  `db:"tool_call_id"`
	CreatedAt    time.Time       `db:"created_at"`
	Metadata     json.RawMessage `db:"metadata"`
}

// MessageRepository handles message database operations
type MessageRepository struct {
	db *sqlx.DB
}

// NewMessageRepository creates a new message repository
func NewMessageRepository(db *sqlx.DB) *MessageRepository {
	return &MessageRepository{db: db}
}

// Create creates a new message
func (r *MessageRepository) Create(sessionID string, msg providers.Message) error {
	dbMsg := &Message{
		ID:        uuid.New().String(),
		SessionID: sessionID,
		Role:      msg.Role,
		Content:   msg.Content,
		CreatedAt: time.Now(),
		Metadata:  json.RawMessage("{}"),
	}

	// Handle optional fields
	if msg.FunctionCall != nil {
		funcCallJSON, err := json.Marshal(msg.FunctionCall)
		if err != nil {
			return fmt.Errorf("failed to marshal function call: %w", err)
		}
		dbMsg.FunctionCall = funcCallJSON
	}

	if len(msg.ToolCalls) > 0 {
		toolCallsJSON, err := json.Marshal(msg.ToolCalls)
		if err != nil {
			return fmt.Errorf("failed to marshal tool calls: %w", err)
		}
		dbMsg.ToolCalls = toolCallsJSON
	}

	if msg.ToolCallID != "" {
		dbMsg.ToolCallID = sql.NullString{String: msg.ToolCallID, Valid: true}
	}

	query := `
		INSERT INTO messages (id, session_id, role, content, function_call, tool_calls, tool_call_id, created_at, metadata)
		VALUES (:id, :session_id, :role, :content, :function_call, :tool_calls, :tool_call_id, :created_at, :metadata)`

	_, err := r.db.NamedExec(query, dbMsg)
	if err != nil {
		return fmt.Errorf("failed to create message: %w", err)
	}

	// Update session's updated_at
	updateQuery := `UPDATE sessions SET updated_at = CURRENT_TIMESTAMP WHERE id = $1`
	if _, err := r.db.Exec(updateQuery, sessionID); err != nil {
		return fmt.Errorf("failed to update session timestamp: %w", err)
	}

	return nil
}

// ListBySession retrieves all messages for a session
func (r *MessageRepository) ListBySession(sessionID string) ([]providers.Message, error) {
	var dbMessages []Message
	query := `SELECT * FROM messages WHERE session_id = $1 ORDER BY created_at ASC`
	
	if err := r.db.Select(&dbMessages, query, sessionID); err != nil {
		return nil, fmt.Errorf("failed to list messages: %w", err)
	}

	// Convert to provider messages
	messages := make([]providers.Message, len(dbMessages))
	for i, dbMsg := range dbMessages {
		msg := providers.Message{
			Role:    dbMsg.Role,
			Content: dbMsg.Content,
		}

		// Handle optional fields
		if len(dbMsg.FunctionCall) > 0 && string(dbMsg.FunctionCall) != "null" {
			var funcCall providers.FunctionCall
			if err := json.Unmarshal(dbMsg.FunctionCall, &funcCall); err == nil {
				msg.FunctionCall = &funcCall
			}
		}

		if len(dbMsg.ToolCalls) > 0 && string(dbMsg.ToolCalls) != "null" {
			var toolCalls []providers.ToolCall
			if err := json.Unmarshal(dbMsg.ToolCalls, &toolCalls); err == nil {
				msg.ToolCalls = toolCalls
			}
		}

		if dbMsg.ToolCallID.Valid {
			msg.ToolCallID = dbMsg.ToolCallID.String
		}

		messages[i] = msg
	}

	return messages, nil
}

// DeleteBySession deletes all messages for a session
func (r *MessageRepository) DeleteBySession(sessionID string) error {
	query := `DELETE FROM messages WHERE session_id = $1`
	
	_, err := r.db.Exec(query, sessionID)
	if err != nil {
		return fmt.Errorf("failed to delete messages: %w", err)
	}

	return nil
}

// CountBySession returns the number of messages in a session
func (r *MessageRepository) CountBySession(sessionID string) (int, error) {
	var count int
	query := `SELECT COUNT(*) FROM messages WHERE session_id = $1`
	
	if err := r.db.Get(&count, query, sessionID); err != nil {
		return 0, fmt.Errorf("failed to count messages: %w", err)
	}

	return count, nil
}