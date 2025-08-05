package postgres

import (
	"context"
	"database/sql"
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
func (r *SessionRepository) Create(ctx context.Context, session repository.Session) (string, error) {
	session.ID = uuid.New().String()
	session.CreatedAt = time.Now()
	session.UpdatedAt = time.Now()
	
	if len(session.Metadata) == 0 {
		session.Metadata = []byte("{}")
	}
	
	query := `
		INSERT INTO sessions (id, title, provider, model, created_at, updated_at, metadata)
		VALUES (:id, :title, :provider, :model, :created_at, :updated_at, :metadata)
	`
	
	_, err := r.db.NamedExecContext(ctx, query, session)
	if err != nil {
		return "", err
	}
	
	return session.ID, nil
}

// Get retrieves a session by ID
func (r *SessionRepository) Get(ctx context.Context, id string) (*repository.Session, error) {
	var session repository.Session
	query := `
		SELECT id, title, provider, model, created_at, updated_at, metadata
		FROM sessions
		WHERE id = $1
	`
	
	err := r.db.GetContext(ctx, &session, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	
	return &session, nil
}

// List retrieves all sessions
func (r *SessionRepository) List(ctx context.Context) ([]*repository.Session, error) {
	var sessions []*repository.Session
	query := `
		SELECT id, title, provider, model, created_at, updated_at, metadata
		FROM sessions
		ORDER BY updated_at DESC
	`
	
	err := r.db.SelectContext(ctx, &sessions, query)
	if err != nil {
		return nil, err
	}
	
	return sessions, nil
}

// Update updates a session
func (r *SessionRepository) Update(ctx context.Context, id string, updates map[string]interface{}) error {
	updates["updated_at"] = time.Now()
	
	// Build dynamic update query
	setClause := ""
	params := map[string]interface{}{"id": id}
	
	for key, value := range updates {
		if setClause != "" {
			setClause += ", "
		}
		setClause += key + " = :" + key
		params[key] = value
	}
	
	query := "UPDATE sessions SET " + setClause + " WHERE id = :id"
	
	_, err := r.db.NamedExecContext(ctx, query, params)
	return err
}

// Delete deletes a session
func (r *SessionRepository) Delete(ctx context.Context, id string) error {
	query := "DELETE FROM sessions WHERE id = $1"
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}