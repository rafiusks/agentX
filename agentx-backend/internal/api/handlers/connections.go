package handlers

import (
	"github.com/gofiber/fiber/v2"
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
	providerID := c.Query("provider_id")
	
	// Check if connectionService is nil
	if h.connectionService == nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Connection service not initialized",
		})
	}
	
	connections, err := h.connectionService.ListConnections(c.Context(), providerID)
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
	id := c.Params("id")
	
	connection, err := h.connectionService.GetConnection(c.Context(), id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Connection not found",
		})
	}
	
	return c.JSON(connection)
}

// CreateConnection handles POST /api/v1/connections
func (h *ConnectionHandlers) CreateConnection(c *fiber.Ctx) error {
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
	
	connection, err := h.connectionService.CreateConnection(c.Context(), req.ProviderID, req.Name, req.Config)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}
	
	return c.Status(fiber.StatusCreated).JSON(connection)
}

// UpdateConnection handles PUT /api/v1/connections/:id
func (h *ConnectionHandlers) UpdateConnection(c *fiber.Ctx) error {
	id := c.Params("id")
	
	var updates map[string]interface{}
	if err := c.BodyParser(&updates); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}
	
	if err := h.connectionService.UpdateConnection(c.Context(), id, updates); err != nil {
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
	id := c.Params("id")
	
	if err := h.connectionService.DeleteConnection(c.Context(), id); err != nil {
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
	id := c.Params("id")
	
	connection, err := h.connectionService.ToggleConnection(c.Context(), id)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}
	
	return c.JSON(connection)
}

// TestConnection handles POST /api/v1/connections/:id/test
func (h *ConnectionHandlers) TestConnection(c *fiber.Ctx) error {
	id := c.Params("id")
	
	result, err := h.connectionService.TestConnection(c.Context(), id)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}
	
	return c.JSON(result)
}

// SetDefaultConnection handles POST /api/v1/connections/:id/set-default
func (h *ConnectionHandlers) SetDefaultConnection(c *fiber.Ctx) error {
	id := c.Params("id")
	
	if err := h.connectionService.SetDefaultConnection(c.Context(), id); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}
	
	return c.JSON(fiber.Map{
		"message": "Default connection set successfully",
	})
}