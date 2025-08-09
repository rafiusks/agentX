package services

import (
	"context"
	"fmt"
	
	"github.com/jmoiron/sqlx"
	"github.com/agentx/agentx-backend/internal/llm"
	"github.com/agentx/agentx-backend/internal/providers"
	"github.com/agentx/agentx-backend/internal/repository"
)

// SessionProviderAdapter adapts OrchestrationService to llm.SessionProvider interface
type SessionProviderAdapter struct {
	orchestrator *OrchestrationService
}

// GetMessages implements llm.SessionProvider interface
func (s *SessionProviderAdapter) GetMessages(ctx context.Context, sessionID string) ([]llm.SessionMessage, error) {
	messages, err := s.orchestrator.GetMessages(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	
	// Convert repository.Message to llm.SessionMessage
	result := make([]llm.SessionMessage, len(messages))
	for i, msg := range messages {
		result[i] = llm.SessionMessage{
			Role:    msg.Role,
			Content: msg.Content,
		}
	}
	return result, nil
}

// Services holds all service instances
type Services struct {
	// Primary orchestrator - all clients should use this
	Orchestrator *OrchestrationService
	
	// Core services (not directly accessed by handlers)
	Gateway       *llm.Gateway
	LLM           *llm.Service      // NEW: General LLM service for all AI operations
	Connection    *ConnectionService
	ContextMemory *ContextMemoryService
	Config        *ConfigService
	
	// Legacy services (keeping minimal for compatibility)
	Chat      *ChatService        // DEPRECATED: Use Orchestrator (kept for backwards compatibility)
	Providers *providers.Registry // DEPRECATED: Use Gateway (kept for connection management)
}

// NewServices creates all service instances
func NewServices(
	db *sqlx.DB,
	providers *providers.Registry,
	sessionRepo repository.SessionRepository,
	messageRepo repository.MessageRepository,
	configRepo repository.ConfigRepository,
	connectionRepo repository.ConnectionRepository,
) *Services {
	configService := NewConfigService(configRepo)
	contextMemory := NewContextMemoryService(db)
	
	// Create LLM Gateway with proper initialization FIRST
	fmt.Printf("[Services] Initializing LLM Gateway with %d providers\n", len(providers.GetAll()))
	gateway, err := llm.InitializeGateway(providers)
	if err != nil {
		fmt.Printf("[Services] Warning: Failed to initialize LLM Gateway: %v\n", err)
		// Create a basic gateway as fallback
		gateway = llm.NewGateway(
			llm.WithMetrics(llm.NewMetricsCollector()),
			llm.WithCircuitBreaker(llm.NewCircuitBreaker()),
		)
	} else {
		fmt.Printf("[Services] LLM Gateway initialized successfully\n")
	}
	
	// Create ConnectionService with Gateway (SINGLE SOURCE OF TRUTH)
	connectionService := NewConnectionService(connectionRepo, gateway)
	
	// Create the general LLM service  
	fmt.Printf("[Services] Creating LLM service\n")
	
	// Create session provider adapter
	sessionProvider := &SessionProviderAdapter{orchestrator: nil} // Will be set after orchestrator creation
	llmService := llm.NewService(gateway, sessionProvider)
	
	// Create the main orchestrator
	orchestrator := NewOrchestrationService(
		gateway,
		sessionRepo,
		messageRepo,
		contextMemory,
		connectionService,
		configService,
		llmService,
	)
	
	// Wire up the session provider with the orchestrator
	sessionProvider.orchestrator = orchestrator
	
	return &Services{
		// Primary service
		Orchestrator: orchestrator,
		
		// Core services
		Gateway:       gateway,
		LLM:           llmService,
		Connection:    connectionService,
		ContextMemory: contextMemory,
		Config:        configService,
		
		// Minimal legacy services for compatibility
		Chat:      NewChatService(providers, sessionRepo, messageRepo),
		Providers: providers,
	}
}