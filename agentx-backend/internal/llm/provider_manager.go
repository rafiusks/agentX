package llm

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// ProviderManager manages provider lifecycle and access
type ProviderManager struct {
	providers map[string]Provider       // key: userID:connectionID
	configs   map[string]ProviderConfig // key: userID:connectionID
	health    map[string]HealthStatus   // key: userID:connectionID
	factory   ProviderFactory
	mu        sync.RWMutex
}

// ProviderFactory creates provider instances
type ProviderFactory interface {
	CreateProvider(config ProviderConfig) (Provider, error)
}

// NewProviderManager creates a new provider manager
func NewProviderManager() *ProviderManager {
	return &ProviderManager{
		providers: make(map[string]Provider),
		configs:   make(map[string]ProviderConfig),
		health:    make(map[string]HealthStatus),
		factory:   &DefaultProviderFactory{},
	}
}

// RegisterProvider registers a new provider instance
func (pm *ProviderManager) RegisterProvider(userID, connectionID string, config ProviderConfig) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	key := pm.makeKey(userID, connectionID)
	
	// Close existing provider if any
	if existing, exists := pm.providers[key]; exists {
		existing.Close()
	}

	// Create new provider
	provider, err := pm.factory.CreateProvider(config)
	if err != nil {
		return fmt.Errorf("failed to create provider: %w", err)
	}

	// Test provider health
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	if err := provider.HealthCheck(ctx); err != nil {
		provider.Close()
		return fmt.Errorf("provider health check failed: %w", err)
	}

	// Store provider and config
	pm.providers[key] = provider
	pm.configs[key] = config
	pm.health[key] = HealthStatus{
		Provider:  config.Type,
		Status:    "healthy",
		LastCheck: time.Now(),
	}

	fmt.Printf("[ProviderManager] Registered provider: %s for user %s\n", key, userID)
	return nil
}

// GetProvider returns a provider for a user and connection
func (pm *ProviderManager) GetProvider(userID, connectionID string) (Provider, error) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	key := pm.makeKey(userID, connectionID)
	provider, exists := pm.providers[key]
	if !exists {
		return nil, fmt.Errorf("provider not found: %s", key)
	}

	return provider, nil
}

// GetProviderByKey returns a provider by its full key
func (pm *ProviderManager) GetProviderByKey(key string) (Provider, error) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	provider, exists := pm.providers[key]
	if !exists {
		return nil, fmt.Errorf("provider not found: %s", key)
	}

	return provider, nil
}

// RemoveProvider removes a provider registration
func (pm *ProviderManager) RemoveProvider(userID, connectionID string) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	key := pm.makeKey(userID, connectionID)
	
	if provider, exists := pm.providers[key]; exists {
		provider.Close()
		delete(pm.providers, key)
		delete(pm.configs, key)
		delete(pm.health, key)
		fmt.Printf("[ProviderManager] Removed provider: %s\n", key)
	}

	return nil
}

// GetAvailableModels returns all available models for a user
func (pm *ProviderManager) GetAvailableModels(ctx context.Context, userID string) ([]ModelInfo, error) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	var allModels []ModelInfo
	
	for key, provider := range pm.providers {
		// Check if this provider belongs to the user
		if !pm.isUserProvider(key, userID) {
			continue
		}

		// Get models from provider
		models, err := provider.GetModels(ctx)
		if err != nil {
			fmt.Printf("[ProviderManager] Failed to get models from %s: %v\n", key, err)
			continue
		}

		// Add connection info to models
		for i := range models {
			models[i].Provider = key
		}
		
		allModels = append(allModels, models...)
	}

	return allModels, nil
}

// GetUserProviders returns all providers for a user
func (pm *ProviderManager) GetUserProviders(userID string) map[string]Provider {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	userProviders := make(map[string]Provider)
	
	for key, provider := range pm.providers {
		if pm.isUserProvider(key, userID) {
			userProviders[key] = provider
		}
	}

	return userProviders
}

// HealthCheck performs health checks on all providers
func (pm *ProviderManager) HealthCheck(ctx context.Context) map[string]HealthStatus {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	results := make(map[string]HealthStatus)
	
	for key, provider := range pm.providers {
		start := time.Now()
		err := provider.HealthCheck(ctx)
		latency := time.Since(start)

		status := HealthStatus{
			Provider:  pm.configs[key].Type,
			Latency:   latency,
			LastCheck: time.Now(),
		}

		if err != nil {
			status.Status = "unhealthy"
			status.Error = err.Error()
		} else {
			status.Status = "healthy"
		}

		// Get model count
		if models, err := provider.GetModels(ctx); err == nil {
			status.Models = len(models)
		}

		pm.health[key] = status
		results[key] = status
	}

	return results
}

// GetHealthStatus returns the health status for a specific provider
func (pm *ProviderManager) GetHealthStatus(userID, connectionID string) (HealthStatus, bool) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	key := pm.makeKey(userID, connectionID)
	status, exists := pm.health[key]
	return status, exists
}

// Shutdown gracefully shuts down all providers
func (pm *ProviderManager) Shutdown(ctx context.Context) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	var errors []error
	
	for key, provider := range pm.providers {
		if err := provider.Close(); err != nil {
			errors = append(errors, fmt.Errorf("failed to close provider %s: %w", key, err))
		}
	}

	// Clear maps
	pm.providers = make(map[string]Provider)
	pm.configs = make(map[string]ProviderConfig)
	pm.health = make(map[string]HealthStatus)

	if len(errors) > 0 {
		return fmt.Errorf("shutdown errors: %v", errors)
	}

	return nil
}

// makeKey creates a provider key from user and connection IDs
func (pm *ProviderManager) makeKey(userID, connectionID string) string {
	return fmt.Sprintf("%s:%s", userID, connectionID)
}

// isUserProvider checks if a provider key belongs to a user
func (pm *ProviderManager) isUserProvider(key, userID string) bool {
	return len(key) > len(userID) && key[:len(userID)] == userID
}