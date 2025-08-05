package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/agentx/agentx-backend/internal/repository"
)

// connectionRepository implements repository.ConnectionRepository
type connectionRepository struct {
	db *sqlx.DB
}

// NewConnectionRepository creates a new PostgreSQL connection repository
func NewConnectionRepository(db *sqlx.DB) repository.ConnectionRepository {
	return &connectionRepository{db: db}
}

// Create creates a new provider connection
func (r *connectionRepository) Create(ctx context.Context, providerID, name string, config map[string]interface{}) (string, error) {
	id := uuid.New().String()
	
	configJSON, err := json.Marshal(config)
	if err != nil {
		return "", fmt.Errorf("failed to marshal config: %w", err)
	}
	
	query := `
		INSERT INTO provider_connections (id, provider_id, name, config)
		VALUES ($1, $2, $3, $4)
		RETURNING id
	`
	
	var returnedID string
	err = r.db.QueryRowxContext(ctx, query, id, providerID, name, configJSON).Scan(&returnedID)
	if err != nil {
		return "", fmt.Errorf("failed to create connection: %w", err)
	}
	
	return returnedID, nil
}

// GetByID retrieves a connection by its ID
func (r *connectionRepository) GetByID(ctx context.Context, id string) (*repository.ProviderConnection, error) {
	query := `
		SELECT id, provider_id, name, enabled, config, metadata, created_at, updated_at
		FROM provider_connections
		WHERE id = $1
	`
	
	var conn dbConnection
	err := r.db.GetContext(ctx, &conn, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("connection not found")
		}
		return nil, fmt.Errorf("failed to get connection: %w", err)
	}
	
	return conn.toModel()
}

// GetByProviderID retrieves all connections for a specific provider
func (r *connectionRepository) GetByProviderID(ctx context.Context, providerID string) ([]*repository.ProviderConnection, error) {
	query := `
		SELECT id, provider_id, name, enabled, config, metadata, created_at, updated_at
		FROM provider_connections
		WHERE provider_id = $1
		ORDER BY created_at DESC
	`
	
	var dbConns []dbConnection
	err := r.db.SelectContext(ctx, &dbConns, query, providerID)
	if err != nil {
		return nil, fmt.Errorf("failed to list connections: %w", err)
	}
	
	connections := make([]*repository.ProviderConnection, 0, len(dbConns))
	for _, dbConn := range dbConns {
		conn, err := dbConn.toModel()
		if err != nil {
			return nil, err
		}
		connections = append(connections, conn)
	}
	
	return connections, nil
}

// List retrieves all connections
func (r *connectionRepository) List(ctx context.Context) ([]*repository.ProviderConnection, error) {
	query := `
		SELECT id, provider_id, name, enabled, config, metadata, created_at, updated_at
		FROM provider_connections
		ORDER BY provider_id, created_at DESC
	`
	
	var dbConns []dbConnection
	err := r.db.SelectContext(ctx, &dbConns, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list connections: %w", err)
	}
	
	connections := make([]*repository.ProviderConnection, 0, len(dbConns))
	for _, dbConn := range dbConns {
		conn, err := dbConn.toModel()
		if err != nil {
			return nil, err
		}
		connections = append(connections, conn)
	}
	
	return connections, nil
}

// Update updates a connection
func (r *connectionRepository) Update(ctx context.Context, id string, updates map[string]interface{}) error {
	// Build dynamic update query
	setClauses := make([]string, 0)
	args := make([]interface{}, 0)
	argCount := 1
	
	for key, value := range updates {
		switch key {
		case "name":
			setClauses = append(setClauses, fmt.Sprintf("name = $%d", argCount))
			args = append(args, value)
			argCount++
		case "enabled":
			setClauses = append(setClauses, fmt.Sprintf("enabled = $%d", argCount))
			args = append(args, value)
			argCount++
		case "config":
			configJSON, err := json.Marshal(value)
			if err != nil {
				return fmt.Errorf("failed to marshal config: %w", err)
			}
			setClauses = append(setClauses, fmt.Sprintf("config = $%d", argCount))
			args = append(args, configJSON)
			argCount++
		case "metadata":
			metadataJSON, err := json.Marshal(value)
			if err != nil {
				return fmt.Errorf("failed to marshal metadata: %w", err)
			}
			setClauses = append(setClauses, fmt.Sprintf("metadata = $%d", argCount))
			args = append(args, metadataJSON)
			argCount++
		}
	}
	
	if len(setClauses) == 0 {
		return nil // Nothing to update
	}
	
	// Add ID as the last argument
	args = append(args, id)
	
	query := fmt.Sprintf(`
		UPDATE provider_connections
		SET %s
		WHERE id = $%d
	`, joinStrings(setClauses, ", "), argCount)
	
	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to update connection: %w", err)
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	
	if rowsAffected == 0 {
		return fmt.Errorf("connection not found")
	}
	
	return nil
}

// Delete deletes a connection
func (r *connectionRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM provider_connections WHERE id = $1`
	
	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete connection: %w", err)
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	
	if rowsAffected == 0 {
		return fmt.Errorf("connection not found")
	}
	
	return nil
}

// SetDefault sets the default connection for a provider
func (r *connectionRepository) SetDefault(ctx context.Context, providerID string, connectionID string) error {
	query := `
		INSERT INTO default_connections (provider_id, connection_id)
		VALUES ($1, $2)
		ON CONFLICT (provider_id) DO UPDATE
		SET connection_id = EXCLUDED.connection_id
	`
	
	_, err := r.db.ExecContext(ctx, query, providerID, connectionID)
	if err != nil {
		return fmt.Errorf("failed to set default connection: %w", err)
	}
	
	return nil
}

// GetDefault gets the default connection for a provider
func (r *connectionRepository) GetDefault(ctx context.Context, providerID string) (*repository.ProviderConnection, error) {
	query := `
		SELECT pc.id, pc.provider_id, pc.name, pc.enabled, pc.config, pc.metadata, pc.created_at, pc.updated_at
		FROM provider_connections pc
		JOIN default_connections dc ON pc.id = dc.connection_id
		WHERE dc.provider_id = $1 AND pc.enabled = true
	`
	
	var conn dbConnection
	err := r.db.GetContext(ctx, &conn, query, providerID)
	if err != nil {
		if err == sql.ErrNoRows {
			// No default set, return any enabled connection for the provider
			return r.getFirstEnabled(ctx, providerID)
		}
		return nil, fmt.Errorf("failed to get default connection: %w", err)
	}
	
	return conn.toModel()
}

// getFirstEnabled returns the first enabled connection for a provider
func (r *connectionRepository) getFirstEnabled(ctx context.Context, providerID string) (*repository.ProviderConnection, error) {
	query := `
		SELECT id, provider_id, name, enabled, config, metadata, created_at, updated_at
		FROM provider_connections
		WHERE provider_id = $1 AND enabled = true
		ORDER BY created_at ASC
		LIMIT 1
	`
	
	var conn dbConnection
	err := r.db.GetContext(ctx, &conn, query, providerID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("no enabled connections for provider %s", providerID)
		}
		return nil, fmt.Errorf("failed to get connection: %w", err)
	}
	
	return conn.toModel()
}

// dbConnection represents the database structure
type dbConnection struct {
	ID         string         `db:"id"`
	ProviderID string         `db:"provider_id"`
	Name       string         `db:"name"`
	Enabled    bool           `db:"enabled"`
	Config     []byte         `db:"config"`
	Metadata   sql.NullString `db:"metadata"`
	CreatedAt  sql.NullTime   `db:"created_at"`
	UpdatedAt  sql.NullTime   `db:"updated_at"`
}

// toModel converts the database structure to the repository model
func (c *dbConnection) toModel() (*repository.ProviderConnection, error) {
	conn := &repository.ProviderConnection{
		ID:         c.ID,
		ProviderID: c.ProviderID,
		Name:       c.Name,
		Enabled:    c.Enabled,
		CreatedAt:  c.CreatedAt.Time,
		UpdatedAt:  c.UpdatedAt.Time,
	}
	
	// Unmarshal config
	if err := json.Unmarshal(c.Config, &conn.Config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}
	
	// Unmarshal metadata if present
	if c.Metadata.Valid {
		if err := json.Unmarshal([]byte(c.Metadata.String), &conn.Metadata); err != nil {
			return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
		}
	} else {
		conn.Metadata = make(map[string]interface{})
	}
	
	return conn, nil
}

// Helper function to join strings
func joinStrings(strs []string, sep string) string {
	result := ""
	for i, s := range strs {
		if i > 0 {
			result += sep
		}
		result += s
	}
	return result
}