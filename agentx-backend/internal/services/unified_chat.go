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
}

// NewUnifiedChatService creates a new unified chat service
func NewUnifiedChatService(
	providers *providers.Registry,
	configService *ConfigService,
	sessionRepo repository.SessionRepository,
	messageRepo repository.MessageRepository,
) *UnifiedChatService {
	return &UnifiedChatService{
		providers:   providers,
		router:      NewRequestRouter(providers, configService),
		adapters:    adapters.NewRegistry(),
		sessionRepo: sessionRepo,
		messageRepo: messageRepo,
	}
}

// Chat handles a unified chat request
func (s *UnifiedChatService) Chat(ctx context.Context, req models.UnifiedChatRequest) (*models.UnifiedChatResponse, error) {
	startTime := time.Now()
	
	// 1. Route the request to the best provider/model
	providerID, modelID, err := s.router.RouteRequest(ctx, req)
	if err != nil {
		return nil, s.createErrorResponse(err, "routing_failed")
	}
	
	// 2. Get the provider
	provider := s.providers.Get(providerID)
	if provider == nil {
		return nil, s.createErrorResponse(fmt.Errorf("provider not found"), providerID)
	}
	
	// 3. Get the appropriate adapter
	adapter := s.adapters.GetOrDefault(providerID)
	
	// 4. Normalize the request for the provider
	providerReq, err := adapter.NormalizeRequest(req)
	if err != nil {
		return nil, s.createErrorResponse(err, providerID)
	}
	providerReq.Model = modelID
	
	// 5. Save user message if session exists
	if req.SessionID != "" {
		for _, msg := range req.Messages {
			if msg.Role == "user" {
				s.messageRepo.Create(ctx, repository.Message{
					SessionID: req.SessionID,
					Role:      msg.Role,
					Content:   msg.Content,
				})
			}
		}
	}
	
	// 6. Make the request to the provider
	resp, err := provider.Complete(ctx, providerReq)
	if err != nil {
		// Try fallback if available
		if fallbackResp, fallbackErr := s.tryFallback(ctx, req, adapter.NormalizeError(err, providerID)); fallbackErr == nil {
			return fallbackResp, nil
		}
		return nil, s.createErrorResponse(err, providerID)
	}
	
	// 7. Normalize the response
	unifiedResp, err := adapter.NormalizeResponse(resp)
	if err != nil {
		return nil, s.createErrorResponse(err, providerID)
	}
	
	// 8. Add metadata
	unifiedResp.Metadata = s.router.GetRoutingMetadata(providerID, modelID, req.Preferences)
	unifiedResp.Metadata.LatencyMs = time.Since(startTime).Milliseconds()
	
	// 9. Save assistant message if session exists
	if req.SessionID != "" {
		s.messageRepo.Create(ctx, repository.Message{
			SessionID: req.SessionID,
			Role:      unifiedResp.Role,
			Content:   unifiedResp.Content,
		})
		
		// Update session
		s.sessionRepo.Update(ctx, req.SessionID, map[string]interface{}{
			"provider": providerID,
			"model":    modelID,
		})
	}
	
	return unifiedResp, nil
}

// StreamChat handles a unified streaming chat request
func (s *UnifiedChatService) StreamChat(ctx context.Context, req models.UnifiedChatRequest) (<-chan models.UnifiedStreamChunk, error) {
	// 1. Route the request
	providerID, modelID, err := s.router.RouteRequest(ctx, req)
	if err != nil {
		return s.createErrorStream(err, "routing_failed"), nil
	}
	
	// 2. Get the provider
	provider := s.providers.Get(providerID)
	if provider == nil {
		return s.createErrorStream(fmt.Errorf("provider not found"), providerID), nil
	}
	
	// 3. Get the adapter
	adapter := s.adapters.GetOrDefault(providerID)
	
	// 4. Normalize the request
	providerReq, err := adapter.NormalizeRequest(req)
	if err != nil {
		return s.createErrorStream(err, providerID), nil
	}
	providerReq.Model = modelID
	
	// 5. Start streaming from provider
	providerStream, err := provider.StreamComplete(ctx, providerReq)
	if err != nil {
		return s.createErrorStream(err, providerID), nil
	}
	
	// 6. Create unified stream
	unifiedStream := make(chan models.UnifiedStreamChunk)
	
	go func() {
		defer close(unifiedStream)
		
		// Send initial metadata
		unifiedStream <- models.UnifiedStreamChunk{
			Type: "meta",
			Metadata: &models.ChunkMetadata{
				Provider: providerID,
				Model:    modelID,
			},
		}
		
		// Process provider chunks
		var fullContent string
		for chunk := range providerStream {
			// Normalize chunk
			unified, err := adapter.NormalizeStreamChunk(chunk)
			if err != nil {
				continue
			}
			
			// Accumulate content for saving
			if unified.Type == "content" {
				fullContent += unified.Content
			}
			
			unifiedStream <- *unified
		}
		
		// Save messages if session exists
		if req.SessionID != "" {
			// Save user message
			for _, msg := range req.Messages {
				if msg.Role == "user" {
					s.messageRepo.Create(ctx, repository.Message{
						SessionID: req.SessionID,
						Role:      msg.Role,
						Content:   msg.Content,
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