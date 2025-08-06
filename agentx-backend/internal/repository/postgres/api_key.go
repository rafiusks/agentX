package postgres

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/agentx/agentx-backend/internal/models"
)

// APIKeyRepository implements auth.APIKeyRepository
type APIKeyRepository struct {
	db *sqlx.DB
}

// NewAPIKeyRepository creates a new API key repository
func NewAPIKeyRepository(db *sqlx.DB) *APIKeyRepository {
	return &APIKeyRepository{db: db}
}

// Create creates a new API key
func (r *APIKeyRepository) Create(ctx context.Context, apiKey *models.APIKey) error {
	query := `
		INSERT INTO api_keys (
			id, user_id, name, key_prefix, key_hash, scopes,
			expires_at, is_active, created_at, metadata
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10
		)`

	_, err := r.db.ExecContext(ctx, query,
		apiKey.ID, apiKey.UserID, apiKey.Name, apiKey.KeyPrefix, apiKey.KeyHash,
		apiKey.Scopes, apiKey.ExpiresAt, apiKey.IsActive, apiKey.CreatedAt, apiKey.Metadata,
	)
	return err
}

// GetByID retrieves an API key by ID
func (r *APIKeyRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.APIKey, error) {
	var apiKey models.APIKey
	query := `SELECT * FROM api_keys WHERE id = $1`

	err := r.db.GetContext(ctx, &apiKey, query, id)
	if err != nil {
		return nil, err
	}
	return &apiKey, nil
}

// GetByKeyHash retrieves an API key by its hash
func (r *APIKeyRepository) GetByKeyHash(ctx context.Context, keyHash string) (*models.APIKey, error) {
	var apiKey models.APIKey
	query := `SELECT * FROM api_keys WHERE key_hash = $1 AND is_active = true`

	err := r.db.GetContext(ctx, &apiKey, query, keyHash)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, sql.ErrNoRows
		}
		return nil, err
	}
	return &apiKey, nil
}

// ListByUserID lists all API keys for a user
func (r *APIKeyRepository) ListByUserID(ctx context.Context, userID uuid.UUID) ([]*models.APIKey, error) {
	var apiKeys []*models.APIKey
	query := `
		SELECT * FROM api_keys 
		WHERE user_id = $1
		ORDER BY created_at DESC`

	err := r.db.SelectContext(ctx, &apiKeys, query, userID)
	return apiKeys, err
}

// UpdateLastUsed updates the last used timestamp
func (r *APIKeyRepository) UpdateLastUsed(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE api_keys SET last_used = $1 WHERE id = $2`
	_, err := r.db.ExecContext(ctx, query, time.Now(), id)
	return err
}

// Delete deletes an API key
func (r *APIKeyRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM api_keys WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}