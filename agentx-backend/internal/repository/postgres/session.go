package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/agentx/agentx-backend/internal/repository"
)

// SessionRepository implements repository.SessionRepository using PostgreSQL
type SessionRepository struct {
	db *sqlx.DB
}

// NewSessionRepository creates a new PostgreSQL session repository
func NewSessionRepository(db *sqlx.DB) repository.SessionRepository {
	return &SessionRepository{db: db}
}

// Create creates a new session
func (r *SessionRepository) Create(ctx context.Context, session *repository.Session) error {
	if session.ID == "" {
		session.ID = uuid.New().String()
	}
	session.CreatedAt = time.Now()
	session.UpdatedAt = time.Now()
	
	if len(session.Metadata) == 0 {
		session.Metadata = []byte("{}")
	}
	
	query := `
		INSERT INTO sessions (id, user_id, title, provider, model, created_at, updated_at, metadata)
		VALUES (:id, :user_id, :title, :provider, :model, :created_at, :updated_at, :metadata)
	`
	
	_, err := r.db.NamedExecContext(ctx, query, session)
	return err
}

// Get retrieves a session by ID
func (r *SessionRepository) Get(ctx context.Context, userID uuid.UUID, id string) (*repository.Session, error) {
	var session repository.Session
	query := `
		SELECT id, user_id, title, provider, model, created_at, updated_at, metadata
		FROM sessions
		WHERE id = $1 AND user_id = $2
	`
	
	err := r.db.GetContext(ctx, &session, query, id, userID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	
	return &session, nil
}

// List retrieves all sessions
func (r *SessionRepository) List(ctx context.Context, userID uuid.UUID) ([]*repository.Session, error) {
	var sessions []*repository.Session
	query := `
		SELECT id, user_id, title, provider, model, created_at, updated_at, metadata
		FROM sessions
		WHERE user_id = $1
		ORDER BY updated_at DESC
	`
	
	err := r.db.SelectContext(ctx, &sessions, query, userID)
	if err != nil {
		return nil, err
	}
	
	return sessions, nil
}

// Update updates a session
func (r *SessionRepository) Update(ctx context.Context, userID uuid.UUID, id string, updates map[string]interface{}) error {
	now := time.Now()
	updates["updated_at"] = now
	
	fmt.Printf("[SessionRepository.Update] Updating session %s for user %s at %v\n", id, userID.String(), now)
	
	// Build dynamic update query
	setClause := ""
	params := map[string]interface{}{"id": id, "user_id": userID}
	
	for key, value := range updates {
		if setClause != "" {
			setClause += ", "
		}
		setClause += key + " = :" + key
		params[key] = value
	}
	
	query := "UPDATE sessions SET " + setClause + " WHERE id = :id AND user_id = :user_id"
	fmt.Printf("[SessionRepository.Update] Query: %s\n", query)
	fmt.Printf("[SessionRepository.Update] Params: %+v\n", params)
	
	result, err := r.db.NamedExecContext(ctx, query, params)
	if err != nil {
		fmt.Printf("[SessionRepository.Update] Error: %v\n", err)
		return err
	}
	
	rowsAffected, _ := result.RowsAffected()
	fmt.Printf("[SessionRepository.Update] Rows affected: %d\n", rowsAffected)
	
	return nil
}

// Delete deletes a session
func (r *SessionRepository) Delete(ctx context.Context, userID uuid.UUID, id string) error {
	query := "DELETE FROM sessions WHERE id = $1 AND user_id = $2"
	_, err := r.db.ExecContext(ctx, query, id, userID)
	return err
}