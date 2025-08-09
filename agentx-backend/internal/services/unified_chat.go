package services

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/agentx/agentx-backend/internal/adapters"
	"github.com/agentx/agentx-backend/internal/api/models"
	"github.com/agentx/agentx-backend/internal/providers"
	"github.com/agentx/agentx-backend/internal/repository"
)

// UnifiedChatService handles provider-agnostic chat requests
type UnifiedChatService struct {
	providers      *providers.Registry
	router         *RequestRouter
	adapters       *adapters.Registry
	sessionRepo    repository.SessionRepository
	messageRepo    repository.MessageRepository
	contextMemory  *ContextMemoryService
}

// NewUnifiedChatService creates a new unified chat service
func NewUnifiedChatService(
	providers *providers.Registry,
	configService *ConfigService,
	sessionRepo repository.SessionRepository,
	messageRepo repository.MessageRepository,
	contextMemory *ContextMemoryService,
) *UnifiedChatService {
	return &UnifiedChatService{
		providers:     providers,
		router:        NewRequestRouter(providers, configService),
		adapters:      adapters.NewRegistry(),
		sessionRepo:   sessionRepo,
		messageRepo:   messageRepo,
		contextMemory: contextMemory,
	}
}

// Chat handles a unified chat request
func (s *UnifiedChatService) Chat(ctx context.Context, req models.UnifiedChatRequest) (*models.UnifiedChatResponse, error) {
	// Check for nil service
	if s == nil {
		fmt.Printf("[UnifiedChatService.Chat] Service is nil\n")
		return nil, fmt.Errorf("chat service is nil")
	}
	
	startTime := time.Now()
	
	// 1. Inject relevant context memories if available
	req = s.injectContextMemories(ctx, req)
	
	// 2. Route the request to the best provider/model
	if s.router == nil {
		fmt.Printf("[UnifiedChatService.Chat] Router is nil\n")
		return nil, s.createErrorResponse(fmt.Errorf("router is nil"), "routing_failed")
	}
	
	providerID, modelID, err := s.router.RouteRequest(ctx, req)
	if err != nil {
		return nil, s.createErrorResponse(err, "routing_failed")
	}
	
	// 3. Get the provider
	if s.providers == nil {
		return nil, s.createErrorResponse(fmt.Errorf("providers registry is nil"), providerID)
	}
	
	provider := s.providers.Get(providerID)
	if provider == nil {
		return nil, s.createErrorResponse(fmt.Errorf("provider not found"), providerID)
	}
	
	// 4. Get the appropriate adapter
	if s.adapters == nil {
		return nil, s.createErrorResponse(fmt.Errorf("adapters registry is nil"), providerID)
	}
	
	adapter := s.adapters.GetOrDefault(providerID)
	
	// 5. Normalize the request for the provider
	providerReq, err := adapter.NormalizeRequest(req)
	if err != nil {
		return nil, s.createErrorResponse(err, providerID)
	}
	providerReq.Model = modelID
	
	// 6. Save user message if session exists
	if req.SessionID != "" {
		// Only save the last user message (the new one)
		// The frontend sends the full conversation history, but we only want to persist the new message
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
	}
	
	// 7. Make the request to the provider
	resp, err := provider.Complete(ctx, providerReq)
	if err != nil {
		// Try fallback if available
		if fallbackResp, fallbackErr := s.tryFallback(ctx, req, adapter.NormalizeError(err, providerID)); fallbackErr == nil {
			return fallbackResp, nil
		}
		return nil, s.createErrorResponse(err, providerID)
	}
	
	// 8. Normalize the response
	unifiedResp, err := adapter.NormalizeResponse(resp)
	if err != nil {
		return nil, s.createErrorResponse(err, providerID)
	}
	
	// 9. Add metadata
	unifiedResp.Metadata = s.router.GetRoutingMetadata(providerID, modelID, req.Preferences)
	unifiedResp.Metadata.LatencyMs = time.Since(startTime).Milliseconds()
	
	// 10. Save assistant message if session exists
	if req.SessionID != "" {
		s.messageRepo.Create(ctx, repository.Message{
			SessionID: req.SessionID,
			Role:      unifiedResp.Role,
			Content:   unifiedResp.Content,
		})
		
		// Update session
		// TODO: Extract userID from context and pass to Update
		// For now, this is a compilation fix - needs proper user context
		// s.sessionRepo.Update(ctx, userID, req.SessionID, map[string]interface{}{
		// 	"provider": providerID,
		// 	"model":    modelID,
		// })
	}
	
	return unifiedResp, nil
}

// StreamChat handles a unified streaming chat request
func (s *UnifiedChatService) StreamChat(ctx context.Context, req models.UnifiedChatRequest) (<-chan models.UnifiedStreamChunk, error) {
	fmt.Printf("[UnifiedChatService.StreamChat] Starting with ConnectionID: %s\n", req.Preferences.ConnectionID)
	
	// 1. Inject relevant context memories if available
	req = s.injectContextMemories(ctx, req)
	
	// 2. Route the request
	providerID, modelID, err := s.router.RouteRequest(ctx, req)
	if err != nil {
		fmt.Printf("[UnifiedChatService.StreamChat] Routing failed: %v\n", err)
		return s.createErrorStream(err, "routing_failed"), nil
	}
	
	fmt.Printf("[UnifiedChatService.StreamChat] Routed to provider: %s, model: %s\n", providerID, modelID)
	
	// 3. Get the provider
	provider := s.providers.Get(providerID)
	if provider == nil {
		fmt.Printf("[UnifiedChatService.StreamChat] Provider not found: %s\n", providerID)
		return s.createErrorStream(fmt.Errorf("provider not found"), providerID), nil
	}
	
	// 4. Get the adapter
	adapter := s.adapters.GetOrDefault(providerID)
	
	// 5. Normalize the request
	providerReq, err := adapter.NormalizeRequest(req)
	if err != nil {
		return s.createErrorStream(err, providerID), nil
	}
	providerReq.Model = modelID
	
	// 6. Start streaming from provider
	fmt.Printf("[UnifiedChatService.StreamChat] Calling provider.StreamComplete with model: %s\n", providerReq.Model)
	providerStream, err := provider.StreamComplete(ctx, providerReq)
	if err != nil {
		fmt.Printf("[UnifiedChatService.StreamChat] Provider stream error: %v\n", err)
		return s.createErrorStream(err, providerID), nil
	}
	
	fmt.Printf("[UnifiedChatService.StreamChat] Provider stream created successfully\n")
	
	// 7. Create unified stream
	unifiedStream := make(chan models.UnifiedStreamChunk)
	
	go func() {
		defer close(unifiedStream)
		
		fmt.Printf("[UnifiedChatService.StreamChat] Goroutine started\n")
		
		// Send initial metadata
		unifiedStream <- models.UnifiedStreamChunk{
			Type: "meta",
			Metadata: &models.ChunkMetadata{
				Provider: providerID,
				Model:    modelID,
			},
		}
		
		fmt.Printf("[UnifiedChatService.StreamChat] Sent metadata chunk\n")
		
		// Process provider chunks
		var fullContent string
		chunkCount := 0
		for chunk := range providerStream {
			chunkCount++
			fmt.Printf("[UnifiedChatService.StreamChat] Processing provider chunk %d\n", chunkCount)
			
			// Normalize chunk
			unified, err := adapter.NormalizeStreamChunk(chunk)
			if err != nil {
				fmt.Printf("[UnifiedChatService.StreamChat] Error normalizing chunk: %v\n", err)
				continue
			}
			
			// Accumulate content for saving
			if unified.Type == "content" {
				fullContent += unified.Content
			}
			
			unifiedStream <- *unified
		}
		
		fmt.Printf("[UnifiedChatService.StreamChat] Provider stream ended after %d chunks\n", chunkCount)
		
		// Save messages if session exists
		if req.SessionID != "" {
			// Only save the last user message (the new one)
			// The frontend sends the full conversation history, but we only want to persist the new message
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
	
	return unifiedStream, nil
}

// GetAvailableModels returns all available models with their capabilities
func (s *UnifiedChatService) GetAvailableModels(ctx context.Context) (*models.UnifiedModelsResponse, error) {
	var allModels []models.ModelInfo
	
	// Get models from all providers
	for providerID, provider := range s.providers.GetAll() {
		// Skip unhealthy providers
		if !s.router.GetHealthMonitor().IsHealthy(providerID) {
			continue
		}
		
		providerModels, err := provider.GetModels(ctx)
		if err != nil {
			continue
		}
		
		// Convert to unified model info
		for _, model := range providerModels {
			caps := providers.GetCapabilitiesForModel(model.ID)
			
			modelInfo := models.ModelInfo{
				ID:           fmt.Sprintf("%s/%s", providerID, model.ID),
				Provider:     providerID,
				DisplayName:  caps.DisplayName,
				Description:  caps.Description,
				Capabilities: caps.Capabilities,
				Pricing: models.PricingInfo{
					Tier: caps.PricingTier,
				},
				Status: models.ModelStatus{
					Available: true,
					Health:    "healthy",
				},
			}
			
			allModels = append(allModels, modelInfo)
		}
	}
	
	return &models.UnifiedModelsResponse{
		Models: allModels,
		Total:  len(allModels),
	}, nil
}

// tryFallback attempts to use a fallback provider/model
func (s *UnifiedChatService) tryFallback(ctx context.Context, req models.UnifiedChatRequest, err *models.UnifiedError) (*models.UnifiedChatResponse, error) {
	if err.Fallback == nil {
		return nil, fmt.Errorf("no fallback available")
	}
	
	// Override preferences with fallback
	req.Preferences.Provider = err.Fallback.Provider
	req.Preferences.Model = err.Fallback.Model
	
	// Try again with fallback
	return s.Chat(ctx, req)
}

// createErrorResponse creates an error response
func (s *UnifiedChatService) createErrorResponse(err error, provider string) error {
	if s == nil || s.adapters == nil {
		return fmt.Errorf("service error: %v", err)
	}
	adapter := s.adapters.GetOrDefault(provider)
	if adapter == nil {
		return fmt.Errorf("adapter error: %v", err)
	}
	unified := adapter.NormalizeError(err, provider)
	if unified == nil {
		return err
	}
	return fmt.Errorf("%s: %s", unified.Code, unified.Message)
}

// createErrorStream creates an error stream
func (s *UnifiedChatService) createErrorStream(err error, provider string) <-chan models.UnifiedStreamChunk {
	ch := make(chan models.UnifiedStreamChunk, 1)
	adapter := s.adapters.GetOrDefault(provider)
	unified := adapter.NormalizeError(err, provider)
	
	ch <- models.UnifiedStreamChunk{
		Type:  "error",
		Error: unified,
	}
	close(ch)
	return ch
}

// injectContextMemories enriches the request with relevant context memories
func (s *UnifiedChatService) injectContextMemories(ctx context.Context, req models.UnifiedChatRequest) models.UnifiedChatRequest {
	// TODO: Extract userID from context properly
	// For now, we'll skip if we can't get userID
	// userID := middleware.GetUserContext(ctx).UserID
	
	// Get relevant memories for this session
	if s.contextMemory != nil && req.SessionID != "" {
		// For demonstration, we'll inject memories as a system message
		// In production, this should be more sophisticated
		
		// Example of how it would work with proper user context:
		// memories, err := s.contextMemory.GetRelevant(ctx, userID, req.SessionID, 5)
		// if err == nil && len(memories) > 0 {
		//     contextMessage := s.formatMemoriesAsContext(memories)
		//     // Prepend context as system message
		//     req.Messages = append([]providers.Message{
		//         {Role: "system", Content: contextMessage},
		//     }, req.Messages...)
		// }
	}
	
	return req
}

// formatMemoriesAsContext formats memories into a context message
func (s *UnifiedChatService) formatMemoriesAsContext(memories []ContextMemory) string {
	if len(memories) == 0 {
		return ""
	}
	
	context := "Relevant context from previous conversations:\n\n"
	for _, memory := range memories {
		// Format each memory based on its namespace
		switch memory.Namespace {
		case "project":
			context += fmt.Sprintf("Project Information (%s): %s\n", memory.Key, string(memory.Value))
		case "preference":
			context += fmt.Sprintf("User Preference: %s = %s\n", memory.Key, string(memory.Value))
		case "fact":
			context += fmt.Sprintf("Known Fact: %s\n", string(memory.Value))
		default:
			context += fmt.Sprintf("%s: %s\n", memory.Key, string(memory.Value))
		}
	}
	
	context += "\nPlease use this context to provide more relevant and personalized responses."
	return context
}

// UpdateSessionTimestamp updates the session's updated_at timestamp
func (s *UnifiedChatService) UpdateSessionTimestamp(ctx context.Context, userID uuid.UUID, sessionID string) error {
	fmt.Printf("[UnifiedChatService.UpdateSessionTimestamp] Updating session %s for user %s\n", sessionID, userID.String())
	// Pass empty map - the repository will automatically set updated_at
	err := s.sessionRepo.Update(ctx, userID, sessionID, map[string]interface{}{})
	if err != nil {
		fmt.Printf("[UnifiedChatService.UpdateSessionTimestamp] Error updating session: %v\n", err)
	} else {
		fmt.Printf("[UnifiedChatService.UpdateSessionTimestamp] Successfully updated session timestamp\n")
	}
	return err
}

// MaybeAutoLabelSession checks if a session should be auto-labeled and generates a title if needed
func (s *UnifiedChatService) MaybeAutoLabelSession(ctx context.Context, userID uuid.UUID, sessionID string) error {
	// Get the session
	session, err := s.sessionRepo.Get(ctx, userID, sessionID)
	if err != nil {
		return fmt.Errorf("failed to get session: %w", err)
	}
	
	// Check if title is still default (Chat 1, Chat 2, etc)
	if !isDefaultTitle(session.Title) {
		fmt.Printf("[MaybeAutoLabelSession] Session %s has custom title '%s', skipping auto-label\n", sessionID, session.Title)
		return nil
	}
	
	// Get messages for the session
	messages, err := s.messageRepo.ListBySession(ctx, sessionID)
	if err != nil {
		return fmt.Errorf("failed to get messages: %w", err)
	}
	
	// Check if we have at least 5 messages (including system messages)
	userAndAssistantMessages := 0
	for _, msg := range messages {
		if msg.Role == "user" || msg.Role == "assistant" {
			userAndAssistantMessages++
		}
	}
	
	if userAndAssistantMessages < 5 {
		fmt.Printf("[MaybeAutoLabelSession] Session %s has only %d messages, need 5+ for auto-label\n", sessionID, userAndAssistantMessages)
		return nil
	}
	
	// Generate a title using LLM summarization
	// For auto-labeling, we don't have a connection ID, so pass empty string
	title, err := s.generateTitleWithLLM(ctx, userID.String(), messages, session, "")
	if err != nil {
		fmt.Printf("[MaybeAutoLabelSession] Failed to generate title with LLM: %v, falling back to simple extraction\n", err)
		// Fallback to simple extraction
		title = generateTitleFromMessages(messages)
	}
	
	if title == "" {
		return nil
	}
	
	// Update the session title
	fmt.Printf("[MaybeAutoLabelSession] Auto-labeling session %s with title: %s\n", sessionID, title)
	return s.sessionRepo.Update(ctx, userID, sessionID, map[string]interface{}{
		"title": title,
	})
}

// isDefaultTitle checks if a title is in the default format (Chat 1, Chat 2, etc)
func isDefaultTitle(title string) bool {
	// Check for "Chat N" pattern
	if len(title) < 5 || !strings.HasPrefix(title, "Chat ") {
		return false
	}
	
	// Check if the rest is a number
	numberPart := strings.TrimPrefix(title, "Chat ")
	for _, char := range numberPart {
		if char < '0' || char > '9' {
			return false
		}
	}
	
	return true
}

// GenerateTitleForSession generates a title for a session based on its messages (public method for API)
func (s *UnifiedChatService) GenerateTitleForSession(ctx context.Context, userID string, messages []repository.Message, session *repository.Session, connectionID string) (string, error) {
	fmt.Printf("[GenerateTitleForSession] Starting with userID: %s, connectionID: %s, messages: %d\n", userID, connectionID, len(messages))
	
	// Check for nil session
	if session == nil {
		fmt.Printf("[GenerateTitleForSession] Session is nil, using simple extraction\n")
		return generateTitleFromMessages(messages), nil
	}
	
	// Try LLM generation first
	title, err := s.generateTitleWithLLM(ctx, userID, messages, session, connectionID)
	if err != nil {
		fmt.Printf("[GenerateTitleForSession] LLM generation failed: %v, falling back to simple extraction\n", err)
		// Fall back to simple extraction
		return generateTitleFromMessages(messages), nil
	}
	return title, nil
}

// generateTitleWithLLM uses an LLM to generate a concise title from the conversation
func (s *UnifiedChatService) generateTitleWithLLM(ctx context.Context, userID string, messages []repository.Message, session *repository.Session, connectionID string) (string, error) {
	fmt.Printf("[generateTitleWithLLM] Starting with userID: %s, connectionID: %s\n", userID, connectionID)
	
	// Check for nil inputs
	if s == nil {
		return "", fmt.Errorf("service is nil")
	}
	if session == nil {
		return "", fmt.Errorf("session is nil")
	}
	
	// Check if we have any providers available
	if s.providers == nil {
		fmt.Printf("[generateTitleWithLLM] Provider registry is nil\n")
		return "", fmt.Errorf("no providers available")
	}
	
	allProviders := s.providers.GetAll()
	if len(allProviders) == 0 {
		fmt.Printf("[generateTitleWithLLM] No providers registered\n")
		return "", fmt.Errorf("no providers registered")
	}
	
	// Build conversation context for the LLM
	var conversationParts []string
	messageCount := 0
	for _, msg := range messages {
		if msg.Role == "user" || msg.Role == "assistant" {
			// Limit to first 10 messages to avoid token limits
			if messageCount >= 10 {
				break
			}
			roleLabel := "User"
			if msg.Role == "assistant" {
				roleLabel = "Assistant"
			}
			// Truncate very long messages
			content := msg.Content
			if len(content) > 500 {
				content = content[:500] + "..."
			}
			conversationParts = append(conversationParts, fmt.Sprintf("%s: %s", roleLabel, content))
			messageCount++
		}
	}
	
	conversationContext := strings.Join(conversationParts, "\n\n")
	
	// Create a prompt for title generation
	prompt := fmt.Sprintf(`Generate a concise, descriptive title for this conversation. The title should:
- Be 3-7 words maximum
- Capture the main topic or purpose of the conversation
- Be clear and specific
- Not include phrases like "Chat about" or "Discussion of"
- Not use quotation marks

Conversation:
%s

Title:`, conversationContext)
	
	// Create a request for title generation using the same provider/model as the session
	titleRequest := models.UnifiedChatRequest{
		Messages: []providers.Message{
			{
				Role:    "system",
				Content: "You are a helpful assistant that creates concise, descriptive titles for conversations. Respond with ONLY the title, nothing else.",
			},
			{
				Role:    "user",
				Content: prompt,
			},
		},
		Temperature: floatPtr(0.3), // Lower temperature for more consistent titles
		MaxTokens:   intPtr(20),    // Limit response length
	}
	
	// Use the connection ID if provided, otherwise fall back to session provider/model
	if connectionID != "" {
		// Build the full registry key if we have userID
		if userID != "" {
			titleRequest.Preferences.ConnectionID = fmt.Sprintf("%s:%s", userID, connectionID)
			fmt.Printf("[generateTitleWithLLM] Using full registry key: %s\n", titleRequest.Preferences.ConnectionID)
		} else {
			titleRequest.Preferences.ConnectionID = connectionID
			fmt.Printf("[generateTitleWithLLM] Using connection ID: %s\n", connectionID)
		}
	} else if session.Provider.Valid && session.Provider.String != "" {
		titleRequest.Preferences.Provider = session.Provider.String
		if session.Model.Valid && session.Model.String != "" {
			titleRequest.Preferences.Model = session.Model.String
		}
		fmt.Printf("[generateTitleWithLLM] Using session provider: %s, model: %s\n", session.Provider.String, session.Model.String)
	} else {
		// If neither is available, the router will use the default
		fmt.Printf("[generateTitleWithLLM] Session %s has no provider/model set and no connection ID provided, using default\n", session.ID)
	}
	
	// Make the LLM request
	fmt.Printf("[generateTitleWithLLM] About to call s.Chat with preferences: %+v\n", titleRequest.Preferences)
	
	// Remove this check - methods can't be nil in Go
	
	// Add panic recovery for the Chat call
	var response *models.UnifiedChatResponse
	var err error
	
	func() {
		defer func() {
			if r := recover(); r != nil {
				fmt.Printf("[generateTitleWithLLM] PANIC in Chat method: %v\n", r)
				err = fmt.Errorf("panic in chat: %v", r)
			}
		}()
		response, err = s.Chat(ctx, titleRequest)
	}()
	if err != nil {
		return "", fmt.Errorf("failed to generate title: %w", err)
	}
	
	// Check for nil response
	if response == nil {
		return "", fmt.Errorf("received nil response from LLM")
	}
	
	// Clean up the generated title
	title := strings.TrimSpace(response.Content)
	title = strings.Trim(title, "\"'") // Remove quotes if LLM added them
	
	// Validate title length
	if len(title) > 100 {
		// Truncate if too long
		title = title[:97] + "..."
	}
	
	if len(title) < 3 {
		// Too short, fall back to simple extraction
		return "", fmt.Errorf("generated title too short")
	}
	
	return title, nil
}

// Helper functions for creating pointers
func floatPtr(f float32) *float32 {
	return &f
}

func intPtr(i int) *int {
	return &i
}

// generateTitleFromMessages generates a meaningful title from conversation messages (fallback)
func generateTitleFromMessages(messages []repository.Message) string {
	// Find the first substantial user message
	var firstUserMessage string
	for _, msg := range messages {
		if msg.Role == "user" && len(msg.Content) > 10 {
			firstUserMessage = msg.Content
			break
		}
	}
	
	if firstUserMessage == "" {
		return ""
	}
	
	// Truncate and clean up the message to create a title
	title := firstUserMessage
	
	// Remove markdown formatting
	title = strings.ReplaceAll(title, "*", "")
	title = strings.ReplaceAll(title, "_", "")
	title = strings.ReplaceAll(title, "`", "")
	title = strings.ReplaceAll(title, "#", "")
	
	// Take first sentence or first 50 characters
	if idx := strings.Index(title, "."); idx > 0 && idx < 50 {
		title = title[:idx]
	} else if idx := strings.Index(title, "?"); idx > 0 && idx < 50 {
		title = title[:idx+1]
	} else if len(title) > 50 {
		// Find last space before 50 chars
		title = title[:50]
		if idx := strings.LastIndex(title, " "); idx > 0 {
			title = title[:idx] + "..."
		}
	}
	
	// Trim whitespace
	title = strings.TrimSpace(title)
	
	// Capitalize first letter
	if len(title) > 0 {
		title = strings.ToUpper(string(title[0])) + title[1:]
	}
	
	return title
}