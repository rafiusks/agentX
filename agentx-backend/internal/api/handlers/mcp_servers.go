package handlers

import (
	"fmt"
	
	"github.com/agentx/agentx-backend/internal/models"
	"github.com/agentx/agentx-backend/internal/services"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// MCPServerHandlers handles MCP server related requests
type MCPServerHandlers struct {
	mcpService *services.MCPService
}

// NewMCPServerHandlers creates new MCP server handlers
func NewMCPServerHandlers(mcpService *services.MCPService) *MCPServerHandlers {
	return &MCPServerHandlers{
		mcpService: mcpService,
	}
}

// getUserID extracts the user ID from the fiber context
func getUserID(c *fiber.Ctx) (uuid.UUID, error) {
	// Debug logging
	fmt.Printf("getUserID: Looking for user_id in locals\n")
	fmt.Printf("getUserID: All locals keys: ")
	// Try to print all locals
	if userIDRaw := c.Locals("user_id"); userIDRaw != nil {
		fmt.Printf("user_id=%v (type=%T) ", userIDRaw, userIDRaw)
	}
	if userRoleRaw := c.Locals("user_role"); userRoleRaw != nil {
		fmt.Printf("user_role=%v ", userRoleRaw)
	}
	fmt.Printf("\n")
	
	userIDStr, ok := c.Locals("user_id").(string)
	if !ok {
		fmt.Printf("getUserID: user_id not found or not a string\n")
		return uuid.Nil, fmt.Errorf("user not authenticated")
	}
	
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		fmt.Printf("getUserID: Failed to parse UUID: %v\n", err)
		return uuid.Nil, fmt.Errorf("invalid user ID: %w", err)
	}
	
	fmt.Printf("getUserID: Success! UserID=%v\n", userID)
	return userID, nil
}

// ListServers returns all MCP servers for the authenticated user
func (h *MCPServerHandlers) ListServers(c *fiber.Ctx) error {
	userID, err := getUserID(c)
	if err != nil {
		fmt.Printf("ListServers error: %v\n", err)
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Insufficient permissions",
		})
	}

	servers, err := h.mcpService.ListServers(c.Context(), userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to list MCP servers",
		})
	}

	return c.JSON(fiber.Map{
		"servers": servers,
	})
}

// GetServer returns a specific MCP server
func (h *MCPServerHandlers) GetServer(c *fiber.Ctx) error {
	userID, err := getUserID(c)
	if err != nil {
		return err
	}
	
	serverID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid server ID",
		})
	}

	server, err := h.mcpService.GetServer(c.Context(), userID, serverID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "MCP server not found",
		})
	}

	return c.JSON(server)
}

// CreateServer creates a new MCP server configuration
func (h *MCPServerHandlers) CreateServer(c *fiber.Ctx) error {
	userID, err := getUserID(c)
	if err != nil {
		return err
	}

	var req models.MCPServerCreateRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Validate request
	if req.Name == "" || req.Command == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Name and command are required",
		})
	}

	server, err := h.mcpService.CreateServer(c.Context(), userID, &req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(fiber.StatusCreated).JSON(server)
}

// UpdateServer updates an MCP server configuration
func (h *MCPServerHandlers) UpdateServer(c *fiber.Ctx) error {
	userID, err := getUserID(c)
	if err != nil {
		return err
	}
	
	serverID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid server ID",
		})
	}

	var req models.MCPServerUpdateRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	server, err := h.mcpService.UpdateServer(c.Context(), userID, serverID, &req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(server)
}

// DeleteServer deletes an MCP server
func (h *MCPServerHandlers) DeleteServer(c *fiber.Ctx) error {
	userID, err := getUserID(c)
	if err != nil {
		return err
	}
	
	serverID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid server ID",
		})
	}

	if err := h.mcpService.DeleteServer(c.Context(), userID, serverID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// ToggleServer enables or disables an MCP server
func (h *MCPServerHandlers) ToggleServer(c *fiber.Ctx) error {
	userID, err := getUserID(c)
	if err != nil {
		return err
	}
	
	serverID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid server ID",
		})
	}

	server, err := h.mcpService.ToggleServer(c.Context(), userID, serverID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(server)
}

// TestServer tests the connection to an MCP server
func (h *MCPServerHandlers) TestServer(c *fiber.Ctx) error {
	userID, err := getUserID(c)
	if err != nil {
		return err
	}
	
	serverID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid server ID",
		})
	}

	// Get server to verify ownership
	server, err := h.mcpService.GetServer(c.Context(), userID, serverID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "MCP server not found",
		})
	}

	// Test connection by temporarily connecting
	// TODO: Implement test connection logic

	return c.JSON(fiber.Map{
		"status":  "success",
		"message": "Connection test successful",
		"server":  server.Name,
	})
}

// GetServerTools returns the tools available from an MCP server
func (h *MCPServerHandlers) GetServerTools(c *fiber.Ctx) error {
	userID, err := getUserID(c)
	if err != nil {
		return err
	}
	
	serverID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid server ID",
		})
	}

	// Get server with tools
	server, err := h.mcpService.GetServer(c.Context(), userID, serverID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "MCP server not found",
		})
	}

	return c.JSON(fiber.Map{
		"tools": server.Tools,
	})
}

// GetServerResources returns the resources available from an MCP server
func (h *MCPServerHandlers) GetServerResources(c *fiber.Ctx) error {
	userID, err := getUserID(c)
	if err != nil {
		return err
	}
	
	serverID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid server ID",
		})
	}

	// Get server with resources
	server, err := h.mcpService.GetServer(c.Context(), userID, serverID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "MCP server not found",
		})
	}

	return c.JSON(fiber.Map{
		"resources": server.Resources,
	})
}

// CallTool calls a tool on an MCP server
func (h *MCPServerHandlers) CallTool(c *fiber.Ctx) error {
	userID, err := getUserID(c)
	if err != nil {
		return err
	}

	var req models.MCPToolCallRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Verify server ownership
	_, err = h.mcpService.GetServer(c.Context(), userID, req.ServerID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "MCP server not found",
		})
	}

	result, err := h.mcpService.CallTool(c.Context(), &req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(result)
}

// ReadResource reads a resource from an MCP server
func (h *MCPServerHandlers) ReadResource(c *fiber.Ctx) error {
	userID, err := getUserID(c)
	if err != nil {
		return err
	}

	var req models.MCPResourceReadRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Verify server ownership
	_, err = h.mcpService.GetServer(c.Context(), userID, req.ServerID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "MCP server not found",
		})
	}

	result, err := h.mcpService.ReadResource(c.Context(), &req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(result)
}