package services

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/agentx/agentx-backend/internal/api/models"
	"github.com/agentx/agentx-backend/internal/llm"
	"github.com/agentx/agentx-backend/internal/repository"
)

// UnifiedChatAdapter wraps the LLM Gateway to provide UnifiedChatService compatibility
// This is a transitional adapter that allows gradual migration from the old system
type UnifiedChatAdapter struct {
	gateway       *llm.Gateway
	sessionRepo   repository.SessionRepository
	messageRepo   repository.MessageRepository
	contextMemory *ContextMemoryService
	
	// Keep reference to connection service for provider initialization
	connectionService *ConnectionService
}

// NewUnifiedChatAdapter creates an adapter that makes LLMService compatible with UnifiedChatService interface
func NewUnifiedChatAdapter(
	gateway *llm.Gateway,
	sessionRepo repository.SessionRepository,
	messageRepo repository.MessageRepository,
	contextMemory *ContextMemoryService,
	connectionService *ConnectionService,
) *UnifiedChatAdapter {
	return &UnifiedChatAdapter{
		gateway:           gateway,
		sessionRepo:       sessionRepo,
		messageRepo:       messageRepo,
		contextMemory:     contextMemory,
		connectionService: connectionService,
	}
}

// Chat handles a unified chat request through the gateway
func (a *UnifiedChatAdapter) Chat(ctx context.Context, req models.UnifiedChatRequest) (*models.UnifiedChatResponse, error) {
	// Extract user ID from context or connection ID
	userID := a.extractUserID(ctx, req.Preferences.ConnectionID)
	
	// Convert to gateway request
	gatewayReq := a.convertToGatewayRequest(req, userID)
	
	// Send through gateway
	resp, err := a.gateway.Complete(ctx, gatewayReq)
	if err != nil {
		return nil, fmt.Errorf("gateway error: %w", err)
	}
	
	// Convert response
	unifiedResp := a.convertFromGatewayResponse(resp)
	
	// Save messages if session exists
	if req.SessionID != "" {
		a.saveMessages(ctx, req, unifiedResp)
	}
	
	return unifiedResp, nil
}

// StreamChat handles streaming chat through the gateway
func (a *UnifiedChatAdapter) StreamChat(ctx context.Context, req models.UnifiedChatRequest) (<-chan models.UnifiedStreamChunk, error) {
	fmt.Printf("[UnifiedChatAdapter.StreamChat] Starting with ConnectionID: %s\n", req.Preferences.ConnectionID)
	
	// Extract user ID
	userID := a.extractUserID(ctx, req.Preferences.ConnectionID)
	
	// Convert to gateway request
	gatewayReq := a.convertToGatewayRequest(req, userID)
	gatewayReq.Stream = true
	
	// Get stream from gateway
	gatewayStream, err := a.gateway.StreamComplete(ctx, gatewayReq)
	if err != nil {
		return nil, fmt.Errorf("gateway stream error: %w", err)
	}
	
	// Create output channel
	out := make(chan models.UnifiedStreamChunk)
	
	// Process stream
	go func() {
		defer close(out)
		var fullContent string
		
		// Send initial metadata
		out <- models.UnifiedStreamChunk{
			Type: "meta",
			Metadata: &models.ChunkMetadata{
				Provider: gatewayReq.Preferences.Provider,
				Model:    gatewayReq.Model,
			},
		}
		
		for chunk := range gatewayStream {
			// Convert chunk
			unifiedChunk := a.convertStreamChunk(chunk)
			
			// Accumulate content
			if chunk.Type == "content" {
				fullContent += chunk.Content
			}
			
			// Send chunk
			select {
			case out <- *unifiedChunk:
			case <-ctx.Done():
				return
			}
		}
		
		// Save messages if session exists
		if req.SessionID != "" && fullContent != "" {
			a.saveStreamMessages(ctx, req, fullContent)
		}
	}()
	
	return out, nil
}

// GetAvailableModels returns available models through the gateway
func (a *UnifiedChatAdapter) GetAvailableModels(ctx context.Context) (*models.UnifiedModelsResponse, error) {
	// For now, return empty as we need user context for gateway
	// This method needs refactoring to accept userID
	return &models.UnifiedModelsResponse{
		Models: []models.ModelInfo{},
		Total:  0,
	}, nil
}

// UpdateSessionTimestamp updates the session's updated_at timestamp
func (a *UnifiedChatAdapter) UpdateSessionTimestamp(ctx context.Context, userID uuid.UUID, sessionID string) error {
	return a.sessionRepo.Update(ctx, userID, sessionID, map[string]interface{}{})
}

// MaybeAutoLabelSession checks if a session should be auto-labeled
func (a *UnifiedChatAdapter) MaybeAutoLabelSession(ctx context.Context, userID uuid.UUID, sessionID string) error {
	// Get the session
	session, err := a.sessionRepo.Get(ctx, userID, sessionID)
	if err != nil {
		return fmt.Errorf("failed to get session: %w", err)
	}
	
	// Check if title is still default
	if !isDefaultTitle(session.Title) {
		return nil
	}
	
	// Get messages
	messages, err := a.messageRepo.ListBySession(ctx, sessionID)
	if err != nil {
		return fmt.Errorf("failed to get messages: %w", err)
	}
	
	// Check message count
	if len(messages) < 5 {
		return nil
	}
	
	// Generate title through gateway
	title, err := a.GenerateTitleForSession(ctx, userID.String(), messages, session, "")
	if err != nil {
		// Fallback to simple extraction
		title = generateTitleFromMessages(messages)
	}
	
	if title == "" {
		return nil
	}
	
	// Update the session title
	return a.sessionRepo.Update(ctx, userID, sessionID, map[string]interface{}{
		"title": title,
	})
}

// GenerateTitleForSession generates a title using the gateway
func (a *UnifiedChatAdapter) GenerateTitleForSession(ctx context.Context, userID string, messages []repository.Message, session *repository.Session, connectionID string) (string, error) {
	// Build conversation context
	conversationContext := a.buildConversationContext(messages)
	
	// Create prompt
	prompt := fmt.Sprintf(`Generate a concise, descriptive title for this conversation. The title should:
- Be 3-7 words maximum
- Capture the main topic or purpose of the conversation
- Be clear and specific
- Not include phrases like "Chat about" or "Discussion of"
- Not use quotation marks

Conversation:
%s

Title:`, conversationContext)
	
	// Create gateway request
	gatewayReq := &llm.Request{
		Messages: []llm.Message{
			{
				Role:    "system",
				Content: "You are a helpful assistant that creates concise, descriptive titles for conversations. Respond with ONLY the title, nothing else.",
			},
			{
				Role:    "user",
				Content: prompt,
			},
		},
		Temperature: floatPtr(0.3),
		MaxTokens:   intPtr(20),
		UserID:      userID,
	}
	
	// Set connection preference if provided
	if connectionID != "" {
		gatewayReq.Preferences.ConnectionID = connectionID
	} else if session != nil && session.Provider.Valid {
		gatewayReq.Preferences.Provider = session.Provider.String
		if session.Model.Valid {
			gatewayReq.Preferences.Model = session.Model.String
		}
	}
	
	// Send through gateway
	resp, err := a.gateway.Complete(ctx, gatewayReq)
	if err != nil {
		return "", fmt.Errorf("gateway error: %w", err)
	}
	
	// Clean up title
	title := strings.TrimSpace(resp.Content)
	title = strings.Trim(title, "\"'")
	
	// Validate
	if len(title) > 100 {
		title = title[:97] + "..."
	}
	
	if len(title) < 3 {
		return "", fmt.Errorf("generated title too short")
	}
	
	return title, nil
}

// Helper methods

func (a *UnifiedChatAdapter) extractUserID(ctx context.Context, connectionID string) string {
	// Try to extract from connection ID format: userID:connectionID
	if connectionID != "" {
		parts := strings.Split(connectionID, ":")
		if len(parts) == 2 {
			return parts[0]
		}
	}
	
	// TODO: Extract from context when available
	return ""
}

func (a *UnifiedChatAdapter) convertToGatewayRequest(req models.UnifiedChatRequest, userID string) *llm.Request {
	// Convert messages
	var messages []llm.Message
	for _, msg := range req.Messages {
		messages = append(messages, llm.Message{
			Role:    msg.Role,
			Content: msg.Content,
		})
	}
	
	// Convert tools
	var tools []llm.Tool
	for _, fn := range req.Functions {
		tools = append(tools, llm.Tool{
			Type: "function",
			Function: llm.ToolFunction{
				Name:        fn.Name,
				Description: fn.Description,
				Parameters:  fn.Parameters,
			},
		})
	}
	
	// Extract clean connection ID if it includes userID prefix
	connectionID := req.Preferences.ConnectionID
	if strings.Contains(connectionID, ":") {
		parts := strings.Split(connectionID, ":")
		if len(parts) == 2 {
			connectionID = parts[1]
		}
	}
	
	return &llm.Request{
		Messages:    messages,
		Tools:       tools,
		Temperature: req.Temperature,
		MaxTokens:   req.MaxTokens,
		Model:       req.Preferences.Model,
		Preferences: llm.Preferences{
			Provider:     req.Preferences.Provider,
			Model:        req.Preferences.Model,
			ConnectionID: connectionID,
		},
		Requirements: llm.Requirements{
			RequireTools: len(tools) > 0,
		},
		UserID: userID,
	}
}

func (a *UnifiedChatAdapter) convertFromGatewayResponse(resp *llm.Response) *models.UnifiedChatResponse {
	return &models.UnifiedChatResponse{
		ID:      resp.ID,
		Role:    resp.Role,
		Content: resp.Content,
		Usage: models.Usage{
			PromptTokens:     resp.Usage.PromptTokens,
			CompletionTokens: resp.Usage.CompletionTokens,
			TotalTokens:      resp.Usage.TotalTokens,
		},
		Metadata: models.ResponseMetadata{
			Provider:      resp.Provider,
			Model:         resp.Model,
			LatencyMs:     resp.Metadata.LatencyMs,
			RoutingReason: fmt.Sprintf("Gateway routed to %s", resp.Provider),
		},
	}
}

func (a *UnifiedChatAdapter) convertStreamChunk(chunk *llm.StreamChunk) *models.UnifiedStreamChunk {
	return &models.UnifiedStreamChunk{
		Type:    chunk.Type,
		Content: chunk.Content,
		Error:   a.convertError(chunk.Error),
		Metadata: &models.ChunkMetadata{
			Provider: chunk.Provider,
			Model:    chunk.Model,
		},
	}
}

func (a *UnifiedChatAdapter) convertError(err error) *models.UnifiedError {
	if err == nil {
		return nil
	}
	return &models.UnifiedError{
		Code:    "gateway_error",
		Message: err.Error(),
	}
}

func (a *UnifiedChatAdapter) saveMessages(ctx context.Context, req models.UnifiedChatRequest, resp *models.UnifiedChatResponse) {
	// Save user message
	if len(req.Messages) > 0 {
		lastMsg := req.Messages[len(req.Messages)-1]
		if lastMsg.Role == "user" {
			a.messageRepo.Create(ctx, repository.Message{
				SessionID: req.SessionID,
				Role:      lastMsg.Role,
				Content:   lastMsg.Content,
			})
		}
	}
	
	// Save assistant response
	a.messageRepo.Create(ctx, repository.Message{
		SessionID: req.SessionID,
		Role:      resp.Role,
		Content:   resp.Content,
	})
}

func (a *UnifiedChatAdapter) saveStreamMessages(ctx context.Context, req models.UnifiedChatRequest, fullContent string) {
	// Save user message
	if len(req.Messages) > 0 {
		lastMsg := req.Messages[len(req.Messages)-1]
		if lastMsg.Role == "user" {
			a.messageRepo.Create(ctx, repository.Message{
				SessionID: req.SessionID,
				Role:      lastMsg.Role,
				Content:   lastMsg.Content,
			})
		}
	}
	
	// Save assistant response
	if fullContent != "" {
		a.messageRepo.Create(ctx, repository.Message{
			SessionID: req.SessionID,
			Role:      "assistant",
			Content:   fullContent,
		})
	}
}

func (a *UnifiedChatAdapter) buildConversationContext(messages []repository.Message) string {
	var conversationParts []string
	messageCount := 0
	
	for _, msg := range messages {
		if msg.Role == "user" || msg.Role == "assistant" {
			if messageCount >= 10 {
				break
			}
			roleLabel := "User"
			if msg.Role == "assistant" {
				roleLabel = "Assistant"
			}
			content := msg.Content
			if len(content) > 500 {
				content = content[:500] + "..."
			}
			conversationParts = append(conversationParts, fmt.Sprintf("%s: %s", roleLabel, content))
			messageCount++
		}
	}
	
	return strings.Join(conversationParts, "\n\n")
}