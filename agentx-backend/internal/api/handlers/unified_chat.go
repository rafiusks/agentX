package handlers

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"
	"github.com/agentx/agentx-backend/internal/api/models"
	"github.com/agentx/agentx-backend/internal/providers"
	"github.com/agentx/agentx-backend/internal/services"
)

// UnifiedChatHandler handles unified chat requests
type UnifiedChatHandler struct {
	chatService *services.UnifiedChatService
}

// NewUnifiedChatHandler creates a new unified chat handler
func NewUnifiedChatHandler(chatService *services.UnifiedChatService) *UnifiedChatHandler {
	return &UnifiedChatHandler{
		chatService: chatService,
	}
}

// Chat handles POST /api/v1/chat
func (h *UnifiedChatHandler) Chat(c *fiber.Ctx) error {
	var req models.UnifiedChatRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}
	
	// Get response
	resp, err := h.chatService.Chat(c.Context(), req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}
	
	return c.JSON(resp)
}

// StreamChat handles WebSocket /api/v1/chat/stream
func (h *UnifiedChatHandler) StreamChat(c *websocket.Conn) {
	defer c.Close()
	
	// Read the request
	var req models.UnifiedChatRequest
	if err := c.ReadJSON(&req); err != nil {
		c.WriteJSON(models.UnifiedStreamChunk{
			Type: "error",
			Error: &models.UnifiedError{
				Code:    "invalid_request",
				Message: "Failed to parse request",
				Type:    models.ErrorTypeInvalid,
			},
		})
		return
	}
	
	// Get context from locals
	ctx := context.Background()
	if fc, ok := c.Locals("fiber_ctx").(*fiber.Ctx); ok {
		ctx = fc.UserContext()
	}
	
	// Get stream
	stream, err := h.chatService.StreamChat(ctx, req)
	if err != nil {
		c.WriteJSON(models.UnifiedStreamChunk{
			Type: "error",
			Error: &models.UnifiedError{
				Code:    "stream_error",
				Message: err.Error(),
				Type:    models.ErrorTypeProvider,
			},
		})
		return
	}
	
	// Stream chunks to WebSocket
	for chunk := range stream {
		if err := c.WriteJSON(chunk); err != nil {
			// Client disconnected
			break
		}
	}
}

// StreamChatSSE handles SSE GET /api/v1/chat/stream
func (h *UnifiedChatHandler) StreamChatSSE(c *fiber.Ctx) error {
	// Parse request from query params or body
	var req models.UnifiedChatRequest
	
	// Try to parse from body first
	if err := c.BodyParser(&req); err != nil {
		// Fallback to query params for simple requests
		messagesJSON := c.Query("messages")
		if messagesJSON != "" {
			if err := json.Unmarshal([]byte(messagesJSON), &req.Messages); err != nil {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
					"error": "Invalid messages format",
				})
			}
		} else {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Messages required",
			})
		}
	}
	
	// Set SSE headers
	c.Set("Content-Type", "text/event-stream")
	c.Set("Cache-Control", "no-cache")
	c.Set("Connection", "keep-alive")
	c.Set("Transfer-Encoding", "chunked")
	
	// Get stream
	stream, err := h.chatService.StreamChat(c.Context(), req)
	if err != nil {
		fmt.Fprintf(c, "event: error\ndata: %s\n\n", err.Error())
		return nil
	}
	
	// Create a writer with flushing capability
	c.Context().SetBodyStreamWriter(func(w *bufio.Writer) {
		// Stream chunks as SSE
		for chunk := range stream {
			data, _ := json.Marshal(chunk)
			fmt.Fprintf(w, "event: message\ndata: %s\n\n", string(data))
			w.Flush()
		}
		
		// Send done event
		fmt.Fprintf(w, "event: done\ndata: {}\n\n")
		w.Flush()
	})
	
	return nil
}

// GetModels handles GET /api/v1/models
func (h *UnifiedChatHandler) GetModels(c *fiber.Ctx) error {
	models, err := h.chatService.GetAvailableModels(c.Context())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}
	
	return c.JSON(models)
}

// Legacy endpoints for compatibility

// ChatCompletions handles POST /api/v1/chat/completions (OpenAI-compatible)
func (h *UnifiedChatHandler) ChatCompletions(c *fiber.Ctx) error {
	// Parse OpenAI-style request
	var openAIReq struct {
		Messages    []providers.Message `json:"messages"`
		Model       string             `json:"model"`
		Temperature *float32           `json:"temperature,omitempty"`
		MaxTokens   *int               `json:"max_tokens,omitempty"`
		Stream      bool               `json:"stream"`
	}
	
	if err := c.BodyParser(&openAIReq); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}
	
	// Convert to unified request
	unifiedReq := models.UnifiedChatRequest{
		Messages:    openAIReq.Messages,
		Temperature: openAIReq.Temperature,
		MaxTokens:   openAIReq.MaxTokens,
	}
	
	// If model is specified, set it as a preference
	if openAIReq.Model != "" {
		parts := strings.SplitN(openAIReq.Model, "/", 2)
		if len(parts) == 2 {
			unifiedReq.Preferences.Provider = parts[0]
			unifiedReq.Preferences.Model = parts[1]
		} else {
			unifiedReq.Preferences.Model = openAIReq.Model
		}
	}
	
	if openAIReq.Stream {
		// Handle streaming
		c.Set("Content-Type", "text/event-stream")
		c.Set("Cache-Control", "no-cache")
		c.Set("Connection", "keep-alive")
		
		stream, err := h.chatService.StreamChat(c.Context(), unifiedReq)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": err.Error(),
			})
		}
		
		c.Context().SetBodyStreamWriter(func(w *bufio.Writer) {
			for chunk := range stream {
				// Convert to OpenAI format
				openAIChunk := h.convertToOpenAIStreamChunk(chunk)
				data, _ := json.Marshal(openAIChunk)
				fmt.Fprintf(w, "data: %s\n\n", string(data))
				w.Flush()
			}
			fmt.Fprintf(w, "data: [DONE]\n\n")
			w.Flush()
		})
		
		return nil
	} else {
		// Handle non-streaming
		resp, err := h.chatService.Chat(c.Context(), unifiedReq)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": err.Error(),
			})
		}
		
		// Convert to OpenAI format
		openAIResp := h.convertToOpenAIResponse(resp)
		return c.JSON(openAIResp)
	}
}

// convertToOpenAIResponse converts unified response to OpenAI format
func (h *UnifiedChatHandler) convertToOpenAIResponse(resp *models.UnifiedChatResponse) map[string]interface{} {
	return map[string]interface{}{
		"id":      resp.ID,
		"object":  "chat.completion",
		"created": time.Now().Unix(),
		"model":   fmt.Sprintf("%s/%s", resp.Metadata.Provider, resp.Metadata.Model),
		"choices": []map[string]interface{}{
			{
				"index": 0,
				"message": map[string]interface{}{
					"role":    resp.Role,
					"content": resp.Content,
				},
				"finish_reason": "stop",
			},
		},
		"usage": map[string]interface{}{
			"prompt_tokens":     resp.Usage.PromptTokens,
			"completion_tokens": resp.Usage.CompletionTokens,
			"total_tokens":      resp.Usage.TotalTokens,
		},
	}
}

// convertToOpenAIStreamChunk converts unified stream chunk to OpenAI format
func (h *UnifiedChatHandler) convertToOpenAIStreamChunk(chunk models.UnifiedStreamChunk) map[string]interface{} {
	switch chunk.Type {
	case "content":
		return map[string]interface{}{
			"id":      "chatcmpl-" + time.Now().Format("20060102150405"),
			"object":  "chat.completion.chunk",
			"created": time.Now().Unix(),
			"model":   fmt.Sprintf("%s/%s", chunk.Metadata.Provider, chunk.Metadata.Model),
			"choices": []map[string]interface{}{
				{
					"index": 0,
					"delta": map[string]interface{}{
						"content": chunk.Content,
					},
				},
			},
		}
	case "done":
		return map[string]interface{}{
			"id":      "chatcmpl-" + time.Now().Format("20060102150405"),
			"object":  "chat.completion.chunk",
			"created": time.Now().Unix(),
			"choices": []map[string]interface{}{
				{
					"index":         0,
					"delta":         map[string]interface{}{},
					"finish_reason": "stop",
				},
			},
		}
	default:
		return map[string]interface{}{}
	}
}