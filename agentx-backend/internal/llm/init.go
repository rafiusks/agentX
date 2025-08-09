package llm

import (
	"fmt"
	
	"github.com/agentx/agentx-backend/internal/providers"
)

// InitializeGateway creates and configures a new LLM Gateway
func InitializeGateway(providerRegistry *providers.Registry) (*Gateway, error) {
	// Create gateway with default options
	gateway := NewGateway(
		WithMetrics(NewMetricsCollector()),
		WithCircuitBreaker(NewCircuitBreaker()),
	)
	
	// Configure router
	gateway.router.SetProviderManager(gateway.providers)
	gateway.router.SetConfigManager(gateway.config)
	
	// Add default routing rules
	gateway.router.AddRoutingRule(RoutingRule{
		Name:     "prefer-fast-models",
		Priority: 10,
		Condition: func(req *Request) bool {
			return req.Requirements.MaxLatency > 0
		},
		Action: func(req *Request) *RouteInfo {
			// Route to faster models for latency-sensitive requests
			return &RouteInfo{
				Model:  "gpt-3.5-turbo",
				Reason: "fast model for low latency",
			}
		},
	})
	
	gateway.router.AddRoutingRule(RoutingRule{
		Name:     "prefer-quality-models",
		Priority: 5,
		Condition: func(req *Request) bool {
			return req.Requirements.MinQuality == "high"
		},
		Action: func(req *Request) *RouteInfo {
			// Route to higher quality models
			return &RouteInfo{
				Model:  "gpt-4",
				Reason: "quality model for high quality requirement",
			}
		},
	})
	
	// Migrate existing providers if available
	if providerRegistry != nil {
		if err := migrateProviders(gateway, providerRegistry); err != nil {
			return nil, fmt.Errorf("failed to migrate providers: %w", err)
		}
	}
	
	return gateway, nil
}

// migrateProviders migrates existing providers to the new gateway
func migrateProviders(gateway *Gateway, registry *providers.Registry) error {
	// Get all providers from the registry
	allProviders := registry.GetAll()
	
	for key, provider := range allProviders {
		// Parse the key (format: "userID:connectionID")
		var userID, connectionID string
		if n, _ := fmt.Sscanf(key, "%s:%s", &userID, &connectionID); n != 2 {
			// Skip invalid keys
			continue
		}
		
		// Determine provider type from the provider name
		providerType := determineProviderType(provider)
		
		// Create provider config
		config := ProviderConfig{
			Type: providerType,
			Name: provider.Name(),
		}
		
		// Register in the new gateway
		if err := gateway.RegisterProvider(userID, connectionID, config); err != nil {
			fmt.Printf("[Migration] Failed to migrate provider %s: %v\n", key, err)
			// Continue with other providers
		} else {
			fmt.Printf("[Migration] Successfully migrated provider %s\n", key)
		}
	}
	
	return nil
}

// determineProviderType determines the provider type from the provider instance
func determineProviderType(provider providers.Provider) string {
	name := provider.Name()
	
	// Simple heuristic based on provider name
	switch name {
	case "openai", "gpt", "chatgpt":
		return "openai"
	case "anthropic", "claude":
		return "anthropic"
	case "ollama", "local", "llama":
		return "local"
	default:
		return "openai" // Default to OpenAI
	}
}

// CreateFromExistingProvider wraps an existing provider with the new interface
func CreateFromExistingProvider(provider providers.Provider, userID, connectionID string) error {
	// This would be used to wrap existing providers with the new interface
	// For now, we'll use the stub implementation
	return nil
}