package llm

import (
	"context"
	"fmt"
	
	"github.com/google/uuid"
	"github.com/agentx/agentx-backend/internal/repository"
)

// ExampleChatService shows how to refactor ChatService to use the LLM Gateway
type ExampleChatService struct {
	gateway     *Gateway
	sessionRepo repository.SessionRepository
	messageRepo repository.MessageRepository
}

// NewExampleChatService creates a new chat service using the LLM Gateway
func NewExampleChatService(gateway *Gateway, sessionRepo repository.SessionRepository, messageRepo repository.MessageRepository) *ExampleChatService {
	return &ExampleChatService{
		gateway:     gateway,
		sessionRepo: sessionRepo,
		messageRepo: messageRepo,
	}
}

// SendMessage sends a message using the centralized LLM Gateway
func (s *ExampleChatService) SendMessage(ctx context.Context, userID uuid.UUID, sessionID, connectionID, message string) (string, error) {
	// Get session messages for context
	messages, err := s.messageRepo.ListBySession(ctx, sessionID)
	if err != nil {
		return "", fmt.Errorf("failed to get messages: %w", err)
	}
	
	// Convert to LLM messages
	llmMessages := make([]Message, 0, len(messages)+1)
	for _, msg := range messages {
		llmMessages = append(llmMessages, Message{
			Role:    msg.Role,
			Content: msg.Content,
		})
	}
	
	// Add the new user message
	llmMessages = append(llmMessages, Message{
		Role:    "user",
		Content: message,
	})
	
	// Create unified request
	req := NewRequest(userID.String(), llmMessages).
		WithSession(sessionID).
		WithConnection(connectionID).
		WithContext(ctx)
	
	// Send to gateway - all routing, middleware, metrics handled automatically
	resp, err := s.gateway.Complete(ctx, req)
	if err != nil {
		return "", fmt.Errorf("gateway request failed: %w", err)
	}
	
	// Save messages to database
	// Save user message
	s.messageRepo.Create(ctx, repository.Message{
		SessionID: sessionID,
		Role:      "user",
		Content:   message,
	})
	
	// Save assistant response
	assistantContent := resp.GetContent()
	s.messageRepo.Create(ctx, repository.Message{
		SessionID: sessionID,
		Role:      "assistant",
		Content:   assistantContent,
	})
	
	// Update session with provider info from response
	s.sessionRepo.Update(ctx, userID, sessionID, map[string]interface{}{
		"provider": resp.Metadata.Provider,
		"model":    resp.Metadata.Model,
	})
	
	return assistantContent, nil
}

// StreamMessage streams a message using the centralized LLM Gateway
func (s *ExampleChatService) StreamMessage(ctx context.Context, userID uuid.UUID, sessionID, connectionID, message string) (<-chan string, error) {
	// Get session messages for context
	messages, err := s.messageRepo.ListBySession(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get messages: %w", err)
	}
	
	// Convert to LLM messages
	llmMessages := make([]Message, 0, len(messages)+1)
	for _, msg := range messages {
		llmMessages = append(llmMessages, Message{
			Role:    msg.Role,
			Content: msg.Content,
		})
	}
	
	// Add the new user message
	llmMessages = append(llmMessages, Message{
		Role:    "user",
		Content: message,
	})
	
	// Create unified request with streaming enabled
	req := NewRequest(userID.String(), llmMessages).
		WithSession(sessionID).
		WithConnection(connectionID).
		WithStreaming().
		WithContext(ctx)
	
	// Get stream from gateway
	stream, err := s.gateway.StreamComplete(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("gateway stream failed: %w", err)
	}
	
	// Create output channel for simplified content
	out := make(chan string)
	
	go func() {
		defer close(out)
		
		var fullContent string
		
		// Process stream chunks
		for chunk := range stream {
			if chunk.Error != nil {
				// Log error but continue
				fmt.Printf("[Stream Error] %v\n", chunk.Error)
				continue
			}
			
			// Extract content from chunk
			for _, choice := range chunk.Choices {
				if choice.Delta.Content != "" {
					fullContent += choice.Delta.Content
					out <- choice.Delta.Content
				}
			}
		}
		
		// Save messages after streaming completes
		// Save user message
		s.messageRepo.Create(context.Background(), repository.Message{
			SessionID: sessionID,
			Role:      "user",
			Content:   message,
		})
		
		// Save assistant response
		if fullContent != "" {
			s.messageRepo.Create(context.Background(), repository.Message{
				SessionID: sessionID,
				Role:      "assistant",
				Content:   fullContent,
			})
		}
	}()
	
	return out, nil
}

// GenerateTitle generates a title using the LLM Gateway
func (s *ExampleChatService) GenerateTitle(ctx context.Context, userID uuid.UUID, sessionID, connectionID string) (string, error) {
	// Get session messages
	messages, err := s.messageRepo.ListBySession(ctx, sessionID)
	if err != nil {
		return "", fmt.Errorf("failed to get messages: %w", err)
	}
	
	if len(messages) == 0 {
		return "", fmt.Errorf("no messages to generate title from")
	}
	
	// Create a summary of the conversation
	var summary string
	for i, msg := range messages {
		if i > 5 { // Limit to first few messages
			break
		}
		summary += fmt.Sprintf("%s: %s\n", msg.Role, msg.Content)
		if len(summary) > 1000 { // Limit summary length
			break
		}
	}
	
	// Create title generation request
	titleMessages := []Message{
		{
			Role:    "system",
			Content: "Generate a short, descriptive title (3-7 words) for this conversation. Respond with ONLY the title.",
		},
		{
			Role:    "user",
			Content: fmt.Sprintf("Conversation:\n%s\n\nTitle:", summary),
		},
	}
	
	req := NewRequest(userID.String(), titleMessages).
		WithConnection(connectionID).
		WithTemperature(0.3).
		WithMaxTokens(20).
		WithContext(ctx)
	
	// Send to gateway
	resp, err := s.gateway.Complete(ctx, req)
	if err != nil {
		return "", fmt.Errorf("title generation failed: %w", err)
	}
	
	return resp.GetContent(), nil
}

// GetMetrics returns current LLM usage metrics
func (s *ExampleChatService) GetMetrics() map[string]interface{} {
	return s.gateway.GetMetrics()
}

// HealthCheck performs a health check on all providers
func (s *ExampleChatService) HealthCheck(ctx context.Context) map[string]HealthStatus {
	return s.gateway.HealthCheck(ctx)
}