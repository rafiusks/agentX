package handlers

import (
	"fmt"

	"github.com/gofiber/fiber/v2"
	"github.com/agentx/agentx-backend/internal/api/middleware"
	"github.com/agentx/agentx-backend/internal/services"
)

// CreateSession creates a new chat session
func CreateSession(svc *services.Services) fiber.Handler {
	return func(c *fiber.Ctx) error {
		userContext := middleware.GetUserContext(c)
		if userContext == nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Not authenticated",
			})
		}
		
		var req struct {
			Title string `json:"title"`
		}
		
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Invalid request body",
			})
		}
		
		if req.Title == "" {
			req.Title = "New Chat"
		}
		
		session, err := svc.Orchestrator.CreateSession(c.Context(), userContext.UserID, req.Title)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": err.Error(),
			})
		}
		
		return c.JSON(session)
	}
}

// GetSessions returns all sessions
func GetSessions(svc *services.Services) fiber.Handler {
	return func(c *fiber.Ctx) error {
		userContext := middleware.GetUserContext(c)
		if userContext == nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Not authenticated",
			})
		}
		
		sessions, err := svc.Orchestrator.ListSessions(c.Context(), userContext.UserID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": err.Error(),
			})
		}
		
		return c.JSON(fiber.Map{
			"sessions": sessions,
		})
	}
}

// GetSession returns a specific session
func GetSession(svc *services.Services) fiber.Handler {
	return func(c *fiber.Ctx) error {
		userContext := middleware.GetUserContext(c)
		if userContext == nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Not authenticated",
			})
		}
		
		sessionID := c.Params("id")
		
		session, err := svc.Orchestrator.GetSession(c.Context(), userContext.UserID, sessionID)
		if err != nil {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Session not found",
			})
		}
		
		return c.JSON(session)
	}
}

// DeleteSession deletes a session
func DeleteSession(svc *services.Services) fiber.Handler {
	return func(c *fiber.Ctx) error {
		userContext := middleware.GetUserContext(c)
		if userContext == nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Not authenticated",
			})
		}
		
		sessionID := c.Params("id")
		
		err := svc.Orchestrator.DeleteSession(c.Context(), userContext.UserID, sessionID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": err.Error(),
			})
		}
		
		return c.JSON(fiber.Map{
			"message": "Session deleted successfully",
		})
	}
}

// GetSessionMessages returns messages for a session
func GetSessionMessages(svc *services.Services) fiber.Handler {
	return func(c *fiber.Ctx) error {
		sessionID := c.Params("id")
		
		messages, err := svc.Orchestrator.GetMessages(c.Context(), sessionID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": err.Error(),
			})
		}
		
		return c.JSON(fiber.Map{
			"messages": messages,
		})
	}
}

// UpdateSession updates a session
func UpdateSession(svc *services.Services) fiber.Handler {
	return func(c *fiber.Ctx) error {
		userContext := middleware.GetUserContext(c)
		if userContext == nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Not authenticated",
			})
		}
		
		sessionID := c.Params("id")
		
		var req struct {
			Title    string                 `json:"title"`
			Metadata map[string]interface{} `json:"metadata"`
		}
		
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Invalid request body",
			})
		}
		
		// Build updates map
		updates := make(map[string]interface{})
		if req.Title != "" {
			updates["title"] = req.Title
		}
		if req.Metadata != nil {
			updates["metadata"] = req.Metadata
		}
		
		// Update session
		err := svc.Orchestrator.UpdateSession(c.Context(), userContext.UserID, sessionID, updates)
		if err != nil {
			fmt.Printf("[UpdateSession] Error updating session: %v\n", err)
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to update session",
			})
		}
		
		// Get updated session
		session, err := svc.Orchestrator.GetSession(c.Context(), userContext.UserID, sessionID)
		if err != nil {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Session not found",
			})
		}
		
		return c.JSON(session)
	}
}


