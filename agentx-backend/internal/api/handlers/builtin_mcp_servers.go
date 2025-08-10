package handlers

import (
	"github.com/agentx/agentx-backend/internal/mcp"
	"github.com/agentx/agentx-backend/internal/services"
	"github.com/gofiber/fiber/v2"
)

// BuiltinMCPServerHandlers handles built-in MCP server related requests
type BuiltinMCPServerHandlers struct {
	builtinManager *mcp.BuiltinMCPManager
	mcpService     *services.MCPService
}

// NewBuiltinMCPServerHandlers creates new built-in MCP server handlers
func NewBuiltinMCPServerHandlers(builtinManager *mcp.BuiltinMCPManager, mcpService *services.MCPService) *BuiltinMCPServerHandlers {
	return &BuiltinMCPServerHandlers{
		builtinManager: builtinManager,
		mcpService:     mcpService,
	}
}

// ListBuiltinServers returns all available built-in MCP servers
func (h *BuiltinMCPServerHandlers) ListBuiltinServers(c *fiber.Ctx) error {
	servers := h.builtinManager.GetBuiltinServers()
	
	return c.JSON(fiber.Map{
		"servers": servers,
	})
}

// GetBuiltinServer returns a specific built-in MCP server
func (h *BuiltinMCPServerHandlers) GetBuiltinServer(c *fiber.Ctx) error {
	serverID := c.Params("id")
	if serverID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Server ID is required",
		})
	}

	server, err := h.builtinManager.GetBuiltinServer(serverID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Built-in server not found",
		})
	}

	return c.JSON(server)
}

// GetUserBuiltinServers returns built-in servers with user-specific status
func (h *BuiltinMCPServerHandlers) GetUserBuiltinServers(c *fiber.Ctx) error {
	userID, err := getUserID(c)
	if err != nil {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Insufficient permissions",
		})
	}

	servers := h.builtinManager.GetBuiltinServers()
	
	// Add user-specific enabled status
	for i := range servers {
		servers[i].Enabled = h.builtinManager.IsServerEnabledForUser(userID, servers[i].ID)
	}

	return c.JSON(fiber.Map{
		"servers": servers,
	})
}

// EnableBuiltinServerRequest represents the request to enable/disable a built-in server
type EnableBuiltinServerRequest struct {
	Enabled bool `json:"enabled"`
}

// ToggleBuiltinServer enables or disables a built-in MCP server for the current user
func (h *BuiltinMCPServerHandlers) ToggleBuiltinServer(c *fiber.Ctx) error {
	userID, err := getUserID(c)
	if err != nil {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Insufficient permissions",
		})
	}

	serverID := c.Params("id")
	if serverID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Server ID is required",
		})
	}

	var req EnableBuiltinServerRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Toggle the server
	if err := h.builtinManager.SetUserServerEnabled(userID, serverID, req.Enabled); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	// Get updated server info
	server, err := h.builtinManager.GetBuiltinServer(serverID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to get updated server info",
		})
	}

	// Add user-specific status
	server.Enabled = req.Enabled

	return c.JSON(fiber.Map{
		"server": server,
		"message": "Built-in server status updated successfully",
	})
}

// ConvertToRegularServer converts a built-in server to a regular user-managed MCP server
func (h *BuiltinMCPServerHandlers) ConvertToRegularServer(c *fiber.Ctx) error {
	userID, err := getUserID(c)
	if err != nil {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Insufficient permissions",
		})
	}

	serverID := c.Params("id")
	if serverID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Server ID is required",
		})
	}

	// Convert to regular server configuration
	serverConfig, err := h.builtinManager.ConvertToRegularMCPServer(userID, serverID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	// Create regular MCP server
	regularServer, err := h.mcpService.CreateServer(c.Context(), userID, serverConfig)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to create regular MCP server",
		})
	}

	// Disable the built-in server for this user
	if err := h.builtinManager.SetUserServerEnabled(userID, serverID, false); err != nil {
		// Log warning but don't fail the request
		c.Locals("warning", "Failed to disable built-in server: "+err.Error())
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"server":  regularServer,
		"message": "Successfully converted built-in server to regular server",
	})
}

// GetBuiltinServerStatus returns the status of built-in servers for a user
func (h *BuiltinMCPServerHandlers) GetBuiltinServerStatus(c *fiber.Ctx) error {
	userID, err := getUserID(c)
	if err != nil {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Insufficient permissions",
		})
	}

	enabledServers := h.builtinManager.GetUserEnabledServers(userID)
	
	return c.JSON(fiber.Map{
		"enabled_count":   len(enabledServers),
		"enabled_servers": enabledServers,
	})
}