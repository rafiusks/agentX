package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/agentx/agentx-backend/internal/repository"
	"github.com/agentx/agentx-backend/internal/services"
)

// CreateSession creates a new chat session
func CreateSession(svc *services.Services) fiber.Handler {
	return func(c *fiber.Ctx) error {
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
		
		session, err := svc.Chat.CreateSession(c.UserContext(), req.Title)
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
		sessions, err := svc.Chat.GetSessions(c.UserContext())
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
		sessionID := c.Params("id")
		
		session, err := svc.Chat.GetSession(c.UserContext(), sessionID)
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
		sessionID := c.Params("id")
		
		err := svc.Chat.DeleteSession(c.UserContext(), sessionID)
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
		
		messages, err := svc.Chat.GetSessionMessages(c.UserContext(), sessionID)
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

// SendMessage sends a message in a session (legacy endpoint)
func SendMessage(svc *services.Services) fiber.Handler {
	return func(c *fiber.Ctx) error {
		sessionID := c.Params("id")
		
		var req struct {
			Content  string `json:"content"`
			Provider string `json:"provider,omitempty"`
			Model    string `json:"model,omitempty"`
		}
		
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Invalid request body",
			})
		}
		
		// Create message
		userMsg := repository.Message{
			SessionID: sessionID,
			Role:      "user",
			Content:   req.Content,
		}
		
		// Get session history
		messages, err := svc.Chat.GetSessionMessages(c.UserContext(), sessionID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": err.Error(),
			})
		}
		
		// Save user message
		userMsgID, err := svc.Chat.SaveMessage(c.UserContext(), userMsg)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": err.Error(),
			})
		}
		
		// Prepare provider messages
		providerMessages := make([]repository.Message, len(messages))
		copy(providerMessages, messages)
		providerMessages = append(providerMessages, userMsg)
		
		// Send to provider (using legacy method)
		response, err := svc.Chat.SendToProvider(c.UserContext(), sessionID, providerMessages, req.Provider, req.Model)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": err.Error(),
			})
		}
		
		// Extract content from response
		content := ""
		if len(response.Choices) > 0 {
			content = response.Choices[0].Message.Content
		}
		
		// Save assistant message
		assistantMsg := repository.Message{
			SessionID: sessionID,
			Role:      "assistant",
			Content:   content,
		}
		
		assistantMsgID, err := svc.Chat.SaveMessage(c.UserContext(), assistantMsg)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": err.Error(),
			})
		}
		
		return c.JSON(fiber.Map{
			"user_message": fiber.Map{
				"id":      userMsgID,
				"content": req.Content,
			},
			"assistant_message": fiber.Map{
				"id":      assistantMsgID,
				"content": content,
			},
			"usage": response.Usage,
		})
	}
}