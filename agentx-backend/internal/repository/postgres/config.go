package postgres

import (
	"context"
	"database/sql"
	"encoding/json"

	"github.com/jmoiron/sqlx"
	"github.com/agentx/agentx-backend/internal/repository"
)

// ConfigRepository implements repository.ConfigRepository using PostgreSQL
type ConfigRepository struct {
	db *sqlx.DB
}

// NewConfigRepository creates a new PostgreSQL config repository
func NewConfigRepository(db *sqlx.DB) repository.ConfigRepository {
	return &ConfigRepository{db: db}
}

// Get retrieves a configuration value
func (r *ConfigRepository) Get(ctx context.Context, key string) (interface{}, error) {
	var value json.RawMessage
	query := "SELECT value FROM configs WHERE key = $1"
	
	err := r.db.GetContext(ctx, &value, query, key)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	
	var result interface{}
	err = json.Unmarshal(value, &result)
	return result, err
}

// Set stores a configuration value
func (r *ConfigRepository) Set(ctx context.Context, key string, value interface{}) error {
	jsonValue, err := json.Marshal(value)
	if err != nil {
		return err
	}
	
	query := `
		INSERT INTO configs (key, value)
		VALUES ($1, $2)
		ON CONFLICT (key) DO UPDATE
		SET value = EXCLUDED.value
	`
	
	_, err = r.db.ExecContext(ctx, query, key, jsonValue)
	return err
}

// GetAll retrieves all configuration values
func (r *ConfigRepository) GetAll(ctx context.Context) (map[string]interface{}, error) {
	rows, err := r.db.QueryContext(ctx, "SELECT key, value FROM configs")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	configs := make(map[string]interface{})
	for rows.Next() {
		var key string
		var value json.RawMessage
		
		if err := rows.Scan(&key, &value); err != nil {
			return nil, err
		}
		
		var parsed interface{}
		if err := json.Unmarshal(value, &parsed); err != nil {
			return nil, err
		}
		
		configs[key] = parsed
	}
	
	return configs, rows.Err()
}

// Delete removes a configuration value
func (r *ConfigRepository) Delete(ctx context.Context, key string) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM configs WHERE key = $1", key)
	return err
}