package services

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/agentx/agentx-backend/internal/models"
	"github.com/agentx/agentx-backend/internal/repository"
	"github.com/agentx/agentx-backend/internal/llm"
	"github.com/agentx/agentx-backend/internal/config"
	"github.com/agentx/agentx-backend/internal/providers/factory"
)

// ConnectionService manages provider connections
type ConnectionService struct {
	connectionRepo repository.ConnectionRepository
	gateway        *llm.Gateway // Single source of truth for provider registration
}

// NewConnectionService creates a new connection service
func NewConnectionService(connectionRepo repository.ConnectionRepository, gateway *llm.Gateway) *ConnectionService {
	return &ConnectionService{
		connectionRepo: connectionRepo,
		gateway:        gateway,
	}
}

// InitializeConnections loads and registers all enabled connections on startup
// This is now per-user and should be called when a user logs in
func (s *ConnectionService) InitializeUserConnections(ctx context.Context, userID uuid.UUID) error {
	fmt.Printf("[InitializeUserConnections] Starting for user: %s\n", userID.String())
	
	connections, err := s.connectionRepo.List(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to list connections: %w", err)
	}
	
	fmt.Printf("[InitializeUserConnections] Found %d connections for user %s\n", len(connections), userID.String())
	
	for _, conn := range connections {
		fmt.Printf("[InitializeUserConnections] Connection %s (ID: %s) - Enabled: %v\n", conn.Name, conn.ID, conn.Enabled)
		if conn.Enabled {
			if err := s.initializeProvider(userID, conn); err != nil {
				// Log error but continue with other connections
				fmt.Printf("[InitializeUserConnections] Failed to initialize connection %s (%s): %v\n", conn.Name, conn.ID, err)
			} else {
				fmt.Printf("[InitializeUserConnections] Successfully initialized connection %s (%s)\n", conn.Name, conn.ID)
			}
		}
	}
	
	fmt.Printf("[InitializeUserConnections] Completed for user: %s\n", userID.String())
	return nil
}

// ListConnections returns all connections, optionally filtered by provider ID
func (s *ConnectionService) ListConnections(ctx context.Context, userID uuid.UUID, providerID string) ([]*models.ConnectionWithStatus, error) {
	var connections []*repository.ProviderConnection
	var err error
	
	if providerID != "" {
		connections, err = s.connectionRepo.GetByProviderID(ctx, userID, providerID)
	} else {
		connections, err = s.connectionRepo.List(ctx, userID)
	}
	
	if err != nil {
		return nil, err
	}
	
	// Convert to models with status
	result := make([]*models.ConnectionWithStatus, 0, len(connections))
	for _, conn := range connections {
		connID, _ := uuid.Parse(conn.ID)
		withStatus := &models.ConnectionWithStatus{
			ProviderConnection: &models.ProviderConnection{
				ID:         connID,
				ProviderID: conn.ProviderID,
				Name:       conn.Name,
				Enabled:    conn.Enabled,
				Config:     conn.Config,
				Metadata:   conn.Metadata,
				CreatedAt:  conn.CreatedAt,
				UpdatedAt:  conn.UpdatedAt,
			},
		}
		
		// Check connection status with Gateway
		if conn.Enabled {
			// Try to get provider from Gateway to check if it exists
			models, err := s.gateway.GetAvailableModels(context.Background(), userID.String())
			hasProvider := err == nil && len(models) > 0
			
			if hasProvider {
				withStatus.Status = "connected"
			} else {
				// Try to initialize the provider if not in Gateway
				if err := s.initializeProvider(userID, conn); err == nil {
					withStatus.Status = "connected"
				} else {
					withStatus.Status = "disconnected"
				}
			}
		} else {
			withStatus.Status = "disabled"
		}
		
		result = append(result, withStatus)
	}
	
	return result, nil
}

// GetConnection returns a single connection by ID
func (s *ConnectionService) GetConnection(ctx context.Context, userID uuid.UUID, id string) (*models.ConnectionWithStatus, error) {
	conn, err := s.connectionRepo.GetByID(ctx, userID, id)
	if err != nil {
		return nil, err
	}
	
	connID, _ := uuid.Parse(conn.ID)
	withStatus := &models.ConnectionWithStatus{
		ProviderConnection: &models.ProviderConnection{
			ID:         connID,
			ProviderID: conn.ProviderID,
			Name:       conn.Name,
			Enabled:    conn.Enabled,
			Config:     conn.Config,
			Metadata:   conn.Metadata,
			CreatedAt:  conn.CreatedAt,
			UpdatedAt:  conn.UpdatedAt,
		},
	}
	
	// Check connection status with Gateway
	if conn.Enabled {
		// Try to get provider from Gateway to check if it exists
		models, err := s.gateway.GetAvailableModels(context.Background(), userID.String())
		hasProvider := err == nil && len(models) > 0
		
		if hasProvider {
			withStatus.Status = "connected"
		} else {
			// Try to initialize the provider if not in Gateway
			if err := s.initializeProvider(userID, conn); err == nil {
				withStatus.Status = "connected"
			} else {
				withStatus.Status = "disconnected"
			}
		}
	} else {
		withStatus.Status = "disabled"
	}
	
	return withStatus, nil
}

// CreateConnection creates a new connection
func (s *ConnectionService) CreateConnection(ctx context.Context, userID uuid.UUID, providerID, name string, cfg map[string]interface{}) (*models.ConnectionWithStatus, error) {
	// Create the connection in the database
	id, err := s.connectionRepo.Create(ctx, userID, providerID, name, cfg)
	if err != nil {
		return nil, err
	}
	
	// Get the created connection
	conn, err := s.connectionRepo.GetByID(ctx, userID, id)
	if err != nil {
		return nil, err
	}
	
	// Initialize the provider if enabled
	if conn.Enabled {
		if err := s.initializeProvider(userID, conn); err != nil {
			// Log error but don't fail creation
			fmt.Printf("Failed to initialize provider for connection %s: %v\n", id, err)
		}
	}
	
	return s.GetConnection(ctx, userID, id)
}

// UpdateConnection updates a connection
func (s *ConnectionService) UpdateConnection(ctx context.Context, userID uuid.UUID, id string, updates map[string]interface{}) error {
	// Update in database
	if err := s.connectionRepo.Update(ctx, userID, id, updates); err != nil {
		return err
	}
	
	// Get updated connection
	conn, err := s.connectionRepo.GetByID(ctx, userID, id)
	if err != nil {
		return err
	}
	
	// Reinitialize provider if config changed
	if _, hasConfig := updates["config"]; hasConfig || updates["enabled"] != nil {
		// Remove old provider instance from Gateway
		s.gateway.RemoveProvider(userID.String(), id)
		
		// Initialize new instance if enabled
		if conn.Enabled {
			if err := s.initializeProvider(userID, conn); err != nil {
				return fmt.Errorf("failed to reinitialize provider: %w", err)
			}
		}
	}
	
	return nil
}

// DeleteConnection deletes a connection
func (s *ConnectionService) DeleteConnection(ctx context.Context, userID uuid.UUID, id string) error {
	// Remove provider instance from Gateway
	s.gateway.RemoveProvider(userID.String(), id)
	
	// Delete from database
	return s.connectionRepo.Delete(ctx, userID, id)
}

// ToggleConnection toggles a connection's enabled state
func (s *ConnectionService) ToggleConnection(ctx context.Context, userID uuid.UUID, id string) (*models.ConnectionWithStatus, error) {
	conn, err := s.connectionRepo.GetByID(ctx, userID, id)
	if err != nil {
		return nil, err
	}
	
	// Toggle enabled state
	updates := map[string]interface{}{
		"enabled": !conn.Enabled,
	}
	
	if err := s.UpdateConnection(ctx, userID, id, updates); err != nil {
		return nil, err
	}
	
	return s.GetConnection(ctx, userID, id)
}

// TestConnection tests a connection
func (s *ConnectionService) TestConnection(ctx context.Context, userID uuid.UUID, id string) (*models.TestConnectionResponse, error) {
	conn, err := s.connectionRepo.GetByID(ctx, userID, id)
	if err != nil {
		return nil, err
	}
	
	// Create a temporary provider instance for testing
	providerConfig := config.ProviderConfig{
		Type:    conn.ProviderID,
		Name:    conn.Name,
		APIKey:  getStringFromMap(conn.Config, "api_key"),
		BaseURL: getStringFromMap(conn.Config, "base_url"),
	}
	
	provider, err := factory.CreateProvider(conn.ProviderID, providerConfig)
	if err != nil {
		return &models.TestConnectionResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to create provider: %v", err),
		}, nil
	}
	
	// Test the provider
	_, err = provider.GetModels(ctx)
	if err != nil {
		return &models.TestConnectionResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to list models: %v", err),
		}, nil
	}
	
	// Update metadata with test results
	metadata := conn.Metadata
	if metadata == nil {
		metadata = make(map[string]interface{})
	}
	
	now := time.Now()
	metadata["last_tested"] = now
	metadata["test_status"] = "success"
	metadata["test_message"] = "Connection successful"
	
	// Update metadata in database
	updates := map[string]interface{}{
		"metadata": metadata,
	}
	s.connectionRepo.Update(ctx, userID, id, updates)
	
	// If the test was successful and the connection is enabled, register the provider with Gateway
	if conn.Enabled {
		// Re-initialize provider with Gateway
		if err := s.initializeProvider(userID, conn); err != nil {
			// Log error but don't fail the test since the connection itself works
			fmt.Printf("[TestConnection] Warning: Failed to register provider with Gateway: %v\n", err)
		}
	}
	
	return &models.TestConnectionResponse{
		Success: true,
		Message: "Connection successful",
		Details: map[string]interface{}{
			"tested_at": now,
		},
	}, nil
}

// SetDefaultConnection sets a connection as the default for its provider
func (s *ConnectionService) SetDefaultConnection(ctx context.Context, userID uuid.UUID, id string) error {
	conn, err := s.connectionRepo.GetByID(ctx, userID, id)
	if err != nil {
		return err
	}
	
	return s.connectionRepo.SetDefault(ctx, userID, conn.ProviderID, id)
}

// GetDefaultConnection gets the default connection for a provider
func (s *ConnectionService) GetDefaultConnection(ctx context.Context, userID uuid.UUID, providerID string) (*models.ConnectionWithStatus, error) {
	conn, err := s.connectionRepo.GetDefault(ctx, userID, providerID)
	if err != nil {
		return nil, err
	}
	
	return s.GetConnection(ctx, userID, conn.ID)
}

// initializeProvider creates and registers a provider instance
func (s *ConnectionService) initializeProvider(userID uuid.UUID, conn *repository.ProviderConnection) error {
	fmt.Printf("[initializeProvider] Creating provider - Type: %s, Name: %s, ConnectionID: %s\n", conn.ProviderID, conn.Name, conn.ID)
	
	// Map provider types to Gateway-supported types
	providerType := s.mapProviderType(conn.ProviderID)
	fmt.Printf("[initializeProvider] Mapped provider type %s -> %s\n", conn.ProviderID, providerType)
	
	// Create provider config for the Gateway
	gatewayConfig := llm.ProviderConfig{
		Type:         providerType,
		Name:         conn.Name,
		APIKey:       getStringFromMap(conn.Config, "api_key"),
		BaseURL:      getStringFromMap(conn.Config, "base_url"),
		Organization: getStringFromMap(conn.Config, "organization"),
	}
	
	// Register with Gateway (SINGLE SOURCE OF TRUTH)
	fmt.Printf("[initializeProvider] Registering with Gateway - UserID: %s, ConnectionID: %s\n", userID.String(), conn.ID)
	err := s.gateway.RegisterProvider(userID.String(), conn.ID, gatewayConfig)
	if err != nil {
		fmt.Printf("[initializeProvider] Failed to register with Gateway: %v\n", err)
		return fmt.Errorf("failed to register provider with Gateway: %w", err)
	}
	
	fmt.Printf("[initializeProvider] Successfully registered provider with Gateway\n")
	return nil
}

// EnsureConnectionInitialized ensures a specific connection is initialized
func (s *ConnectionService) EnsureConnectionInitialized(ctx context.Context, userID uuid.UUID, connectionID string) error {
	// Check if already registered with Gateway by trying to get models
	models, err := s.gateway.GetAvailableModels(ctx, userID.String())
	if err == nil && len(models) > 0 {
		fmt.Printf("[EnsureConnectionInitialized] Connection %s already registered with Gateway for user %s\n", connectionID, userID.String())
		return nil
	}
	
	fmt.Printf("[EnsureConnectionInitialized] Connection %s not found in Gateway, attempting to initialize\n", connectionID)
	
	// Get the connection from database
	conn, err := s.connectionRepo.GetByID(ctx, userID, connectionID)
	if err != nil {
		fmt.Printf("[EnsureConnectionInitialized] Failed to get connection from database: %v\n", err)
		return fmt.Errorf("connection not found: %w", err)
	}
	
	if !conn.Enabled {
		fmt.Printf("[EnsureConnectionInitialized] Connection %s is disabled\n", connectionID)
		return fmt.Errorf("connection is disabled")
	}
	
	// Initialize the provider with Gateway
	if err := s.initializeProvider(userID, conn); err != nil {
		fmt.Printf("[EnsureConnectionInitialized] Failed to initialize provider: %v\n", err)
		return fmt.Errorf("failed to initialize provider: %w", err)
	}
	
	fmt.Printf("[EnsureConnectionInitialized] Successfully initialized connection %s for user %s with Gateway\n", connectionID, userID.String())
	return nil
}

// GetConnectionConfig retrieves the configuration for a connection
func (s *ConnectionService) GetConnectionConfig(ctx context.Context, userID uuid.UUID, connectionID string) (map[string]interface{}, error) {
	conn, err := s.connectionRepo.GetByID(ctx, userID, connectionID)
	if err != nil {
		return nil, fmt.Errorf("connection not found: %w", err)
	}
	return conn.Config, nil
}

// mapProviderType maps database provider types to Gateway-supported types
func (s *ConnectionService) mapProviderType(dbProviderType string) string {
	switch dbProviderType {
	case "anthropic-claude":
		return "anthropic"
	case "ollama", "local-llm":
		return "local"
	default:
		// Return as-is for standard types (openai, openai-compatible, anthropic, local)
		return dbProviderType
	}
}

// getStringFromMap safely gets a string value from a map
func getStringFromMap(m map[string]interface{}, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}