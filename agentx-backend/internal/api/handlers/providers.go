package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/agentx/agentx-backend/internal/services"
)

// GetProviders returns all configured providers
func GetProviders(svc *services.Services) fiber.Handler {
	return func(c *fiber.Ctx) error {
		providers := []fiber.Map{}
		
		// Get all registered providers
		for id, provider := range svc.Providers.GetAll() {
			providers = append(providers, fiber.Map{
				"id":      id,
				"name":    provider.Name(),
				"enabled": true,
				"status":  "ready",
				"type":    id,
			})
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

// DiscoverModels discovers available models for a provider
func DiscoverModels(svc *services.Services) fiber.Handler {
	return func(c *fiber.Ctx) error {
		providerId := c.Params("id")
		
		provider := svc.Providers.Get(providerId)
		if provider == nil {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Provider not found",
			})
		}
		
		models, err := provider.GetModels(c.UserContext())
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": err.Error(),
			})
		}
		
		return c.JSON(models)
	}
}

// GetProvidersHealth returns health status for all providers
func GetProvidersHealth(svc *services.Services) fiber.Handler {
	return func(c *fiber.Ctx) error {
		health := svc.Router.GetHealthMonitor().GetAllHealth()
		
		return c.JSON(fiber.Map{
			"providers": health,
		})
	}
}