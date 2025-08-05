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

// Session represents a chat session in the database
type Session struct {
	ID        string                 `db:"id"`
	Title     string                 `db:"title"`
	Provider  sql.NullString         `db:"provider"`
	Model     sql.NullString         `db:"model"`
	CreatedAt time.Time              `db:"created_at"`
	UpdatedAt time.Time              `db:"updated_at"`
	Metadata  json.RawMessage        `db:"metadata"`
}

// SessionRepository handles session database operations
type SessionRepository struct {
	db *sqlx.DB
}

// NewSessionRepository creates a new session repository
func NewSessionRepository(db *sqlx.DB) *SessionRepository {
	return &SessionRepository{db: db}
}

// Create creates a new session
func (r *SessionRepository) Create(title string) (*Session, error) {
	session := &Session{
		ID:        uuid.New().String(),
		Title:     title,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Metadata:  json.RawMessage("{}"),
	}

	query := `
		INSERT INTO sessions (id, title, created_at, updated_at, metadata)
		VALUES (:id, :title, :created_at, :updated_at, :metadata)
		RETURNING *`

	rows, err := r.db.NamedQuery(query, session)
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}
	defer rows.Close()

	if rows.Next() {
		if err := rows.StructScan(session); err != nil {
			return nil, fmt.Errorf("failed to scan session: %w", err)
		}
	}

	return session, nil
}

// Get retrieves a session by ID
func (r *SessionRepository) Get(id string) (*Session, error) {
	var session Session
	query := `SELECT * FROM sessions WHERE id = $1`
	
	if err := r.db.Get(&session, query, id); err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("session not found")
		}
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	return &session, nil
}

// List retrieves all sessions ordered by updated_at desc
func (r *SessionRepository) List() ([]*Session, error) {
	var sessions []*Session
	query := `SELECT * FROM sessions ORDER BY updated_at DESC`
	
	if err := r.db.Select(&sessions, query); err != nil {
		return nil, fmt.Errorf("failed to list sessions: %w", err)
	}

	return sessions, nil
}

// Update updates a session
func (r *SessionRepository) Update(session *Session) error {
	query := `
		UPDATE sessions 
		SET title = :title, provider = :provider, model = :model, 
		    metadata = :metadata, updated_at = CURRENT_TIMESTAMP
		WHERE id = :id`

	_, err := r.db.NamedExec(query, session)
	if err != nil {
		return fmt.Errorf("failed to update session: %w", err)
	}

	return nil
}

// Delete deletes a session
func (r *SessionRepository) Delete(id string) error {
	query := `DELETE FROM sessions WHERE id = $1`
	
	result, err := r.db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("session not found")
	}

	return nil
}

// GetWithMessages retrieves a session with all its messages
func (r *SessionRepository) GetWithMessages(id string) (*Session, []providers.Message, error) {
	session, err := r.Get(id)
	if err != nil {
		return nil, nil, err
	}

	// Get messages
	messageRepo := NewMessageRepository(r.db)
	messages, err := messageRepo.ListBySession(id)
	if err != nil {
		return nil, nil, err
	}

	return session, messages, nil
}