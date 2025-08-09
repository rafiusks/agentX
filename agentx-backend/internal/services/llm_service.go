package services

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/agentx/agentx-backend/internal/api/models"
	"github.com/agentx/agentx-backend/internal/llm"
	"github.com/agentx/agentx-backend/internal/providers"
	"github.com/agentx/agentx-backend/internal/repository"
)

// LLMService wraps the LLM Gateway for use by other services
type LLMService struct {
	gateway       *llm.Gateway
	sessionRepo   repository.SessionRepository
	messageRepo   repository.MessageRepository
	contextMemory *ContextMemoryService
}

// NewLLMService creates a new LLM service
func NewLLMService(
	gateway *llm.Gateway,
	sessionRepo repository.SessionRepository,
	messageRepo repository.MessageRepository,
	contextMemory *ContextMemoryService,
) *LLMService {
	return &LLMService{
		gateway:       gateway,
		sessionRepo:   sessionRepo,
		messageRepo:   messageRepo,
		contextMemory: contextMemory,
	}
}

// Chat handles a unified chat request through the gateway
func (s *LLMService) Chat(ctx context.Context, req models.UnifiedChatRequest) (*models.UnifiedChatResponse, error) {
	// Convert to gateway request
	gatewayReq := s.convertToGatewayRequest(req)
	
	// Send through gateway
	resp, err := s.gateway.Complete(ctx, gatewayReq)
	if err != nil {
		return nil, fmt.Errorf("gateway error: %w", err)
	}
	
	// Convert response
	unifiedResp := s.convertFromGatewayResponse(resp)
	
	// Save messages if session exists
	if req.SessionID != "" {
		// Save user message
		if len(req.Messages) > 0 {
			lastMsg := req.Messages[len(req.Messages)-1]
			if lastMsg.Role == "user" {
				s.messageRepo.Create(ctx, repository.Message{
					SessionID: req.SessionID,
					Role:      lastMsg.Role,
					Content:   lastMsg.Content,
				})
			}
		}
		
		// Save assistant response
		s.messageRepo.Create(ctx, repository.Message{
			SessionID: req.SessionID,
			Role:      unifiedResp.Role,
			Content:   unifiedResp.Content,
		})
	}
	
	return unifiedResp, nil
}

// StreamChat handles streaming chat through the gateway
func (s *LLMService) StreamChat(ctx context.Context, req models.UnifiedChatRequest) (<-chan models.UnifiedStreamChunk, error) {
	// Convert to gateway request
	gatewayReq := s.convertToGatewayRequest(req)
	gatewayReq.Stream = true
	
	// Get stream from gateway
	gatewayStream, err := s.gateway.StreamComplete(ctx, gatewayReq)
	if err != nil {
		return nil, fmt.Errorf("gateway stream error: %w", err)
	}
	
	// Create output channel
	out := make(chan models.UnifiedStreamChunk)
	
	// Process stream
	go func() {
		defer close(out)
		var fullContent string
		
		for chunk := range gatewayStream {
			// Convert chunk
			unifiedChunk := s.convertStreamChunk(chunk)
			
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
		if req.SessionID != "" {
			// Save user message
			if len(req.Messages) > 0 {
				lastMsg := req.Messages[len(req.Messages)-1]
				if lastMsg.Role == "user" {
					s.messageRepo.Create(ctx, repository.Message{
						SessionID: req.SessionID,
						Role:      lastMsg.Role,
						Content:   lastMsg.Content,
					})
				}
			}
			
			// Save assistant response
			if fullContent != "" {
				s.messageRepo.Create(ctx, repository.Message{
					SessionID: req.SessionID,
					Role:      "assistant",
					Content:   fullContent,
				})
			}
		}
	}()
	
	return out, nil
}

// GenerateTitleForSession generates a title using the gateway
func (s *LLMService) GenerateTitleForSession(ctx context.Context, userID string, messages []repository.Message, session *repository.Session, connectionID string) (string, error) {
	fmt.Printf("[LLMService.GenerateTitleForSession] Starting with userID: %s, connectionID: %s\n", userID, connectionID)
	
	// Build conversation context
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
	
	conversationContext := strings.Join(conversationParts, "\n\n")
	
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
		Preferences: llm.Preferences{
			ConnectionID: connectionID,
		},
		UserID: userID,
	}
	
	// Send through gateway
	resp, err := s.gateway.Complete(ctx, gatewayReq)
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

// GetAvailableModels returns available models through the gateway
func (s *LLMService) GetAvailableModels(ctx context.Context, userID string) (*models.UnifiedModelsResponse, error) {
	modelsFromGateway, err := s.gateway.GetAvailableModels(ctx, userID)
	if err != nil {
		return nil, err
	}
	
	// Convert to unified response
	var unifiedModels []models.ModelInfo
	for _, model := range modelsFromGateway {
		// Convert string capabilities to Capabilities struct
		caps := s.convertCapabilities(model.Capabilities)
		
		unifiedModels = append(unifiedModels, models.ModelInfo{
			ID:           model.ID,
			Provider:     model.Provider,
			DisplayName:  model.DisplayName,
			Description:  model.Description,
			Capabilities: caps,
			Status: models.ModelStatus{
				Available: true,
				Health:    "healthy",
			},
		})
	}
	
	return &models.UnifiedModelsResponse{
		Models: unifiedModels,
		Total:  len(unifiedModels),
	}, nil
}

// InitializeUserConnections initializes providers for a user's connections
func (s *LLMService) InitializeUserConnections(ctx context.Context, userID uuid.UUID, connectionService *ConnectionService) error {
	fmt.Printf("[LLMService.InitializeUserConnections] Initializing for user: %s\n", userID.String())
	
	// Get user's connections
	connections, err := connectionService.ListConnections(ctx, userID, "")
	if err != nil {
		return fmt.Errorf("failed to list connections: %w", err)
	}
	
	fmt.Printf("[LLMService.InitializeUserConnections] Found %d connections\n", len(connections))
	
	// Register each enabled connection with the gateway
	for _, conn := range connections {
		if conn.Enabled {
			config := llm.ProviderConfig{
				Type:         conn.ProviderID,
				Name:         conn.Name,
				APIKey:       getStringFromConfig(conn.Config, "api_key"),
				BaseURL:      getStringFromConfig(conn.Config, "base_url"),
				Organization: getStringFromConfig(conn.Config, "organization"),
			}
			
			fmt.Printf("[LLMService.InitializeUserConnections] Registering connection %s (%s)\n", conn.Name, conn.ID.String())
			
			if err := s.gateway.RegisterProvider(userID.String(), conn.ID.String(), config); err != nil {
				fmt.Printf("[LLMService.InitializeUserConnections] Failed to register %s: %v\n", conn.Name, err)
			} else {
				fmt.Printf("[LLMService.InitializeUserConnections] Successfully registered %s\n", conn.Name)
			}
		}
	}
	
	return nil
}

// EnsureConnectionInitialized ensures a connection is registered with the gateway
func (s *LLMService) EnsureConnectionInitialized(ctx context.Context, userID uuid.UUID, connectionID string, connectionService *ConnectionService) error {
	// Try to get the connection
	conn, err := connectionService.GetConnection(ctx, userID, connectionID)
	if err != nil {
		return fmt.Errorf("connection not found: %w", err)
	}
	
	if !conn.Enabled {
		return fmt.Errorf("connection is disabled")
	}
	
	// Register with gateway
	config := llm.ProviderConfig{
		Type:         conn.ProviderID,
		Name:         conn.Name,
		APIKey:       getStringFromConfig(conn.Config, "api_key"),
		BaseURL:      getStringFromConfig(conn.Config, "base_url"),
		Organization: getStringFromConfig(conn.Config, "organization"),
	}
	
	return s.gateway.RegisterProvider(userID.String(), connectionID, config)
}

// Helper functions

func (s *LLMService) convertToGatewayRequest(req models.UnifiedChatRequest) *llm.Request {
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
	
	return &llm.Request{
		Messages:    messages,
		Tools:       tools,
		Temperature: req.Temperature,
		MaxTokens:   req.MaxTokens,
		Preferences: llm.Preferences{
			Provider:     req.Preferences.Provider,
			Model:        req.Preferences.Model,
			ConnectionID: req.Preferences.ConnectionID,
		},
		Requirements: llm.Requirements{
			RequireTools: len(tools) > 0,
		},
	}
}

func (s *LLMService) convertFromGatewayResponse(resp *llm.Response) *models.UnifiedChatResponse {
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

func (s *LLMService) convertStreamChunk(chunk *llm.StreamChunk) *models.UnifiedStreamChunk {
	// Note: chunk.Usage would need to be added to StreamChunk type
	// For now, we'll skip usage in streaming chunks
	return &models.UnifiedStreamChunk{
		Type:    chunk.Type,
		Content: chunk.Content,
		Error:   s.convertError(chunk.Error),
		Metadata: &models.ChunkMetadata{
			Provider: chunk.Provider,
			Model:    chunk.Model,
		},
	}
}

func (s *LLMService) convertError(err error) *models.UnifiedError {
	if err == nil {
		return nil
	}
	return &models.UnifiedError{
		Code:    "gateway_error",
		Message: err.Error(),
	}
}

func (s *LLMService) convertUsage(usage *llm.Usage) *models.Usage {
	if usage == nil {
		return nil
	}
	return &models.Usage{
		PromptTokens:     usage.PromptTokens,
		CompletionTokens: usage.CompletionTokens,
		TotalTokens:      usage.TotalTokens,
	}
}

func getStringFromConfig(config map[string]interface{}, key string) string {
	if v, ok := config[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// convertCapabilities converts string capabilities to Capabilities struct
func (s *LLMService) convertCapabilities(caps []string) providers.Capabilities {
	result := providers.Capabilities{}
	
	for _, cap := range caps {
		switch cap {
		case "chat":
			result.Chat = true
		case "streaming":
			result.Streaming = true
		case "function_calling":
			result.FunctionCalling = true
		case "vision":
			result.Vision = true
		case "embeddings":
			result.Embeddings = true
		case "audio_input":
			result.AudioInput = true
		case "audio_output":
			result.AudioOutput = true
		}
	}
	
	return result
}

// Helper functions for creating pointers are defined in unified_chat.go