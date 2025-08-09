package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// ContextMemory represents a stored memory item
type ContextMemory struct {
	ID           string          `db:"id" json:"id"`
	UserID       string          `db:"user_id" json:"user_id"`
	ProjectID    *string         `db:"project_id" json:"project_id,omitempty"`
	Namespace    string          `db:"namespace" json:"namespace"`
	Key          string          `db:"key" json:"key"`
	Value        json.RawMessage `db:"value" json:"value"`
	Importance   float32         `db:"importance" json:"importance"`
	AccessCount  int             `db:"access_count" json:"access_count"`
	LastAccessed time.Time       `db:"last_accessed" json:"last_accessed"`
	ExpiresAt    *time.Time      `db:"expires_at" json:"expires_at,omitempty"`
	Metadata     json.RawMessage `db:"metadata" json:"metadata"`
	CreatedAt    time.Time       `db:"created_at" json:"created_at"`
	UpdatedAt    time.Time       `db:"updated_at" json:"updated_at"`
}

// ContextMemoryService manages persistent context across conversations
type ContextMemoryService struct {
	db *sqlx.DB
}

// NewContextMemoryService creates a new context memory service
func NewContextMemoryService(db *sqlx.DB) *ContextMemoryService {
	return &ContextMemoryService{
		db: db,
	}
}

// Store saves or updates a memory item
func (s *ContextMemoryService) Store(ctx context.Context, userID string, namespace string, key string, value interface{}) error {
	valueJSON, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal value: %w", err)
	}

	query := `
		INSERT INTO context_memory (user_id, namespace, key, value, metadata)
		VALUES ($1, $2, $3, $4, '{}')
		ON CONFLICT (user_id, namespace, key)
		DO UPDATE SET 
			value = EXCLUDED.value,
			access_count = context_memory.access_count + 1,
			last_accessed = CURRENT_TIMESTAMP,
			updated_at = CURRENT_TIMESTAMP
	`

	_, err = s.db.ExecContext(ctx, query, userID, namespace, key, valueJSON)
	if err != nil {
		return fmt.Errorf("failed to store context memory: %w", err)
	}

	return nil
}

// StoreWithMetadata saves a memory item with additional metadata
func (s *ContextMemoryService) StoreWithMetadata(ctx context.Context, memory ContextMemory) error {
	if memory.ID == "" {
		memory.ID = uuid.New().String()
	}

	query := `
		INSERT INTO context_memory (
			id, user_id, project_id, namespace, key, value, 
			importance, metadata, expires_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9
		)
		ON CONFLICT (user_id, namespace, key)
		DO UPDATE SET 
			value = EXCLUDED.value,
			importance = EXCLUDED.importance,
			metadata = EXCLUDED.metadata,
			expires_at = EXCLUDED.expires_at,
			access_count = context_memory.access_count + 1,
			last_accessed = CURRENT_TIMESTAMP,
			updated_at = CURRENT_TIMESTAMP
	`

	_, err := s.db.ExecContext(ctx, query,
		memory.ID, memory.UserID, memory.ProjectID,
		memory.Namespace, memory.Key, memory.Value,
		memory.Importance, memory.Metadata, memory.ExpiresAt,
	)

	if err != nil {
		return fmt.Errorf("failed to store context memory with metadata: %w", err)
	}

	return nil
}

// Get retrieves a specific memory item
func (s *ContextMemoryService) Get(ctx context.Context, userID string, namespace string, key string) (*ContextMemory, error) {
	var memory ContextMemory
	
	query := `
		UPDATE context_memory 
		SET access_count = access_count + 1,
			last_accessed = CURRENT_TIMESTAMP
		WHERE user_id = $1 AND namespace = $2 AND key = $3
		RETURNING *
	`

	err := s.db.GetContext(ctx, &memory, query, userID, namespace, key)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get context memory: %w", err)
	}

	return &memory, nil
}

// GetByNamespace retrieves all memories in a namespace
func (s *ContextMemoryService) GetByNamespace(ctx context.Context, userID string, namespace string, limit int) ([]ContextMemory, error) {
	var memories []ContextMemory

	query := `
		SELECT * FROM context_memory
		WHERE user_id = $1 AND namespace = $2
			AND (expires_at IS NULL OR expires_at > CURRENT_TIMESTAMP)
		ORDER BY importance DESC, last_accessed DESC
		LIMIT $3
	`

	err := s.db.SelectContext(ctx, &memories, query, userID, namespace, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get memories by namespace: %w", err)
	}

	// Update access counts
	updateQuery := `
		UPDATE context_memory 
		SET access_count = access_count + 1,
			last_accessed = CURRENT_TIMESTAMP
		WHERE user_id = $1 AND namespace = $2 AND id = ANY($3)
	`
	
	ids := make([]string, len(memories))
	for i, m := range memories {
		ids[i] = m.ID
	}
	
	_, _ = s.db.ExecContext(ctx, updateQuery, userID, namespace, ids)

	return memories, nil
}

// GetRelevant retrieves memories relevant to current context
func (s *ContextMemoryService) GetRelevant(ctx context.Context, userID string, sessionID string, limit int) ([]ContextMemory, error) {
	var memories []ContextMemory

	// Get memories that have been referenced in recent sessions or are highly important
	query := `
		WITH recent_refs AS (
			SELECT DISTINCT cm.id
			FROM context_memory cm
			LEFT JOIN context_memory_refs cmr ON cm.id = cmr.memory_id
			LEFT JOIN sessions s ON cmr.session_id = s.id
			WHERE cm.user_id = $1
				AND (expires_at IS NULL OR expires_at > CURRENT_TIMESTAMP)
				AND (
					cm.importance > 0.7
					OR cmr.session_id = $2
					OR (s.updated_at > CURRENT_TIMESTAMP - INTERVAL '7 days')
				)
		)
		SELECT cm.* FROM context_memory cm
		INNER JOIN recent_refs rr ON cm.id = rr.id
		ORDER BY cm.importance DESC, cm.last_accessed DESC
		LIMIT $3
	`

	err := s.db.SelectContext(ctx, &memories, query, userID, sessionID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get relevant memories: %w", err)
	}

	return memories, nil
}

// Search finds memories matching a query
func (s *ContextMemoryService) Search(ctx context.Context, userID string, searchQuery string, limit int) ([]ContextMemory, error) {
	var memories []ContextMemory

	query := `
		SELECT * FROM context_memory
		WHERE user_id = $1
			AND (expires_at IS NULL OR expires_at > CURRENT_TIMESTAMP)
			AND (
				key ILIKE '%' || $2 || '%'
				OR value::text ILIKE '%' || $2 || '%'
			)
		ORDER BY importance DESC, last_accessed DESC
		LIMIT $3
	`

	err := s.db.SelectContext(ctx, &memories, query, userID, searchQuery, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to search memories: %w", err)
	}

	return memories, nil
}

// Delete removes a memory item
func (s *ContextMemoryService) Delete(ctx context.Context, userID string, namespace string, key string) error {
	query := `
		DELETE FROM context_memory
		WHERE user_id = $1 AND namespace = $2 AND key = $3
	`

	result, err := s.db.ExecContext(ctx, query, userID, namespace, key)
	if err != nil {
		return fmt.Errorf("failed to delete memory: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("memory not found")
	}

	return nil
}

// LinkToSession associates memories with a session
func (s *ContextMemoryService) LinkToSession(ctx context.Context, memoryID string, sessionID string, relevanceScore float32) error {
	query := `
		INSERT INTO context_memory_refs (memory_id, session_id, relevance_score)
		VALUES ($1, $2, $3)
		ON CONFLICT DO NOTHING
	`

	_, err := s.db.ExecContext(ctx, query, memoryID, sessionID, relevanceScore)
	if err != nil {
		return fmt.Errorf("failed to link memory to session: %w", err)
	}

	return nil
}

// LinkToMessage associates memories with a message
func (s *ContextMemoryService) LinkToMessage(ctx context.Context, memoryID string, messageID string, relevanceScore float32) error {
	query := `
		INSERT INTO context_memory_refs (memory_id, message_id, relevance_score)
		VALUES ($1, $2, $3)
		ON CONFLICT DO NOTHING
	`

	_, err := s.db.ExecContext(ctx, query, memoryID, messageID, relevanceScore)
	if err != nil {
		return fmt.Errorf("failed to link memory to message: %w", err)
	}

	return nil
}

// CleanupExpired removes expired memories
func (s *ContextMemoryService) CleanupExpired(ctx context.Context) (int64, error) {
	query := `
		DELETE FROM context_memory
		WHERE expires_at IS NOT NULL AND expires_at < CURRENT_TIMESTAMP
	`

	result, err := s.db.ExecContext(ctx, query)
	if err != nil {
		return 0, fmt.Errorf("failed to cleanup expired memories: %w", err)
	}

	return result.RowsAffected()
}

// UpdateImportance adjusts the importance of a memory
func (s *ContextMemoryService) UpdateImportance(ctx context.Context, memoryID string, importance float32) error {
	query := `
		UPDATE context_memory
		SET importance = $2, updated_at = CURRENT_TIMESTAMP
		WHERE id = $1
	`

	_, err := s.db.ExecContext(ctx, query, memoryID, importance)
	if err != nil {
		return fmt.Errorf("failed to update importance: %w", err)
	}

	return nil
}

// GetProjectMemories retrieves all memories for a specific project
func (s *ContextMemoryService) GetProjectMemories(ctx context.Context, userID string, projectID string, limit int) ([]ContextMemory, error) {
	var memories []ContextMemory

	query := `
		SELECT * FROM context_memory
		WHERE user_id = $1 AND project_id = $2
			AND (expires_at IS NULL OR expires_at > CURRENT_TIMESTAMP)
		ORDER BY importance DESC, last_accessed DESC
		LIMIT $3
	`

	err := s.db.SelectContext(ctx, &memories, query, userID, projectID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get project memories: %w", err)
	}

	return memories, nil
}