package llm

import (
	"fmt"
	
	"github.com/agentx/agentx-backend/internal/providers"
	"github.com/agentx/agentx-backend/internal/providers/factory"
)

// DefaultProviderFactory creates provider instances based on configuration
type DefaultProviderFactory struct{}

// CreateProvider creates a provider instance based on config
func (f *DefaultProviderFactory) CreateProvider(config ProviderConfig) (Provider, error) {
	fmt.Printf("[ProviderFactory] Creating provider - Type: %s, Name: %s\n", config.Type, config.Name)
	
	// Convert LLM service config to providers package config
	providerConfig := config.ToProviderConfig()
	
	// Create actual provider using existing factory
	realProvider, err := factory.CreateProvider(config.Name, providerConfig)
	if err != nil {
		fmt.Printf("[ProviderFactory] Failed to create real provider: %v\n", err)
		return nil, fmt.Errorf("failed to create provider: %w", err)
	}
	
	fmt.Printf("[ProviderFactory] Successfully created real provider: %s\n", realProvider.Name())
	
	// Wrap with adapter to bridge interfaces
	adapter := &ProviderAdapter{
		provider: realProvider,
		config:   config,
	}
	
	fmt.Printf("[ProviderFactory] Wrapped with adapter, returning Provider interface\n")
	return adapter, nil
}

// ProviderAdapter wraps existing provider implementations with the new interface
type ProviderAdapter struct {
	provider providers.Provider
	config   ProviderConfig
}

