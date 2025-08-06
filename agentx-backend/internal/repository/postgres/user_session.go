package postgres

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/agentx/agentx-backend/internal/models"
)

// UserSessionRepository implements auth.SessionRepository
type UserSessionRepository struct {
	db *sqlx.DB
}

// NewUserSessionRepository creates a new user session repository
func NewUserSessionRepository(db *sqlx.DB) *UserSessionRepository {
	return &UserSessionRepository{db: db}
}

// Create creates a new user session
func (r *UserSessionRepository) Create(ctx context.Context, session *models.UserSession) error {
	query := `
		INSERT INTO user_sessions (
			id, user_id, token_hash, refresh_token_hash, expires_at, refresh_expires_at,
			ip_address, user_agent, device_name, created_at, last_activity
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11
		)`

	_, err := r.db.ExecContext(ctx, query,
		session.ID, session.UserID, session.TokenHash, session.RefreshTokenHash,
		session.ExpiresAt, session.RefreshExpiresAt, session.IPAddress,
		session.UserAgent, session.DeviceName, session.CreatedAt, session.LastActivity,
	)
	return err
}

// GetByID retrieves a session by ID
func (r *UserSessionRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.UserSession, error) {
	var session models.UserSession
	query := `SELECT * FROM user_sessions WHERE id = $1`

	err := r.db.GetContext(ctx, &session, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, sql.ErrNoRows
		}
		return nil, err
	}
	return &session, nil
}

// GetByTokenHash retrieves a session by token hash
func (r *UserSessionRepository) GetByTokenHash(ctx context.Context, tokenHash string) (*models.UserSession, error) {
	var session models.UserSession
	query := `SELECT * FROM user_sessions WHERE token_hash = $1`

	err := r.db.GetContext(ctx, &session, query, tokenHash)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, sql.ErrNoRows
		}
		return nil, err
	}
	return &session, nil
}

// GetByRefreshTokenHash retrieves a session by refresh token hash
func (r *UserSessionRepository) GetByRefreshTokenHash(ctx context.Context, refreshTokenHash string) (*models.UserSession, error) {
	var session models.UserSession
	query := `SELECT * FROM user_sessions WHERE refresh_token_hash = $1`

	err := r.db.GetContext(ctx, &session, query, refreshTokenHash)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, sql.ErrNoRows
		}
		return nil, err
	}
	return &session, nil
}

// Update updates a session
func (r *UserSessionRepository) Update(ctx context.Context, session *models.UserSession) error {
	query := `
		UPDATE user_sessions SET
			token_hash = $1, refresh_token_hash = $2, expires_at = $3,
			refresh_expires_at = $4, last_activity = $5, revoked_at = $6
		WHERE id = $7`

	_, err := r.db.ExecContext(ctx, query,
		session.TokenHash, session.RefreshTokenHash, session.ExpiresAt,
		session.RefreshExpiresAt, session.LastActivity, session.RevokedAt, session.ID,
	)
	return err
}

// Delete deletes a session
func (r *UserSessionRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM user_sessions WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

// DeleteExpired deletes expired sessions
func (r *UserSessionRepository) DeleteExpired(ctx context.Context) error {
	query := `
		DELETE FROM user_sessions 
		WHERE refresh_expires_at < $1 OR revoked_at IS NOT NULL`
	_, err := r.db.ExecContext(ctx, query, time.Now())
	return err
}

// DeleteUserSessions deletes all sessions for a user
func (r *UserSessionRepository) DeleteUserSessions(ctx context.Context, userID uuid.UUID) error {
	query := `DELETE FROM user_sessions WHERE user_id = $1`
	_, err := r.db.ExecContext(ctx, query, userID)
	return err
}