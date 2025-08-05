package repositories

import (
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/jmoiron/sqlx"
	"github.com/agentx/agentx-backend/internal/config"
)

// ProviderConfigDB represents a provider configuration in the database
type ProviderConfigDB struct {
	ID           string          `db:"id"`
	Type         string          `db:"type"`
	Name         string          `db:"name"`
	BaseURL      sql.NullString  `db:"base_url"`
	APIKey       sql.NullString  `db:"api_key"`
	Models       json.RawMessage `db:"models"`
	DefaultModel sql.NullString  `db:"default_model"`
	Extra        json.RawMessage `db:"extra"`
	IsActive     bool            `db:"is_active"`
}

// ProviderConfigRepository handles provider configuration database operations
type ProviderConfigRepository struct {
	db *sqlx.DB
}

// NewProviderConfigRepository creates a new provider config repository
func NewProviderConfigRepository(db *sqlx.DB) *ProviderConfigRepository {
	return &ProviderConfigRepository{db: db}
}

// GetAll retrieves all active provider configurations
func (r *ProviderConfigRepository) GetAll() (map[string]config.ProviderConfig, error) {
	var configs []ProviderConfigDB
	query := `SELECT * FROM provider_configs WHERE is_active = true`
	
	if err := r.db.Select(&configs, query); err != nil {
		return nil, fmt.Errorf("failed to get provider configs: %w", err)
	}

	result := make(map[string]config.ProviderConfig)
	for _, cfg := range configs {
		providerCfg := config.ProviderConfig{
			Type: cfg.Type,
			Name: cfg.Name,
		}

		// Handle nullable fields
		if cfg.BaseURL.Valid {
			providerCfg.BaseURL = cfg.BaseURL.String
		}
		if cfg.APIKey.Valid {
			providerCfg.APIKey = cfg.APIKey.String
		}
		if cfg.DefaultModel.Valid {
			providerCfg.DefaultModel = cfg.DefaultModel.String
		}

		// Parse JSON fields
		if len(cfg.Models) > 0 {
			json.Unmarshal(cfg.Models, &providerCfg.Models)
		}
		if len(cfg.Extra) > 0 {
			json.Unmarshal(cfg.Extra, &providerCfg.Extra)
		}

		result[cfg.ID] = providerCfg
	}

	return result, nil
}

// Get retrieves a specific provider configuration
func (r *ProviderConfigRepository) Get(id string) (*config.ProviderConfig, error) {
	var cfg ProviderConfigDB
	query := `SELECT * FROM provider_configs WHERE id = $1 AND is_active = true`
	
	if err := r.db.Get(&cfg, query, id); err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("provider config not found")
		}
		return nil, fmt.Errorf("failed to get provider config: %w", err)
	}

	result := &config.ProviderConfig{
		Type: cfg.Type,
		Name: cfg.Name,
	}

	// Handle nullable fields
	if cfg.BaseURL.Valid {
		result.BaseURL = cfg.BaseURL.String
	}
	if cfg.APIKey.Valid {
		result.APIKey = cfg.APIKey.String
	}
	if cfg.DefaultModel.Valid {
		result.DefaultModel = cfg.DefaultModel.String
	}

	// Parse JSON fields
	if len(cfg.Models) > 0 {
		json.Unmarshal(cfg.Models, &result.Models)
	}
	if len(cfg.Extra) > 0 {
		json.Unmarshal(cfg.Extra, &result.Extra)
	}

	return result, nil
}

// Update updates a provider configuration
func (r *ProviderConfigRepository) Update(id string, cfg config.ProviderConfig) error {
	// Convert to DB format
	dbCfg := ProviderConfigDB{
		ID:   id,
		Type: cfg.Type,
		Name: cfg.Name,
	}

	// Handle nullable fields
	if cfg.BaseURL != "" {
		dbCfg.BaseURL = sql.NullString{String: cfg.BaseURL, Valid: true}
	}
	if cfg.APIKey != "" {
		dbCfg.APIKey = sql.NullString{String: cfg.APIKey, Valid: true}
	}
	if cfg.DefaultModel != "" {
		dbCfg.DefaultModel = sql.NullString{String: cfg.DefaultModel, Valid: true}
	}

	// Convert JSON fields
	if cfg.Models != nil {
		modelsJSON, _ := json.Marshal(cfg.Models)
		dbCfg.Models = modelsJSON
	} else {
		dbCfg.Models = json.RawMessage("[]")
	}

	if cfg.Extra != nil {
		extraJSON, _ := json.Marshal(cfg.Extra)
		dbCfg.Extra = extraJSON
	} else {
		dbCfg.Extra = json.RawMessage("{}")
	}

	query := `
		UPDATE provider_configs 
		SET type = :type, name = :name, base_url = :base_url, 
		    api_key = :api_key, models = :models, default_model = :default_model,
		    extra = :extra, updated_at = CURRENT_TIMESTAMP
		WHERE id = :id`

	_, err := r.db.NamedExec(query, dbCfg)
	if err != nil {
		return fmt.Errorf("failed to update provider config: %w", err)
	}

	return nil
}

// Create creates a new provider configuration
func (r *ProviderConfigRepository) Create(id string, cfg config.ProviderConfig) error {
	// Convert to DB format
	dbCfg := ProviderConfigDB{
		ID:       id,
		Type:     cfg.Type,
		Name:     cfg.Name,
		IsActive: true,
	}

	// Handle nullable fields
	if cfg.BaseURL != "" {
		dbCfg.BaseURL = sql.NullString{String: cfg.BaseURL, Valid: true}
	}
	if cfg.APIKey != "" {
		dbCfg.APIKey = sql.NullString{String: cfg.APIKey, Valid: true}
	}
	if cfg.DefaultModel != "" {
		dbCfg.DefaultModel = sql.NullString{String: cfg.DefaultModel, Valid: true}
	}

	// Convert JSON fields
	if cfg.Models != nil {
		modelsJSON, _ := json.Marshal(cfg.Models)
		dbCfg.Models = modelsJSON
	} else {
		dbCfg.Models = json.RawMessage("[]")
	}

	if cfg.Extra != nil {
		extraJSON, _ := json.Marshal(cfg.Extra)
		dbCfg.Extra = extraJSON
	} else {
		dbCfg.Extra = json.RawMessage("{}")
	}

	query := `
		INSERT INTO provider_configs 
		(id, type, name, base_url, api_key, models, default_model, extra, is_active)
		VALUES (:id, :type, :name, :base_url, :api_key, :models, :default_model, :extra, :is_active)`

	_, err := r.db.NamedExec(query, dbCfg)
	if err != nil {
		return fmt.Errorf("failed to create provider config: %w", err)
	}

	return nil
}

// Delete soft deletes a provider configuration
func (r *ProviderConfigRepository) Delete(id string) error {
	query := `UPDATE provider_configs SET is_active = false WHERE id = $1`
	
	_, err := r.db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete provider config: %w", err)
	}

	return nil
}

// GetAppSetting retrieves an app setting
func (r *ProviderConfigRepository) GetAppSetting(key string) (string, error) {
	var value json.RawMessage
	query := `SELECT value FROM app_settings WHERE key = $1`
	
	if err := r.db.Get(&value, query, key); err != nil {
		if err == sql.ErrNoRows {
			return "", nil
		}
		return "", fmt.Errorf("failed to get app setting: %w", err)
	}

	// Unmarshal the JSON value
	var result string
	if err := json.Unmarshal(value, &result); err != nil {
		return "", fmt.Errorf("failed to unmarshal app setting: %w", err)
	}

	return result, nil
}

// SetAppSetting sets an app setting
func (r *ProviderConfigRepository) SetAppSetting(key, value string) error {
	// Marshal the value as JSON
	valueJSON, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal app setting: %w", err)
	}

	query := `
		INSERT INTO app_settings (key, value) 
		VALUES ($1, $2)
		ON CONFLICT (key) DO UPDATE 
		SET value = $2, updated_at = CURRENT_TIMESTAMP`
	
	_, err = r.db.Exec(query, key, valueJSON)
	if err != nil {
		return fmt.Errorf("failed to set app setting: %w", err)
	}

	return nil
}