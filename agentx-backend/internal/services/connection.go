package services

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/agentx/agentx-backend/internal/models"
	"github.com/agentx/agentx-backend/internal/providers"
	"github.com/agentx/agentx-backend/internal/providers/factory"
	"github.com/agentx/agentx-backend/internal/repository"
	"github.com/agentx/agentx-backend/internal/config"
)

// ConnectionService manages provider connections
type ConnectionService struct {
	connectionRepo   repository.ConnectionRepository
	providerRegistry *providers.Registry
}

// NewConnectionService creates a new connection service
func NewConnectionService(connectionRepo repository.ConnectionRepository, providerRegistry *providers.Registry) *ConnectionService {
	return &ConnectionService{
		connectionRepo:   connectionRepo,
		providerRegistry: providerRegistry,
	}
}

// InitializeConnections loads and registers all enabled connections on startup
// This is now per-user and should be called when a user logs in
func (s *ConnectionService) InitializeUserConnections(ctx context.Context, userID uuid.UUID) error {
	connections, err := s.connectionRepo.List(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to list connections: %w", err)
	}
	
	for _, conn := range connections {
		if conn.Enabled {
			if err := s.initializeProvider(userID, conn); err != nil {
				// Log error but continue with other connections
				fmt.Printf("Failed to initialize connection %s (%s): %v\n", conn.Name, conn.ID, err)
			}
		}
	}
	
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
		
		// Check connection status (registry is now per-user)
		if conn.Enabled {
			registryKey := fmt.Sprintf("%s:%s", userID.String(), conn.ID)
			if s.providerRegistry.Has(registryKey) {
				withStatus.Status = "connected"
			} else {
				// Try to initialize the provider if not in registry
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
	
	// Check connection status (registry is now per-user)
	if conn.Enabled {
		registryKey := fmt.Sprintf("%s:%s", userID.String(), conn.ID)
		if s.providerRegistry.Has(registryKey) {
			withStatus.Status = "connected"
		} else {
			// Try to initialize the provider if not in registry
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
		// Remove old provider instance
		registryKey := fmt.Sprintf("%s:%s", userID.String(), id)
		s.providerRegistry.Unregister(registryKey)
		
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
	// Remove provider instance
	registryKey := fmt.Sprintf("%s:%s", userID.String(), id)
	s.providerRegistry.Unregister(registryKey)
	
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
	
	// If the test was successful and the connection is enabled, register the provider
	if conn.Enabled {
		// Remove old provider instance if exists
		registryKey := fmt.Sprintf("%s:%s", userID.String(), conn.ID)
		s.providerRegistry.Unregister(registryKey)
		
		// Register the new provider instance
		s.providerRegistry.Register(registryKey, provider)
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
	providerConfig := config.ProviderConfig{
		Type:    conn.ProviderID,
		Name:    conn.Name,
		APIKey:  getStringFromMap(conn.Config, "api_key"),
		BaseURL: getStringFromMap(conn.Config, "base_url"),
	}
	
	provider, err := factory.CreateProvider(conn.ProviderID, providerConfig)
	if err != nil {
		return err
	}
	
	// Register with user:connection ID as the key for isolation
	registryKey := fmt.Sprintf("%s:%s", userID.String(), conn.ID)
	s.providerRegistry.Register(registryKey, provider)
	
	return nil
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