package services

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/agentx/agentx-backend/internal/api/models"
	"github.com/agentx/agentx-backend/internal/llm"
	"github.com/agentx/agentx-backend/internal/providers"
	"github.com/agentx/agentx-backend/internal/repository"
)

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// OrchestrationService is the central coordinator for all system operations
// It acts as the single entry point for all clients (API, CLI, workers, etc.)
// and coordinates between different services to fulfill requests
type OrchestrationService struct {
	// Core services
	gateway    *llm.Gateway      // LLM Gateway for all AI operations  
	llmService *llm.Service      // LLM Service for structured AI tasks
	cache      *CacheService     // Caching layer for performance
	db         *DatabaseService  // Unified database access
	
	// Repository access (will be moved to DatabaseService)
	sessionRepo repository.SessionRepository
	messageRepo repository.MessageRepository
	
	// Supporting services
	contextMemory *ContextMemoryService
	connections   *ConnectionService
	config        *ConfigService
	mcpTools      *MCPToolIntegration // MCP tool integration
}

// NewOrchestrationService creates a new orchestration service
func NewOrchestrationService(
	gateway *llm.Gateway,
	sessionRepo repository.SessionRepository,
	messageRepo repository.MessageRepository,
	contextMemory *ContextMemoryService,
	connections *ConnectionService,
	config *ConfigService,
	llmService *llm.Service,
	mcpTools *MCPToolIntegration,
) *OrchestrationService {
	return &OrchestrationService{
		gateway:       gateway,
		cache:         NewCacheService(), // Will be enhanced with Redis later
		db:            nil,                // Will be created in phase 2
		sessionRepo:   sessionRepo,
		messageRepo:   messageRepo,
		contextMemory: contextMemory,
		connections:   connections,
		config:        config,
		llmService:    llmService,
		mcpTools:      mcpTools,
	}
}

// =====================================
// Core Chat Operations
// =====================================

// ChatWithUser handles a chat request through the orchestrator with explicit userID
func (o *OrchestrationService) ChatWithUser(ctx context.Context, userID uuid.UUID, req models.UnifiedChatRequest) (*models.UnifiedChatResponse, error) {
	// Initialize user connections if needed
	if userID != uuid.Nil {
		if err := o.InitializeUserConnections(ctx, userID); err != nil {
			fmt.Printf("[OrchestrationService.ChatWithUser] Warning: Failed to initialize connections: %v\n", err)
		}
	}
	
	// Check cache first
	cacheKey := o.generateCacheKey("chat", userID.String(), req)
	if cached := o.cache.Get(cacheKey); cached != nil {
		if resp, ok := cached.(*models.UnifiedChatResponse); ok {
			return resp, nil
		}
	}
	
	// Check if the user's message contains a tool invocation or force web search is enabled
	if len(req.Messages) > 0 && o.mcpTools != nil {
		lastMessage := req.Messages[len(req.Messages)-1]
		if lastMessage.Role == "user" {
			var invocation *ToolInvocation
			var err error
			
			// Force web search if flag is set
			if req.ForceWebSearch {
				// Create a web search invocation for the user's message
				// Enable includeContent to fetch actual page content, not just snippets
				args, _ := json.Marshal(map[string]interface{}{
					"query":         lastMessage.Content,
					"maxResults":    3, // Reduced to 3 to avoid too much content
					"includeContent": true, // Fetch actual page content
				})
				invocation = &ToolInvocation{
					Type:      "builtin",
					ServerID:  "builtin-websearch",
					ToolName:  "web_search",
					Arguments: args,
				}
			} else {
				// Detect tool invocation normally
				invocation, err = o.mcpTools.DetectToolInvocation(lastMessage.Content)
			}
			
			if err == nil && invocation != nil {
				// Invoke the tool
				toolResult, err := o.mcpTools.InvokeToolForUser(ctx, userID, invocation)
				if err == nil && toolResult != nil {
					// Format the result for chat
					formattedResult := o.mcpTools.FormatToolResultForChat(toolResult, invocation.ToolName)
					
					// For web search, prepend results to the last user message instead of adding as separate message
					if invocation.ToolName == "web_search" && len(req.Messages) > 0 {
						// Find the last user message and prepend search context
						for i := len(req.Messages) - 1; i >= 0; i-- {
							if req.Messages[i].Role == "user" {
								// Prepend search results to user's question
								req.Messages[i].Content = formattedResult + "\n\n" + req.Messages[i].Content
								break
							}
						}
					} else {
						// For other tools, add as assistant message
						req.Messages = append(req.Messages, providers.Message{
							Role:    "assistant",
							Content: formattedResult,
						})
					}
					}
				}
			}
		}
	}
	
	// Enrich with context if needed
	if req.SessionID != "" && o.contextMemory != nil {
		// Get relevant context
		messages, err := o.messageRepo.ListBySession(ctx, req.SessionID)
		if err == nil && len(messages) > 0 {
			// Add context to request
			req.Messages = o.enrichWithContext(req.Messages, messages)
		}
	}
	
	// Convert to gateway request
	gatewayReq := o.convertToGatewayRequest(req, userID.String())
	
	// Send through gateway
	resp, err := o.gateway.Complete(ctx, gatewayReq)
	if err != nil {
		return nil, fmt.Errorf("gateway error: %w", err)
	}
	
	// Convert response
	unifiedResp := o.convertFromGatewayResponse(resp)
	
	// Cache successful response
	o.cache.Set(cacheKey, unifiedResp, 5*time.Minute)
	
	// Save messages if session exists
	if req.SessionID != "" {
		o.saveMessages(ctx, req, unifiedResp)
	}
	
	return unifiedResp, nil
}

// StreamChatWithUser handles streaming chat through the orchestrator with explicit userID
func (o *OrchestrationService) StreamChatWithUser(ctx context.Context, userID uuid.UUID, req models.UnifiedChatRequest) (<-chan models.UnifiedStreamChunk, error) {
	// Initialize user connections if needed
	if userID != uuid.Nil {
		if err := o.InitializeUserConnections(ctx, userID); err != nil {
			fmt.Printf("[OrchestrationService.StreamChatWithUser] Warning: Failed to initialize connections: %v\n", err)
		}
	}
	
	// Check if the user's message contains a tool invocation or force web search is enabled
	if len(req.Messages) > 0 && o.mcpTools != nil {
		lastMessage := req.Messages[len(req.Messages)-1]
		if lastMessage.Role == "user" {
			var invocation *ToolInvocation
			var err error
			
			// Force web search if flag is set
			if req.ForceWebSearch {
				// Create a web search invocation for the user's message
				// Enable includeContent to fetch actual page content, not just snippets
				args, _ := json.Marshal(map[string]interface{}{
					"query":         lastMessage.Content,
					"maxResults":    3, // Reduced to 3 to avoid too much content
					"includeContent": true, // Fetch actual page content
				})
				invocation = &ToolInvocation{
					Type:      "builtin",
					ServerID:  "builtin-websearch",
					ToolName:  "web_search",
					Arguments: args,
				}
			} else {
				// Detect tool invocation normally
				invocation, err = o.mcpTools.DetectToolInvocation(lastMessage.Content)
			}
			
			if err == nil && invocation != nil {
				// Invoke the tool
				toolResult, err := o.mcpTools.InvokeToolForUser(ctx, userID, invocation)
				if err == nil && toolResult != nil {
					// Format the result for chat
					formattedResult := o.mcpTools.FormatToolResultForChat(toolResult, invocation.ToolName)
					
					// For web search, prepend results to the last user message instead of adding as separate message
					if invocation.ToolName == "web_search" && len(req.Messages) > 0 {
						// Find the last user message and prepend search context
						for i := len(req.Messages) - 1; i >= 0; i-- {
							if req.Messages[i].Role == "user" {
								// Prepend search results to user's question
								req.Messages[i].Content = formattedResult + "\n\n" + req.Messages[i].Content
								break
							}
						}
					} else {
						// For other tools, add as assistant message
						req.Messages = append(req.Messages, providers.Message{
							Role:    "assistant",
							Content: formattedResult,
						})
					}
					}
				}
			}
		}
	}
	
	// Enrich with context if needed
	if req.SessionID != "" && o.contextMemory != nil {
		messages, err := o.messageRepo.ListBySession(ctx, req.SessionID)
		if err == nil && len(messages) > 0 {
			req.Messages = o.enrichWithContext(req.Messages, messages)
		}
	}
	
	// Convert to gateway request
	gatewayReq := o.convertToGatewayRequest(req, userID.String())
	gatewayReq.Stream = true
	
	// Get stream from gateway
	gatewayStream, err := o.gateway.StreamComplete(ctx, gatewayReq)
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
			unifiedChunk := o.convertStreamChunk(chunk)
			
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
			o.saveStreamedMessages(ctx, req, fullContent)
		}
	}()
	
	return out, nil
}

// =====================================
// Session Management
// =====================================

// CreateSession creates a new chat session
func (o *OrchestrationService) CreateSession(ctx context.Context, userID uuid.UUID, title string) (*repository.Session, error) {
	if title == "" {
		title = "New Chat"
	}
	
	session := &repository.Session{
		ID:        uuid.New().String(),
		UserID:    userID,
		Title:     title,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	
	if err := o.sessionRepo.Create(ctx, session); err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}
	
	// Invalidate user's session list cache
	o.cache.Delete(fmt.Sprintf("sessions:%s", userID.String()))
	
	return session, nil
}

// GetSession retrieves a specific session
func (o *OrchestrationService) GetSession(ctx context.Context, userID uuid.UUID, sessionID string) (*repository.Session, error) {
	// Check cache
	cacheKey := fmt.Sprintf("session:%s:%s", userID.String(), sessionID)
	if cached := o.cache.Get(cacheKey); cached != nil {
		if session, ok := cached.(*repository.Session); ok {
			return session, nil
		}
	}
	
	session, err := o.sessionRepo.Get(ctx, userID, sessionID)
	if err != nil {
		return nil, err
	}
	
	// Cache for future requests
	o.cache.Set(cacheKey, session, 10*time.Minute)
	
	return session, nil
}

// ListSessions returns all sessions for a user
func (o *OrchestrationService) ListSessions(ctx context.Context, userID uuid.UUID) ([]*repository.Session, error) {
	// Check cache
	cacheKey := fmt.Sprintf("sessions:%s", userID.String())
	if cached := o.cache.Get(cacheKey); cached != nil {
		if sessions, ok := cached.([]*repository.Session); ok {
			return sessions, nil
		}
	}
	
	sessions, err := o.sessionRepo.List(ctx, userID)
	if err != nil {
		return nil, err
	}
	
	// Cache for future requests
	o.cache.Set(cacheKey, sessions, 5*time.Minute)
	
	return sessions, nil
}

// DeleteSession removes a session and its messages
func (o *OrchestrationService) DeleteSession(ctx context.Context, userID uuid.UUID, sessionID string) error {
	// Delete from database
	if err := o.sessionRepo.Delete(ctx, userID, sessionID); err != nil {
		return err
	}
	
	// Invalidate caches
	o.cache.Delete(fmt.Sprintf("session:%s:%s", userID.String(), sessionID))
	o.cache.Delete(fmt.Sprintf("sessions:%s", userID.String()))
	o.cache.Delete(fmt.Sprintf("messages:%s", sessionID))
	
	return nil
}

// UpdateSession updates session details
func (o *OrchestrationService) UpdateSession(ctx context.Context, userID uuid.UUID, sessionID string, updates map[string]interface{}) error {
	if err := o.sessionRepo.Update(ctx, userID, sessionID, updates); err != nil {
		return err
	}
	
	// Invalidate caches
	o.cache.Delete(fmt.Sprintf("session:%s:%s", userID.String(), sessionID))
	o.cache.Delete(fmt.Sprintf("sessions:%s", userID.String()))
	
	return nil
}

// =====================================
// Message Management
// =====================================

// GetMessages retrieves messages for a session
func (o *OrchestrationService) GetMessages(ctx context.Context, sessionID string) ([]repository.Message, error) {
	// Check cache
	cacheKey := fmt.Sprintf("messages:%s", sessionID)
	if cached := o.cache.Get(cacheKey); cached != nil {
		if messages, ok := cached.([]repository.Message); ok {
			return messages, nil
		}
	}
	
	messages, err := o.messageRepo.ListBySession(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	
	// Cache for future requests
	o.cache.Set(cacheKey, messages, 5*time.Minute)
	
	return messages, nil
}

// SaveMessage saves a single message
func (o *OrchestrationService) SaveMessage(ctx context.Context, message *repository.Message) error {
	message.ID = uuid.New().String()
	message.CreatedAt = time.Now()
	
	if _, err := o.messageRepo.Create(ctx, *message); err != nil {
		return err
	}
	
	// Invalidate message cache for this session
	o.cache.Delete(fmt.Sprintf("messages:%s", message.SessionID))
	
	// Update session timestamp
	o.sessionRepo.Update(ctx, uuid.Nil, message.SessionID, map[string]interface{}{
		"updated_at": time.Now(),
	})
	
	return nil
}

// =====================================
// Specialized Operations
// =====================================

// GenerateTitle generates a title for a session
func (o *OrchestrationService) GenerateTitle(ctx context.Context, userID uuid.UUID, sessionID string, connectionID string) (string, error) {
	fmt.Printf("[GenerateTitle] Starting for session %s with connectionID: %s\n", sessionID, connectionID)
	
	// Use the new LLM service for title generation
	req := llm.CompletionRequest{
		Task: llm.TaskGenerateTitle,
		Context: map[string]interface{}{
			"session_id": sessionID,
			"user_id":    userID.String(),
		},
		ConnectionID: connectionID,
		Parameters: llm.Parameters{
			MaxTokens:   &[]int{50}[0], // Limit to 50 tokens for titles
			Temperature: &[]float32{0.7}[0],
		},
	}
	
	resp, err := o.llmService.Complete(ctx, userID.String(), req)
	if err != nil {
		fmt.Printf("[GenerateTitle] LLM service failed: %v\n", err)
		return "", fmt.Errorf("failed to generate title: %w", err)
	}
	
	title := strings.TrimSpace(resp.Result)
	if title == "" {
		return "", fmt.Errorf("empty title generated")
	}
	
	fmt.Printf("[GenerateTitle] Generated title: %s\n", title)
	
	// Cache the result
	cacheKey := fmt.Sprintf("title:%s", sessionID)
	o.cache.Set(cacheKey, title, 24*time.Hour) // Cache for 24 hours
	
	return title, nil
}

// =====================================
// UnifiedChatInterface Implementation
// =====================================

// ChatUnified handles unified chat requests without userID (implements UnifiedChatInterface)
func (o *OrchestrationService) Chat(ctx context.Context, req models.UnifiedChatRequest) (*models.UnifiedChatResponse, error) {
	// Extract user ID from context or request
	userID := uuid.Nil
	if req.Preferences.ConnectionID != "" {
		// Try to extract userID from connection ID format "userID:connectionID"
		parts := strings.SplitN(req.Preferences.ConnectionID, ":", 2)
		if len(parts) == 2 {
			if uid, err := uuid.Parse(parts[0]); err == nil {
				userID = uid
			}
		} else if len(parts) == 1 {
			// Single value - assume it's just the user ID
			if uid, err := uuid.Parse(parts[0]); err == nil {
				userID = uid
			}
		}
	}
	
	fmt.Printf("[OrchestrationService.Chat] Extracted UserID: %s from ConnectionID: %s\n", userID, req.Preferences.ConnectionID)
	
	// Pass through to the main ChatWithUser method
	return o.ChatWithUser(ctx, userID, req)
}

// StreamChatUnified handles streaming chat requests without userID (implements UnifiedChatInterface)
func (o *OrchestrationService) StreamChat(ctx context.Context, req models.UnifiedChatRequest) (<-chan models.UnifiedStreamChunk, error) {
	// Extract user ID from context or request
	userID := uuid.Nil
	if req.Preferences.ConnectionID != "" {
		// Try to extract userID from connection ID format "userID:connectionID"
		parts := strings.SplitN(req.Preferences.ConnectionID, ":", 2)
		if len(parts) == 2 {
			if uid, err := uuid.Parse(parts[0]); err == nil {
				userID = uid
			}
		}
	}
	
	// Pass through to the main StreamChatWithUser method
	return o.StreamChatWithUser(ctx, userID, req)
}

// GetAvailableModels returns available models (implements UnifiedChatInterface)
func (o *OrchestrationService) GetAvailableModels(ctx context.Context) (*models.UnifiedModelsResponse, error) {
	// For now, return empty response - this will be enhanced later
	// TODO: Implement model discovery through gateway
	response := &models.UnifiedModelsResponse{
		Models: make([]models.ModelInfo, 0),
		Total:  0,
	}
	
	return response, nil
}

// UpdateSessionTimestamp updates the last activity timestamp for a session (implements UnifiedChatInterface)
func (o *OrchestrationService) UpdateSessionTimestamp(ctx context.Context, userID uuid.UUID, sessionID string) error {
	updates := map[string]interface{}{
		"updated_at": time.Now(),
	}
	return o.sessionRepo.Update(ctx, userID, sessionID, updates)
}

// MaybeAutoLabelSession checks if a session needs auto-labeling and does it (implements UnifiedChatInterface)
func (o *OrchestrationService) MaybeAutoLabelSession(ctx context.Context, userID uuid.UUID, sessionID string) error {
	// Get session
	session, err := o.sessionRepo.Get(ctx, userID, sessionID)
	if err != nil {
		return err
	}
	
	// Skip if already has a custom title
	if session.Title != "New Chat" && session.Title != "" {
		return nil
	}
	
	// Get messages
	messages, err := o.messageRepo.ListBySession(ctx, sessionID)
	if err != nil {
		return err
	}
	
	// Need at least one user message to generate title
	if len(messages) == 0 {
		return nil
	}
	
	// Generate title
	title, err := o.GenerateTitle(ctx, userID, sessionID, "")
	if err != nil {
		return err
	}
	
	// Update session with generated title
	updates := map[string]interface{}{
		"title": title,
	}
	return o.sessionRepo.Update(ctx, userID, sessionID, updates)
}

// GenerateTitleForSession generates a title for a session (implements UnifiedChatInterface)
func (o *OrchestrationService) GenerateTitleForSession(ctx context.Context, userID string, messages []repository.Message, session *repository.Session, connectionID string) (string, error) {
	// Convert string userID to UUID
	uid, err := uuid.Parse(userID)
	if err != nil {
		return "", fmt.Errorf("invalid user ID: %w", err)
	}
	
	// Use the existing GenerateTitle method
	return o.GenerateTitle(ctx, uid, session.ID, connectionID)
}

// =====================================
// Connection Management
// =====================================

// InitializeUserConnections initializes LLM connections for a user
func (o *OrchestrationService) InitializeUserConnections(ctx context.Context, userID uuid.UUID) error {
	// Get user's connections
	connections, err := o.connections.ListConnections(ctx, userID, "")
	if err != nil {
		return fmt.Errorf("failed to list connections: %w", err)
	}
	
	// Register each enabled connection with the gateway
	for _, conn := range connections {
		if conn.Enabled {
			config := llm.ProviderConfig{
				Type:         conn.ProviderID,
				Name:         conn.Name,
				APIKey:       getStringFromConfigOrc(conn.Config, "api_key"),
				BaseURL:      getStringFromConfigOrc(conn.Config, "base_url"),
				Organization: getStringFromConfigOrc(conn.Config, "organization"),
			}
			
			if err := o.gateway.RegisterProvider(userID.String(), conn.ID.String(), config); err != nil {
				fmt.Printf("[Orchestrator] Failed to register connection %s: %v\n", conn.Name, err)
			}
		}
	}
	
	return nil
}

// =====================================
// Helper Methods
// =====================================

func (o *OrchestrationService) convertToGatewayRequest(req models.UnifiedChatRequest, userID string) *llm.Request {
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
		UserID:      userID,
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

func (o *OrchestrationService) convertFromGatewayResponse(resp *llm.Response) *models.UnifiedChatResponse {
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
			Provider: resp.Provider,
			Model:    resp.Model,
		},
	}
}

func (o *OrchestrationService) convertStreamChunk(chunk *llm.StreamChunk) *models.UnifiedStreamChunk {
	return &models.UnifiedStreamChunk{
		Type:    chunk.Type,
		Content: chunk.Content,
		Metadata: &models.ChunkMetadata{
			Provider: chunk.Provider,
			Model:    chunk.Model,
		},
	}
}

func (o *OrchestrationService) enrichWithContext(messages []providers.Message, contextMessages []repository.Message) []providers.Message {
	// Add relevant context from previous messages
	// This is a simplified version - can be enhanced with more sophisticated context selection
	maxContext := 5
	contextCount := 0
	
	var enriched []providers.Message
	for i := len(contextMessages) - 1; i >= 0 && contextCount < maxContext; i-- {
		msg := contextMessages[i]
		if msg.Role == "user" || msg.Role == "assistant" {
			enriched = append([]providers.Message{{
				Role:    msg.Role,
				Content: msg.Content,
			}}, enriched...)
			contextCount++
		}
	}
	
	// Append the new messages
	enriched = append(enriched, messages...)
	
	return enriched
}

func (o *OrchestrationService) generateCacheKey(prefix string, userID string, req interface{}) string {
	// Generate a cache key based on request parameters
	// This is simplified - in production would use a hash of the request
	return fmt.Sprintf("%s:%s:%v", prefix, userID, time.Now().Unix()/60) // 1-minute cache
}

func (o *OrchestrationService) saveMessages(ctx context.Context, req models.UnifiedChatRequest, resp *models.UnifiedChatResponse) {
	// Save user message
	if len(req.Messages) > 0 {
		lastMsg := req.Messages[len(req.Messages)-1]
		if lastMsg.Role == "user" {
			o.messageRepo.Create(ctx, repository.Message{
				ID:        uuid.New().String(),
				SessionID: req.SessionID,
				Role:      lastMsg.Role,
				Content:   lastMsg.Content,
				CreatedAt: time.Now(),
			})
		}
	}
	
	// Save assistant response
	o.messageRepo.Create(ctx, repository.Message{
		ID:        uuid.New().String(),
		SessionID: req.SessionID,
		Role:      resp.Role,
		Content:   resp.Content,
		CreatedAt: time.Now(),
	})
	
	// Invalidate message cache
	o.cache.Delete(fmt.Sprintf("messages:%s", req.SessionID))
}

func (o *OrchestrationService) saveStreamedMessages(ctx context.Context, req models.UnifiedChatRequest, content string) {
	// Save user message
	if len(req.Messages) > 0 {
		lastMsg := req.Messages[len(req.Messages)-1]
		if lastMsg.Role == "user" {
			o.messageRepo.Create(ctx, repository.Message{
				ID:        uuid.New().String(),
				SessionID: req.SessionID,
				Role:      lastMsg.Role,
				Content:   lastMsg.Content,
				CreatedAt: time.Now(),
			})
		}
	}
	
	// Save assistant response
	o.messageRepo.Create(ctx, repository.Message{
		ID:        uuid.New().String(),
		SessionID: req.SessionID,
		Role:      "assistant",
		Content:   content,
		CreatedAt: time.Now(),
	})
	
	// Invalidate message cache
	o.cache.Delete(fmt.Sprintf("messages:%s", req.SessionID))
}

func getStringFromConfigOrc(config map[string]interface{}, key string) string {
	if v, ok := config[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// Helper functions for creating pointers
func floatPtrOrc(f float32) *float32 {
	return &f
}

func intPtrOrc(i int) *int {
	return &i
}