package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/agentx/agentx-backend/internal/api/middleware"
	"github.com/agentx/agentx-backend/internal/services"
)

// ConnectionHandlers handles connection-related endpoints
type ConnectionHandlers struct {
	connectionService *services.ConnectionService
}

// NewConnectionHandlers creates new connection handlers
func NewConnectionHandlers(connectionService *services.ConnectionService) *ConnectionHandlers {
	return &ConnectionHandlers{
		connectionService: connectionService,
	}
}

// ListConnections handles GET /api/v1/connections
func (h *ConnectionHandlers) ListConnections(c *fiber.Ctx) error {
	userContext := middleware.GetUserContext(c)
	if userContext == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Not authenticated",
		})
	}
	
	providerID := c.Query("provider_id")
	
	// Check if connectionService is nil
	if h.connectionService == nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Connection service not initialized",
		})
	}
	
	connections, err := h.connectionService.ListConnections(c.Context(), userContext.UserID, providerID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}
	
	return c.JSON(fiber.Map{
		"connections": connections,
	})
}

// GetConnection handles GET /api/v1/connections/:id
func (h *ConnectionHandlers) GetConnection(c *fiber.Ctx) error {
	userContext := middleware.GetUserContext(c)
	if userContext == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Not authenticated",
		})
	}
	
	id := c.Params("id")
	
	connection, err := h.connectionService.GetConnection(c.Context(), userContext.UserID, id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Connection not found",
		})
	}
	
	return c.JSON(connection)
}

// CreateConnection handles POST /api/v1/connections
func (h *ConnectionHandlers) CreateConnection(c *fiber.Ctx) error {
	userContext := middleware.GetUserContext(c)
	if userContext == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Not authenticated",
		})
	}
	
	var req struct {
		ProviderID string                 `json:"provider_id" validate:"required"`
		Name       string                 `json:"name" validate:"required"`
		Config     map[string]interface{} `json:"config" validate:"required"`
	}
	
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}
	
	connection, err := h.connectionService.CreateConnection(c.Context(), userContext.UserID, req.ProviderID, req.Name, req.Config)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}
	
	return c.Status(fiber.StatusCreated).JSON(connection)
}

// UpdateConnection handles PUT /api/v1/connections/:id
func (h *ConnectionHandlers) UpdateConnection(c *fiber.Ctx) error {
	userContext := middleware.GetUserContext(c)
	if userContext == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Not authenticated",
		})
	}
	
	id := c.Params("id")
	
	var updates map[string]interface{}
	if err := c.BodyParser(&updates); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}
	
	if err := h.connectionService.UpdateConnection(c.Context(), userContext.UserID, id, updates); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}
	
	return c.JSON(fiber.Map{
		"message": "Connection updated successfully",
	})
}

// DeleteConnection handles DELETE /api/v1/connections/:id
func (h *ConnectionHandlers) DeleteConnection(c *fiber.Ctx) error {
	userContext := middleware.GetUserContext(c)
	if userContext == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Not authenticated",
		})
	}
	
	id := c.Params("id")
	
	if err := h.connectionService.DeleteConnection(c.Context(), userContext.UserID, id); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}
	
	return c.JSON(fiber.Map{
		"message": "Connection deleted successfully",
	})
}

// ToggleConnection handles POST /api/v1/connections/:id/toggle
func (h *ConnectionHandlers) ToggleConnection(c *fiber.Ctx) error {
	userContext := middleware.GetUserContext(c)
	if userContext == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Not authenticated",
		})
	}
	
	id := c.Params("id")
	
	connection, err := h.connectionService.ToggleConnection(c.Context(), userContext.UserID, id)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}
	
	return c.JSON(connection)
}

// TestConnection handles POST /api/v1/connections/:id/test
func (h *ConnectionHandlers) TestConnection(c *fiber.Ctx) error {
	userContext := middleware.GetUserContext(c)
	if userContext == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Not authenticated",
		})
	}
	
	id := c.Params("id")
	
	result, err := h.connectionService.TestConnection(c.Context(), userContext.UserID, id)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}
	
	return c.JSON(result)
}

// GetDefaultConnection handles GET /api/v1/connections/default
func (h *ConnectionHandlers) GetDefaultConnection(c *fiber.Ctx) error {
	userContext := middleware.GetUserContext(c)
	if userContext == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Not authenticated",
		})
	}
	
	// For now, just try to get the first OpenAI connection as default
	// This can be improved to use user preferences later
	providerID := c.Query("provider", "openai")
	
	connection, err := h.connectionService.GetDefaultConnection(c.Context(), userContext.UserID, providerID)
	if err != nil {
		// Return null instead of error if no default is set
		return c.JSON(nil)
	}
	
	return c.JSON(connection)
}

// SetDefaultConnection handles POST /api/v1/connections/:id/set-default
func (h *ConnectionHandlers) SetDefaultConnection(c *fiber.Ctx) error {
	userContext := middleware.GetUserContext(c)
	if userContext == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Not authenticated",
		})
	}
	
	id := c.Params("id")
	
	if err := h.connectionService.SetDefaultConnection(c.Context(), userContext.UserID, id); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}
	
	return c.JSON(fiber.Map{
		"message": "Default connection set successfully",
	})
}