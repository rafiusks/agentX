package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/agentx/agentx-backend/internal/services"
)

// GetSettings returns application settings
func GetSettings(svc *services.Services) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// For now, return a stub response with the expected structure
		// TODO: Properly load from database when configs table is populated
		settings := fiber.Map{
			"providers": fiber.Map{
				"openai": fiber.Map{
					"id":   "openai",
					"type": "openai",
					"name": "OpenAI",
				},
				"anthropic": fiber.Map{
					"id":   "anthropic",
					"type": "anthropic",
					"name": "Anthropic",
				},
				"ollama": fiber.Map{
					"id":       "ollama",
					"type":     "ollama",
					"name":     "Ollama",
					"base_url": "http://localhost:11434",
				},
				"lm-studio": fiber.Map{
					"id":       "lm-studio",
					"type":     "openai-compatible",
					"name":     "LM Studio",
					"base_url": "http://localhost:1234",
				},
			},
			"default_provider": "openai",
			"default_model":    "gpt-3.5-turbo",
		}
		
		return c.JSON(settings)
	}
}

// UpdateSettings updates application settings
func UpdateSettings(svc *services.Services) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var settings map[string]interface{}
		
		if err := c.BodyParser(&settings); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Invalid request body",
			})
		}
		
		err := svc.Config.UpdateSettings(c.UserContext(), settings)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": err.Error(),
			})
		}
		
		return c.JSON(fiber.Map{
			"message": "Settings updated successfully",
		})
	}
}