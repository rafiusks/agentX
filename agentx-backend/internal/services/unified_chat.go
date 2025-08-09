package services

import (
	"context"
	"fmt"
	"time"

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
	startTime := time.Now()
	
	// 1. Inject relevant context memories if available
	req = s.injectContextMemories(ctx, req)
	
	// 2. Route the request to the best provider/model
	providerID, modelID, err := s.router.RouteRequest(ctx, req)
	if err != nil {
		return nil, s.createErrorResponse(err, "routing_failed")
	}
	
	// 3. Get the provider
	provider := s.providers.Get(providerID)
	if provider == nil {
		return nil, s.createErrorResponse(fmt.Errorf("provider not found"), providerID)
	}
	
	// 4. Get the appropriate adapter
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
	adapter := s.adapters.GetOrDefault(provider)
	unified := adapter.NormalizeError(err, provider)
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