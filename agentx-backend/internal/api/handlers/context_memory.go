package handlers

import (
	"encoding/json"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/agentx/agentx-backend/internal/api/middleware"
	"github.com/agentx/agentx-backend/internal/services"
)

// ContextMemoryHandlers handles context memory-related endpoints
type ContextMemoryHandlers struct {
	contextMemory *services.ContextMemoryService
}

// NewContextMemoryHandlers creates new context memory handlers
func NewContextMemoryHandlers(contextMemory *services.ContextMemoryService) *ContextMemoryHandlers {
	return &ContextMemoryHandlers{
		contextMemory: contextMemory,
	}
}

// StoreMemory handles POST /api/v1/context/memory
func (h *ContextMemoryHandlers) StoreMemory(c *fiber.Ctx) error {
	userContext := middleware.GetUserContext(c)
	if userContext == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Not authenticated",
		})
	}

	var req struct {
		Namespace  string          `json:"namespace"`
		Key        string          `json:"key" validate:"required"`
		Value      json.RawMessage `json:"value" validate:"required"`
		Importance float32         `json:"importance,omitempty"`
		ProjectID  *string         `json:"project_id,omitempty"`
		ExpiresIn  *int            `json:"expires_in,omitempty"` // seconds
		Metadata   json.RawMessage `json:"metadata,omitempty"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Default namespace if not provided
	if req.Namespace == "" {
		req.Namespace = "default"
	}

	// Build memory object
	memory := services.ContextMemory{
		UserID:     userContext.UserID.String(),
		Namespace:  req.Namespace,
		Key:        req.Key,
		Value:      req.Value,
		Importance: req.Importance,
		ProjectID:  req.ProjectID,
		Metadata:   req.Metadata,
	}

	// Set expiration if provided
	if req.ExpiresIn != nil && *req.ExpiresIn > 0 {
		expiresAt := time.Now().Add(time.Duration(*req.ExpiresIn) * time.Second)
		memory.ExpiresAt = &expiresAt
	}

	// Store the memory
	err := h.contextMemory.StoreWithMetadata(c.Context(), memory)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "Memory stored successfully",
	})
}

// GetMemory handles GET /api/v1/context/memory/:namespace/:key
func (h *ContextMemoryHandlers) GetMemory(c *fiber.Ctx) error {
	userContext := middleware.GetUserContext(c)
	if userContext == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Not authenticated",
		})
	}

	namespace := c.Params("namespace")
	key := c.Params("key")

	memory, err := h.contextMemory.Get(c.Context(), userContext.UserID.String(), namespace, key)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	if memory == nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Memory not found",
		})
	}

	return c.JSON(memory)
}

// ListMemories handles GET /api/v1/context/memory
func (h *ContextMemoryHandlers) ListMemories(c *fiber.Ctx) error {
	userContext := middleware.GetUserContext(c)
	if userContext == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Not authenticated",
		})
	}

	namespace := c.Query("namespace", "default")
	limitStr := c.Query("limit", "20")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		limit = 20
	}

	memories, err := h.contextMemory.GetByNamespace(c.Context(), userContext.UserID.String(), namespace, limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"memories": memories,
		"count":    len(memories),
	})
}

// SearchMemories handles GET /api/v1/context/memory/search
func (h *ContextMemoryHandlers) SearchMemories(c *fiber.Ctx) error {
	userContext := middleware.GetUserContext(c)
	if userContext == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Not authenticated",
		})
	}

	query := c.Query("q")
	if query == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Search query is required",
		})
	}

	limitStr := c.Query("limit", "20")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		limit = 20
	}

	memories, err := h.contextMemory.Search(c.Context(), userContext.UserID.String(), query, limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"memories": memories,
		"count":    len(memories),
		"query":    query,
	})
}

// DeleteMemory handles DELETE /api/v1/context/memory/:namespace/:key
func (h *ContextMemoryHandlers) DeleteMemory(c *fiber.Ctx) error {
	userContext := middleware.GetUserContext(c)
	if userContext == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Not authenticated",
		})
	}

	namespace := c.Params("namespace")
	key := c.Params("key")

	err := h.contextMemory.Delete(c.Context(), userContext.UserID.String(), namespace, key)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "Memory deleted successfully",
	})
}

// GetRelevantMemories handles GET /api/v1/context/memory/relevant/:sessionId
func (h *ContextMemoryHandlers) GetRelevantMemories(c *fiber.Ctx) error {
	userContext := middleware.GetUserContext(c)
	if userContext == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Not authenticated",
		})
	}

	sessionID := c.Params("sessionId")
	limitStr := c.Query("limit", "10")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		limit = 10
	}

	memories, err := h.contextMemory.GetRelevant(c.Context(), userContext.UserID.String(), sessionID, limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"memories":   memories,
		"count":      len(memories),
		"session_id": sessionID,
	})
}

// UpdateImportance handles PUT /api/v1/context/memory/:id/importance
func (h *ContextMemoryHandlers) UpdateImportance(c *fiber.Ctx) error {
	userContext := middleware.GetUserContext(c)
	if userContext == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Not authenticated",
		})
	}

	memoryID := c.Params("id")
	
	var req struct {
		Importance float32 `json:"importance" validate:"required,min=0,max=1"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	err := h.contextMemory.UpdateImportance(c.Context(), memoryID, req.Importance)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "Importance updated successfully",
	})
}