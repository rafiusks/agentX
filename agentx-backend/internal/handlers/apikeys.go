package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// ListAPIKeys returns all API keys for the authenticated user
func (h *Handler) ListAPIKeys(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)

	keys, err := h.authService.GetAPIKeysByUserID(c.Context(), userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to retrieve API keys",
		})
	}

	// Mask the actual keys for security
	maskedKeys := make([]fiber.Map, len(keys))
	for i, key := range keys {
		maskedKey := maskAPIKey(key.Key)
		maskedKeys[i] = fiber.Map{
			"id":        key.ID,
			"name":      key.Name,
			"key":       maskedKey,
			"scopes":    key.Scopes,
			"lastUsed":  key.LastUsedAt,
			"createdAt": key.CreatedAt,
			"expiresAt": key.ExpiresAt,
		}
	}

	return c.JSON(fiber.Map{
		"keys": maskedKeys,
	})
}

// CreateAPIKey creates a new API key for the authenticated user
func (h *Handler) CreateAPIKey(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)

	var req struct {
		Name          string   `json:"name"`
		Scopes        []string `json:"scopes"`
		ExpiresInDays int      `json:"expiresInDays"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if req.Name == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "API key name is required",
		})
	}

	if len(req.Scopes) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "At least one scope is required",
		})
	}

	// Generate a secure random API key
	keyBytes := make([]byte, 32)
	if _, err := rand.Read(keyBytes); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to generate API key",
		})
	}
	apiKey := "agx_" + hex.EncodeToString(keyBytes)

	// Calculate expiration
	var expiresAt *time.Time
	if req.ExpiresInDays > 0 {
		exp := time.Now().AddDate(0, 0, req.ExpiresInDays)
		expiresAt = &exp
	}

	// Create the API key
	key, err := h.authService.CreateAPIKey(c.Context(), userID, req.Name, apiKey, req.Scopes, expiresAt)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to create API key",
		})
	}

	// Return the full key only on creation
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"id":        key.ID,
		"name":      key.Name,
		"key":       apiKey, // Return full key only on creation
		"scopes":    key.Scopes,
		"createdAt": key.CreatedAt,
		"expiresAt": key.ExpiresAt,
	})
}

// DeleteAPIKey deletes an API key
func (h *Handler) DeleteAPIKey(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)
	keyID := c.Params("id")

	keyUUID, err := uuid.Parse(keyID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid API key ID",
		})
	}

	// Verify the key belongs to the user
	key, err := h.authService.GetAPIKeyByID(c.Context(), keyUUID)
	if err != nil || key.UserID != userID {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "API key not found",
		})
	}

	if err := h.authService.DeleteAPIKey(c.Context(), keyUUID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to delete API key",
		})
	}

	return c.JSON(fiber.Map{
		"message": "API key deleted successfully",
	})
}

// maskAPIKey masks an API key for display
func maskAPIKey(key string) string {
	if len(key) <= 8 {
		return "••••••••"
	}
	return key[:4] + "••••••••" + key[len(key)-4:]
}