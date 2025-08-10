package handlers

import (
	"encoding/json"

	"github.com/gofiber/fiber/v2"
)

// CallBuiltinToolRequest represents a request to call a built-in MCP tool
type CallBuiltinToolRequest struct {
	ServerID  string          `json:"server_id"`
	ToolName  string          `json:"tool_name"`
	Arguments json.RawMessage `json:"arguments"`
}

// CallBuiltinTool calls a tool on a built-in MCP server
func (h *BuiltinMCPServerHandlers) CallBuiltinTool(c *fiber.Ctx) error {
	userID, err := getUserID(c)
	if err != nil {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Insufficient permissions",
		})
	}

	var req CallBuiltinToolRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Validate request
	if req.ServerID == "" || req.ToolName == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "server_id and tool_name are required",
		})
	}

	// Check if server is enabled for user
	if !h.builtinManager.IsServerEnabledForUser(userID, req.ServerID) {
		// Try to enable it first
		if err := h.builtinManager.SetUserServerEnabled(userID, req.ServerID, true); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Server is not enabled for user",
			})
		}
	}

	// Call the tool
	result, err := h.builtinManager.CallTool(userID, req.ServerID, req.ToolName, req.Arguments)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"result": result,
	})
}

// GetBuiltinTools returns available tools for a built-in server
func (h *BuiltinMCPServerHandlers) GetBuiltinTools(c *fiber.Ctx) error {
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

	tools, err := h.builtinManager.GetServerTools(userID, serverID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"tools": tools,
	})
}