package postgres

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/agentx/agentx-backend/internal/repository"
)

// MessageRepository implements repository.MessageRepository using PostgreSQL
type MessageRepository struct {
	db *sqlx.DB
}

// NewMessageRepository creates a new PostgreSQL message repository
func NewMessageRepository(db *sqlx.DB) repository.MessageRepository {
	return &MessageRepository{db: db}
}

// Create creates a new message
func (r *MessageRepository) Create(ctx context.Context, message repository.Message) (string, error) {
	message.ID = uuid.New().String()
	message.CreatedAt = time.Now()
	
	if len(message.Metadata) == 0 {
		message.Metadata = []byte("{}")
	}
	
	query := `
		INSERT INTO messages (id, session_id, role, content, function_call, tool_calls, tool_call_id, created_at, metadata)
		VALUES (:id, :session_id, :role, :content, :function_call, :tool_calls, :tool_call_id, :created_at, :metadata)
	`
	
	_, err := r.db.NamedExecContext(ctx, query, message)
	if err != nil {
		return "", err
	}
	
	return message.ID, nil
}

// ListBySession retrieves messages for a session
func (r *MessageRepository) ListBySession(ctx context.Context, sessionID string) ([]repository.Message, error) {
	var messages []repository.Message
	query := `
		SELECT id, session_id, role, content, function_call, tool_calls, tool_call_id, created_at, metadata
		FROM messages
		WHERE session_id = $1
		ORDER BY created_at ASC
	`
	
	err := r.db.SelectContext(ctx, &messages, query, sessionID)
	if err != nil {
		return nil, err
	}
	
	return messages, nil
}

// Delete deletes a message
func (r *MessageRepository) Delete(ctx context.Context, id string) error {
	query := "DELETE FROM messages WHERE id = $1"
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}