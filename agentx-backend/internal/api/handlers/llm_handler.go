package handlers

import (
	"fmt"
	
	"github.com/gofiber/fiber/v2"
	"github.com/agentx/agentx-backend/internal/llm"
	"github.com/agentx/agentx-backend/internal/api/middleware"
)

// LLMHandler handles LLM-related HTTP requests
type LLMHandler struct {
	llmService *llm.Service
}

// NewLLMHandler creates a new LLM handler
func NewLLMHandler(llmService *llm.Service) *LLMHandler {
	return &LLMHandler{
		llmService: llmService,
	}
}

// RegisterRoutes registers LLM routes
func (h *LLMHandler) RegisterRoutes(api fiber.Router) {
	llmGroup := api.Group("/llm")
	
	// General completion endpoint
	llmGroup.Post("/completions", h.HandleCompletion)
	
	// Task-specific endpoints (optional, for better API discoverability)
	tasksGroup := llmGroup.Group("/tasks")
	tasksGroup.Post("/title-generation", h.HandleTitleGeneration)
	tasksGroup.Post("/summarization", h.HandleSummarization)
	
	// Utility endpoints
	llmGroup.Get("/tasks", h.ListSupportedTasks)
}

// HandleCompletion handles general LLM completion requests
func (h *LLMHandler) HandleCompletion(c *fiber.Ctx) error {
	userContext := middleware.GetUserContext(c)
	if userContext == nil {
		fmt.Printf("[LLM Handler] No user context found\n")
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Authentication required",
		})
	}
	
	fmt.Printf("[LLM Handler] Processing completion request for user: %s\n", userContext.UserID.String())
	
	var req llm.CompletionRequest
	if err := c.BodyParser(&req); err != nil {
		fmt.Printf("[LLM Handler] Failed to parse request body: %v\n", err)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}
	
	fmt.Printf("[LLM Handler] Request parsed - Task: %s, ConnectionID: %s\n", req.Task, req.ConnectionID)
	
	// Process completion
	resp, err := h.llmService.Complete(c.Context(), userContext.UserID.String(), req)
	if err != nil {
		fmt.Printf("[LLM Handler] Completion failed: %v\n", err)
		
		// Handle specific error types
		if llmErr, ok := err.(*llm.LLMError); ok {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": llmErr.Message,
				"code":  llmErr.Code,
			})
		}
		
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}
	
	fmt.Printf("[LLM Handler] Completion successful: %s\n", resp.Result)
	
	return c.JSON(resp)
}

// HandleTitleGeneration handles title generation requests (convenience endpoint)
func (h *LLMHandler) HandleTitleGeneration(c *fiber.Ctx) error {
	userContext := middleware.GetUserContext(c)
	if userContext == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Authentication required",
		})
	}
	
	fmt.Printf("[LLM Handler] Processing title generation for user: %s\n", userContext.UserID.String())
	
	// Parse simplified request
	var req struct {
		SessionID    string `json:"session_id" validate:"required"`
		ConnectionID string `json:"connection_id,omitempty"`
		Options      struct {
			Style     string `json:"style,omitempty"`
			MaxLength *int   `json:"max_length,omitempty"`
		} `json:"options,omitempty"`
	}
	
	if err := c.BodyParser(&req); err != nil {
		fmt.Printf("[LLM Handler] Failed to parse title generation request: %v\n", err)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}
	
	if req.SessionID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "session_id is required",
		})
	}
	
	fmt.Printf("[LLM Handler] Title generation for session: %s, connection: %s\n", req.SessionID, req.ConnectionID)
	
	// Convert to general completion request
	completionReq := llm.CompletionRequest{
		Task: llm.TaskGenerateTitle,
		Context: map[string]interface{}{
			"session_id": req.SessionID,
		},
		ConnectionID: req.ConnectionID,
		Parameters: llm.Parameters{
			MaxTokens: req.Options.MaxLength,
		},
	}
	
	// Add style-specific system prompt if specified
	if req.Options.Style != "" {
		switch req.Options.Style {
		case "concise":
			completionReq.Parameters.SystemPrompt = "Generate a very brief title (3-5 words) that captures the essence of the conversation."
		case "descriptive":
			completionReq.Parameters.SystemPrompt = "Generate a descriptive title (8-12 words) that provides context about the conversation."
		case "technical":
			completionReq.Parameters.SystemPrompt = "Generate a technical title focusing on specific technologies, methods, or concepts discussed."
		}
	}
	
	resp, err := h.llmService.Complete(c.Context(), userContext.UserID.String(), completionReq)
	if err != nil {
		fmt.Printf("[LLM Handler] Title generation failed: %v\n", err)
		
		if llmErr, ok := err.(*llm.LLMError); ok {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": llmErr.Message,
				"code":  llmErr.Code,
			})
		}
		
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}
	
	// Return simplified response for this endpoint
	return c.JSON(fiber.Map{
		"title":         resp.Result,
		"provider":      resp.Provider,
		"connection_id": resp.ConnectionID,
		"duration_ms":   resp.Duration.Milliseconds(),
	})
}

// HandleSummarization handles summarization requests (placeholder)
func (h *LLMHandler) HandleSummarization(c *fiber.Ctx) error {
	return c.Status(fiber.StatusNotImplemented).JSON(fiber.Map{
		"error": "Summarization not yet implemented",
	})
}

// ListSupportedTasks returns the list of supported task types
func (h *LLMHandler) ListSupportedTasks(c *fiber.Ctx) error {
	tasks := h.llmService.GetSupportedTasks()
	
	return c.JSON(fiber.Map{
		"tasks": tasks,
		"count": len(tasks),
	})
}