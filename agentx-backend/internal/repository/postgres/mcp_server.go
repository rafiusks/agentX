package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/agentx/agentx-backend/internal/models"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

// MCPServerRepository handles database operations for MCP servers
type MCPServerRepository struct {
	db *sqlx.DB
}

// NewMCPServerRepository creates a new MCPServerRepository
func NewMCPServerRepository(db *sqlx.DB) *MCPServerRepository {
	return &MCPServerRepository{db: db}
}

// Create creates a new MCP server configuration
func (r *MCPServerRepository) Create(ctx context.Context, userID uuid.UUID, req *models.MCPServerCreateRequest) (*models.MCPServer, error) {
	server := &models.MCPServer{
		ID:          uuid.New(),
		UserID:      userID,
		Name:        req.Name,
		Description: req.Description,
		Command:     req.Command,
		Args:        pq.StringArray(req.Args),
		Enabled:     req.Enabled,
		Status:      string(models.MCPServerStatusDisconnected),
	}

	// Convert env map to JSON
	if req.Env != nil {
		envJSON, err := json.Marshal(req.Env)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal env: %w", err)
		}
		server.Env = envJSON
	}

	query := `
		INSERT INTO mcp_servers (
			id, user_id, name, description, command, args, env, enabled, status
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9
		) RETURNING created_at, updated_at`

	err := r.db.QueryRowContext(
		ctx, query,
		server.ID, server.UserID, server.Name, server.Description,
		server.Command, server.Args, server.Env, server.Enabled, server.Status,
	).Scan(&server.CreatedAt, &server.UpdatedAt)

	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23505" {
			return nil, fmt.Errorf("MCP server with name '%s' already exists", req.Name)
		}
		return nil, fmt.Errorf("failed to create MCP server: %w", err)
	}

	return server, nil
}

// Get retrieves an MCP server by ID
func (r *MCPServerRepository) Get(ctx context.Context, userID, serverID uuid.UUID) (*models.MCPServer, error) {
	server := &models.MCPServer{}
	query := `
		SELECT id, user_id, name, description, command, args, env, enabled, status,
		       last_connected_at, capabilities, metadata, created_at, updated_at
		FROM mcp_servers
		WHERE id = $1 AND user_id = $2`

	err := r.db.GetContext(ctx, server, query, serverID, userID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("MCP server not found")
		}
		return nil, fmt.Errorf("failed to get MCP server: %w", err)
	}

	// Load tools for this server
	tools, err := r.GetServerTools(ctx, serverID)
	if err == nil {
		server.Tools = tools
	}

	// Load resources for this server
	resources, err := r.GetServerResources(ctx, serverID)
	if err == nil {
		server.Resources = resources
	}

	return server, nil
}

// List retrieves all MCP servers for a user
func (r *MCPServerRepository) List(ctx context.Context, userID uuid.UUID) ([]models.MCPServer, error) {
	query := `
		SELECT id, user_id, name, description, command, args, env, enabled, status,
		       last_connected_at, capabilities, metadata, created_at, updated_at
		FROM mcp_servers
		WHERE user_id = $1
		ORDER BY created_at DESC`

	servers := []models.MCPServer{}
	err := r.db.SelectContext(ctx, &servers, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to list MCP servers: %w", err)
	}

	return servers, nil
}

// Update updates an MCP server configuration
func (r *MCPServerRepository) Update(ctx context.Context, userID, serverID uuid.UUID, req *models.MCPServerUpdateRequest) (*models.MCPServer, error) {
	// Build dynamic update query
	updates := []string{}
	args := []interface{}{serverID, userID}
	argCount := 2

	if req.Name != "" {
		argCount++
		updates = append(updates, fmt.Sprintf("name = $%d", argCount))
		args = append(args, req.Name)
	}

	if req.Description != "" {
		argCount++
		updates = append(updates, fmt.Sprintf("description = $%d", argCount))
		args = append(args, req.Description)
	}

	if req.Command != "" {
		argCount++
		updates = append(updates, fmt.Sprintf("command = $%d", argCount))
		args = append(args, req.Command)
	}

	if req.Args != nil {
		argCount++
		updates = append(updates, fmt.Sprintf("args = $%d", argCount))
		args = append(args, pq.StringArray(req.Args))
	}

	if req.Env != nil {
		envJSON, err := json.Marshal(req.Env)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal env: %w", err)
		}
		argCount++
		updates = append(updates, fmt.Sprintf("env = $%d", argCount))
		args = append(args, envJSON)
	}

	if req.Enabled != nil {
		argCount++
		updates = append(updates, fmt.Sprintf("enabled = $%d", argCount))
		args = append(args, *req.Enabled)
	}

	if len(updates) == 0 {
		return r.Get(ctx, userID, serverID)
	}

	query := fmt.Sprintf(`
		UPDATE mcp_servers
		SET %s, updated_at = CURRENT_TIMESTAMP
		WHERE id = $1 AND user_id = $2
		RETURNING id, user_id, name, description, command, args, env, enabled, status,
		          last_connected_at, capabilities, metadata, created_at, updated_at`,
		joinStrings(updates, ", "))

	server := &models.MCPServer{}
	err := r.db.GetContext(ctx, server, query, args...)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("MCP server not found")
		}
		return nil, fmt.Errorf("failed to update MCP server: %w", err)
	}

	return server, nil
}

// Delete deletes an MCP server
func (r *MCPServerRepository) Delete(ctx context.Context, userID, serverID uuid.UUID) error {
	query := `DELETE FROM mcp_servers WHERE id = $1 AND user_id = $2`
	
	result, err := r.db.ExecContext(ctx, query, serverID, userID)
	if err != nil {
		return fmt.Errorf("failed to delete MCP server: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("MCP server not found")
	}

	return nil
}

// UpdateStatus updates the connection status of an MCP server
func (r *MCPServerRepository) UpdateStatus(ctx context.Context, serverID uuid.UUID, status models.MCPServerStatus) error {
	query := `
		UPDATE mcp_servers
		SET status = $2, updated_at = CURRENT_TIMESTAMP
		WHERE id = $1`

	_, err := r.db.ExecContext(ctx, query, serverID, string(status))
	if err != nil {
		return fmt.Errorf("failed to update MCP server status: %w", err)
	}

	return nil
}

// UpdateCapabilities updates the capabilities of an MCP server
func (r *MCPServerRepository) UpdateCapabilities(ctx context.Context, serverID uuid.UUID, capabilities interface{}) error {
	capJSON, err := json.Marshal(capabilities)
	if err != nil {
		return fmt.Errorf("failed to marshal capabilities: %w", err)
	}

	query := `
		UPDATE mcp_servers
		SET capabilities = $2, last_connected_at = CURRENT_TIMESTAMP, updated_at = CURRENT_TIMESTAMP
		WHERE id = $1`

	_, err = r.db.ExecContext(ctx, query, serverID, capJSON)
	if err != nil {
		return fmt.Errorf("failed to update MCP server capabilities: %w", err)
	}

	return nil
}

// GetServerTools retrieves all tools for a server
func (r *MCPServerRepository) GetServerTools(ctx context.Context, serverID uuid.UUID) ([]models.MCPTool, error) {
	query := `
		SELECT id, server_id, name, description, input_schema, enabled, 
		       usage_count, last_used_at, created_at, updated_at
		FROM mcp_tools
		WHERE server_id = $1
		ORDER BY name`

	tools := []models.MCPTool{}
	err := r.db.SelectContext(ctx, &tools, query, serverID)
	if err != nil {
		return nil, fmt.Errorf("failed to get server tools: %w", err)
	}

	return tools, nil
}

// UpsertTool creates or updates a tool for a server
func (r *MCPServerRepository) UpsertTool(ctx context.Context, serverID uuid.UUID, tool *models.MCPTool) error {
	tool.ServerID = serverID
	if tool.ID == uuid.Nil {
		tool.ID = uuid.New()
	}

	query := `
		INSERT INTO mcp_tools (
			id, server_id, name, description, input_schema, enabled
		) VALUES (
			$1, $2, $3, $4, $5, $6
		) ON CONFLICT (server_id, name) DO UPDATE SET
			description = EXCLUDED.description,
			input_schema = EXCLUDED.input_schema,
			updated_at = CURRENT_TIMESTAMP`

	_, err := r.db.ExecContext(
		ctx, query,
		tool.ID, tool.ServerID, tool.Name, tool.Description,
		tool.InputSchema, tool.Enabled,
	)

	if err != nil {
		return fmt.Errorf("failed to upsert tool: %w", err)
	}

	return nil
}

// UpdateToolUsage updates the usage statistics for a tool
func (r *MCPServerRepository) UpdateToolUsage(ctx context.Context, toolID uuid.UUID) error {
	query := `
		UPDATE mcp_tools
		SET usage_count = usage_count + 1,
		    last_used_at = CURRENT_TIMESTAMP,
		    updated_at = CURRENT_TIMESTAMP
		WHERE id = $1`

	_, err := r.db.ExecContext(ctx, query, toolID)
	if err != nil {
		return fmt.Errorf("failed to update tool usage: %w", err)
	}

	return nil
}

// GetServerResources retrieves all resources for a server
func (r *MCPServerRepository) GetServerResources(ctx context.Context, serverID uuid.UUID) ([]models.MCPResource, error) {
	query := `
		SELECT id, server_id, uri, name, description, mime_type, metadata, created_at, updated_at
		FROM mcp_resources
		WHERE server_id = $1
		ORDER BY name`

	resources := []models.MCPResource{}
	err := r.db.SelectContext(ctx, &resources, query, serverID)
	if err != nil {
		return nil, fmt.Errorf("failed to get server resources: %w", err)
	}

	return resources, nil
}

// UpsertResource creates or updates a resource for a server
func (r *MCPServerRepository) UpsertResource(ctx context.Context, serverID uuid.UUID, resource *models.MCPResource) error {
	resource.ServerID = serverID
	if resource.ID == uuid.Nil {
		resource.ID = uuid.New()
	}

	query := `
		INSERT INTO mcp_resources (
			id, server_id, uri, name, description, mime_type, metadata
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7
		) ON CONFLICT (server_id, uri) DO UPDATE SET
			name = EXCLUDED.name,
			description = EXCLUDED.description,
			mime_type = EXCLUDED.mime_type,
			metadata = EXCLUDED.metadata,
			updated_at = CURRENT_TIMESTAMP`

	_, err := r.db.ExecContext(
		ctx, query,
		resource.ID, resource.ServerID, resource.URI, resource.Name,
		resource.Description, resource.MimeType, resource.Metadata,
	)

	if err != nil {
		return fmt.Errorf("failed to upsert resource: %w", err)
	}

	return nil
}

