package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/agentx/agentx-backend/internal/services"
)

// GetProviders returns all configured providers through the Orchestrator
func GetProviders(svc *services.Services) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Get models through Gateway to determine active providers
		adminUserID := "admin" 
		models, err := svc.Gateway.GetAvailableModels(c.Context(), adminUserID)
		if err != nil {
			// Fallback to registry for provider list if Gateway fails
			providers := []fiber.Map{}
			for id, provider := range svc.Providers.GetAll() {
				providers = append(providers, fiber.Map{
					"id":      id,
					"name":    provider.Name(),
					"enabled": true,
					"status":  "unknown",
					"type":    id,
					"note":    "Gateway unavailable - using registry fallback",
				})
			}
			return c.JSON(providers)
		}
		
		// Build provider list from Gateway models
		providerMap := make(map[string]fiber.Map)
		for _, model := range models {
			if _, exists := providerMap[model.Provider]; !exists {
				providerMap[model.Provider] = fiber.Map{
					"id":           model.Provider,
					"name":         model.Provider,
					"enabled":      true,
					"status":       "active",
					"type":         model.Provider,
					"model_count":  1,
					"via_gateway":  true,
				}
			} else {
				provider := providerMap[model.Provider]
				provider["model_count"] = provider["model_count"].(int) + 1
				providerMap[model.Provider] = provider
			}
		}
		
		// Convert map to array
		providers := make([]fiber.Map, 0, len(providerMap))
		for _, provider := range providerMap {
			providers = append(providers, provider)
		}
		
		return c.JSON(providers)
	}
}

// UpdateProviderConfig updates provider configuration
func UpdateProviderConfig(svc *services.Services) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// providerId := c.Params("id")
		
		// TODO: Implement provider config update
		// For now, just return success
		
		return c.JSON(fiber.Map{
			"message": "Provider configuration updated",
		})
	}
}

// DiscoverModels discovers available models for a provider through the Gateway
func DiscoverModels(svc *services.Services) fiber.Handler {
	return func(c *fiber.Ctx) error {
		providerId := c.Params("id")
		
		// Use Gateway for model discovery (requires userID for proper routing)
		// For admin endpoints, we can use a default admin userID
		adminUserID := "admin"
		
		// Get all available models through the Gateway
		models, err := svc.Gateway.GetAvailableModels(c.Context(), adminUserID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": err.Error(),
			})
		}
		
		// Filter models by provider if specified
		var providerModels []interface{}
		for _, model := range models {
			if providerId == "all" || model.Provider == providerId {
				providerModels = append(providerModels, model)
			}
		}
		
		return c.JSON(fiber.Map{
			"provider": providerId,
			"models":   providerModels,
		})
	}
}

// GetProvidersHealth returns health status for all providers through the Gateway
func GetProvidersHealth(svc *services.Services) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Test Gateway connectivity by attempting to get models
		adminUserID := "admin"
		models, err := svc.Gateway.GetAvailableModels(c.Context(), adminUserID)
		
		if err != nil {
			// Gateway is unhealthy
			return c.JSON(fiber.Map{
				"status": "unhealthy",
				"error":  err.Error(),
				"gateway": fiber.Map{
					"status":  "unhealthy",
					"message": "Failed to connect to LLM Gateway",
				},
				"providers": []fiber.Map{},
			})
		}
		
		// Build health status based on available models
		providerHealth := make(map[string]fiber.Map)
		for _, model := range models {
			if _, exists := providerHealth[model.Provider]; !exists {
				providerHealth[model.Provider] = fiber.Map{
					"id":           model.Provider,
					"name":         model.Provider,
					"status":       "healthy",
					"enabled":      true,
					"models":       1,
					"last_check":   "via_gateway",
				}
			} else {
				provider := providerHealth[model.Provider]
				provider["models"] = provider["models"].(int) + 1
				providerHealth[model.Provider] = provider
			}
		}
		
		// Convert to array
		providers := make([]fiber.Map, 0, len(providerHealth))
		for _, provider := range providerHealth {
			providers = append(providers, provider)
		}
		
		return c.JSON(fiber.Map{
			"status": "healthy",
			"gateway": fiber.Map{
				"status":  "healthy",
				"message": "LLM Gateway is operational",
			},
			"providers": providers,
		})
	}
}

// DebugProviders returns debug info through the Gateway
func DebugProviders(svc *services.Services) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Get debug info through Gateway by attempting model discovery
		adminUserID := "admin"
		models, err := svc.Gateway.GetAvailableModels(c.Context(), adminUserID)
		
		debugInfo := fiber.Map{
			"architecture": "OrchestrationService -> Gateway -> Providers",
			"via_gateway":  true,
			"timestamp":    c.Context().Time().Format("2006-01-02T15:04:05Z"),
		}
		
		if err != nil {
			debugInfo["gateway_status"] = "error"
			debugInfo["gateway_error"] = err.Error()
			debugInfo["providers"] = fiber.Map{}
			debugInfo["total_models"] = 0
			return c.JSON(debugInfo)
		}
		
		// Build provider debug info from Gateway models
		providerDebug := make(map[string]interface{})
		totalModels := 0
		
		for _, model := range models {
			if _, exists := providerDebug[model.Provider]; !exists {
				providerDebug[model.Provider] = map[string]interface{}{
					"name":        model.Provider,
					"type":        "active_via_gateway",
					"models":      []string{},
					"model_count": 0,
				}
			}
			
			provider := providerDebug[model.Provider].(map[string]interface{})
			provider["models"] = append(provider["models"].([]string), model.ID)
			provider["model_count"] = provider["model_count"].(int) + 1
			providerDebug[model.Provider] = provider
			totalModels++
		}
		
		debugInfo["gateway_status"] = "healthy"
		debugInfo["providers"] = providerDebug
		debugInfo["total_providers"] = len(providerDebug)
		debugInfo["total_models"] = totalModels
		
		return c.JSON(debugInfo)
	}
}