package services

import (
	"context"
	"fmt"
	"strings"

	"github.com/agentx/agentx-backend/internal/providers"
	"github.com/agentx/agentx-backend/internal/repository"
)

// ChatService manages chat sessions and interactions
type ChatService struct {
	sessionRepo repository.SessionRepository
	messageRepo repository.MessageRepository
	providers   *providers.Registry
}

// NewChatService creates a new chat service
func NewChatService(providers *providers.Registry, sessionRepo repository.SessionRepository, messageRepo repository.MessageRepository) *ChatService {
	return &ChatService{
		sessionRepo: sessionRepo,
		messageRepo: messageRepo,
		providers:   providers,
	}
}

// CreateSession creates a new chat session
func (s *ChatService) CreateSession(ctx context.Context, title string) (*repository.Session, error) {
	session := repository.Session{
		Title: title,
	}
	
	sessionID, err := s.sessionRepo.Create(ctx, session)
	if err != nil {
		return nil, err
	}
	
	session.ID = sessionID
	return &session, nil
}

// GetSession retrieves a session by ID
func (s *ChatService) GetSession(ctx context.Context, id string) (*repository.Session, error) {
	return s.sessionRepo.Get(ctx, id)
}

// GetSessions returns all sessions
func (s *ChatService) GetSessions(ctx context.Context) ([]*repository.Session, error) {
	return s.sessionRepo.List(ctx)
}

// DeleteSession deletes a session
func (s *ChatService) DeleteSession(ctx context.Context, id string) error {
	return s.sessionRepo.Delete(ctx, id)
}

// GetSessionMessages returns messages for a session
func (s *ChatService) GetSessionMessages(ctx context.Context, sessionID string) ([]repository.Message, error) {
	return s.messageRepo.ListBySession(ctx, sessionID)
}

// SaveMessage saves a message to a session
func (s *ChatService) SaveMessage(ctx context.Context, message repository.Message) (string, error) {
	return s.messageRepo.Create(ctx, message)
}

// SendToProvider sends messages to a provider and returns the response
func (s *ChatService) SendToProvider(ctx context.Context, sessionID string, messages []repository.Message, providerID, model string) (*providers.CompletionResponse, error) {
	// Get provider
	provider := s.providers.Get(providerID)
	if provider == nil {
		return nil, fmt.Errorf("provider not found: %s", providerID)
	}
	
	// Convert repository messages to provider messages
	providerMessages := make([]providers.Message, len(messages))
	for i, msg := range messages {
		providerMessages[i] = providers.Message{
			Role:    msg.Role,
			Content: msg.Content,
		}
		
		// Handle function calls and tool calls if needed
		if msg.FunctionCall.Valid {
			// Parse function call from JSON
			// providerMessages[i].FunctionCall = ...
		}
		if msg.ToolCalls.Valid {
			// Parse tool calls from JSON
			// providerMessages[i].ToolCalls = ...
		}
	}
	
	// Create completion request
	req := providers.CompletionRequest{
		Messages: providerMessages,
		Model:    model,
	}
	
	// Send to provider
	return provider.Complete(ctx, req)
}

// StreamResponse handles streaming chat responses
func (s *ChatService) StreamResponse(ctx context.Context, sessionID string, message string, providerID, model string) (<-chan providers.StreamChunk, error) {
	// Get session messages
	messages, err := s.GetSessionMessages(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	
	// Add user message
	userMsg := repository.Message{
		SessionID: sessionID,
		Role:      "user",
		Content:   message,
	}
	_, err = s.SaveMessage(ctx, userMsg)
	if err != nil {
		return nil, err
	}
	
	// Get provider
	provider := s.providers.Get(providerID)
	if provider == nil {
		return nil, fmt.Errorf("provider not found: %s", providerID)
	}
	
	// Convert messages to provider format
	providerMessages := make([]providers.Message, len(messages)+1)
	for i, msg := range messages {
		providerMessages[i] = providers.Message{
			Role:    msg.Role,
			Content: msg.Content,
		}
	}
	providerMessages[len(messages)] = providers.Message{
		Role:    "user",
		Content: message,
	}
	
	// Create completion request
	req := providers.CompletionRequest{
		Messages: providerMessages,
		Model:    model,
		Stream:   true,
	}
	
	// Get streaming response
	streamChan, err := provider.StreamComplete(ctx, req)
	if err != nil {
		return nil, err
	}
	
	// Create a new channel to intercept and save the response
	processedChan := make(chan providers.StreamChunk)
	
	go func() {
		defer close(processedChan)
		
		var contentBuilder strings.Builder
		
		for chunk := range streamChan {
			// Forward the chunk
			processedChan <- chunk
			
			// Build the content
			if chunk.Delta != "" {
				contentBuilder.WriteString(chunk.Delta)
			}
			
			// When streaming is complete, save the assistant message
			if chunk.FinishReason != "" {
				assistantMsg := repository.Message{
					SessionID: sessionID,
					Role:      "assistant",
					Content:   contentBuilder.String(),
				}
				s.SaveMessage(context.Background(), assistantMsg)
				
				// Update session with provider info
				s.sessionRepo.Update(ctx, sessionID, map[string]interface{}{
					"provider": providerID,
					"model":    model,
				})
			}
		}
	}()
	
	return processedChan, nil
}