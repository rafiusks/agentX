package handlers

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/agentx/agentx-backend/internal/api/middleware"
	"github.com/agentx/agentx-backend/internal/auth"
)

// APIKeyRequest represents a request to create an API key
type APIKeyRequest struct {
	Name      string    `json:"name" validate:"required"`
	Scopes    []string  `json:"scopes" validate:"required"`
	ExpiresAt *time.Time `json:"expires_at"`
}

// APIKeyResponse represents an API key in responses
type APIKeyResponse struct {
	ID        string     `json:"id"`
	Name      string     `json:"name"`
	KeyPrefix string     `json:"key_prefix"`
	Key       string     `json:"key,omitempty"` // Only sent on creation
	Scopes    []string   `json:"scopes"`
	ExpiresAt *time.Time `json:"expires_at"`
	LastUsed  *time.Time `json:"last_used"`
	CreatedAt time.Time  `json:"created_at"`
}

// ListAPIKeys returns all API keys for the current user
func ListAPIKeys(authService *auth.Service) fiber.Handler {
	return func(c *fiber.Ctx) error {
		userContext := middleware.GetUserContext(c)
		if userContext == nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Not authenticated",
			})
		}

		keys, err := authService.GetAPIKeysByUserID(c.Context(), userContext.UserID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to retrieve API keys",
			})
		}

		// Convert to response format
		response := make([]APIKeyResponse, len(keys))
		for i, key := range keys {
			response[i] = APIKeyResponse{
				ID:        key.ID.String(),
				Name:      key.Name,
				KeyPrefix: key.KeyPrefix,
				Scopes:    key.Scopes,
				ExpiresAt: key.ExpiresAt,
				LastUsed:  key.LastUsedAt,
				CreatedAt: key.CreatedAt,
			}
		}

		return c.JSON(fiber.Map{
			"api_keys": response,
		})
	}
}

// CreateAPIKey creates a new API key
func CreateAPIKey(authService *auth.Service) fiber.Handler {
	return func(c *fiber.Ctx) error {
		userContext := middleware.GetUserContext(c)
		if userContext == nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Not authenticated",
			})
		}

		var req APIKeyRequest
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Invalid request body",
			})
		}

		// Validate request
		if req.Name == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Name is required",
			})
		}

		if len(req.Scopes) == 0 {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "At least one scope is required",
			})
		}

		// Create API key
		apiKey, keyString, err := authService.CreateAPIKey(
			c.Context(),
			userContext.UserID,
			req.Name,
			req.Scopes,
			req.ExpiresAt,
		)
		if err != nil {
			if err == auth.ErrInvalidScope {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
					"error": "Invalid scope(s) provided",
				})
			}
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to create API key",
			})
		}

		return c.Status(fiber.StatusCreated).JSON(APIKeyResponse{
			ID:        apiKey.ID.String(),
			Name:      apiKey.Name,
			KeyPrefix: apiKey.KeyPrefix,
			Key:       keyString, // Only sent on creation
			Scopes:    apiKey.Scopes,
			ExpiresAt: apiKey.ExpiresAt,
			CreatedAt: apiKey.CreatedAt,
		})
	}
}

// RevokeAPIKey revokes an API key
func RevokeAPIKey(authService *auth.Service) fiber.Handler {
	return func(c *fiber.Ctx) error {
		userContext := middleware.GetUserContext(c)
		if userContext == nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Not authenticated",
			})
		}

		keyID := c.Params("id")
		keyUUID, err := uuid.Parse(keyID)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Invalid API key ID",
			})
		}

		// Revoke the key
		if err := authService.RevokeAPIKey(c.Context(), userContext.UserID, keyUUID); err != nil {
			if err.Error() == "unauthorized" {
				return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
					"error": "You don't have permission to revoke this key",
				})
			}
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to revoke API key",
			})
		}

		return c.JSON(fiber.Map{
			"message": "API key revoked successfully",
		})
	}
}