package llm

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// SessionProvider interface for session operations
type SessionProvider interface {
	GetMessages(ctx context.Context, sessionID string) ([]SessionMessage, error)
}

// SessionMessage represents a message from a session
type SessionMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// Service provides a unified interface for LLM operations
type Service struct {
	gateway         *Gateway
	sessionProvider SessionProvider
	taskHandlers    map[TaskType]TaskHandler
}

// NewService creates a new LLM service
func NewService(gateway *Gateway, sessionProvider SessionProvider) *Service {
	s := &Service{
		gateway:         gateway,
		sessionProvider: sessionProvider,
		taskHandlers:    make(map[TaskType]TaskHandler),
	}
	
	// Register task handlers
	s.registerTaskHandlers()
	
	return s
}

// registerTaskHandlers registers all task-specific handlers
func (s *Service) registerTaskHandlers() {
	s.taskHandlers[TaskGenerateTitle] = &TitleGenerationHandler{service: s}
	// Add more handlers as needed
	// s.taskHandlers[TaskSummarize] = &SummarizationHandler{service: s}
}

// Complete processes a completion request
func (s *Service) Complete(ctx context.Context, userID string, req CompletionRequest) (*CompletionResponse, error) {
	startTime := time.Now()
	
	fmt.Printf("[LLM Service] Processing task: %s for user: %s\n", req.Task, userID)
	
	// Validate request
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}
	
	// Get task handler
	handler, exists := s.taskHandlers[req.Task]
	if !exists {
		// Fall back to generic handler for custom tasks
		handler = &GenericHandler{service: s}
		fmt.Printf("[LLM Service] Using generic handler for task: %s\n", req.Task)
	} else {
		fmt.Printf("[LLM Service] Using specialized handler for task: %s\n", req.Task)
	}
	
	// Inject user context
	if req.Context == nil {
		req.Context = make(map[string]interface{})
	}
	req.Context["user_id"] = userID
	
	// Resolve connection if not specified
	if req.ConnectionID == "" {
		userUUID, err := uuid.Parse(userID)
		if err != nil {
			return nil, fmt.Errorf("invalid user ID: %w", err)
		}
		
		conn, err := s.resolveConnection(ctx, userUUID, req.ProviderHints)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve connection: %w", err)
		}
		req.ConnectionID = conn
		fmt.Printf("[LLM Service] Auto-resolved connection: %s\n", conn)
	} else {
		fmt.Printf("[LLM Service] Using specified connection: %s\n", req.ConnectionID)
	}
	
	// Validate task-specific requirements
	if err := handler.ValidateRequest(req); err != nil {
		return nil, fmt.Errorf("task validation failed: %w", err)
	}
	
	// Execute task
	resp, err := handler.Handle(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("task execution failed: %w", err)
	}
	
	// Add timing information
	resp.Duration = time.Since(startTime)
	
	fmt.Printf("[LLM Service] Task completed in %v: %s\n", resp.Duration, req.Task)
	
	return resp, nil
}

// resolveConnection determines the best connection to use
func (s *Service) resolveConnection(ctx context.Context, userID uuid.UUID, hints ProviderHints) (string, error) {
	fmt.Printf("[LLM Service] Resolving connection for user: %s\n", userID.String())
	
	// Get available models from Gateway (this tells us what connections are active)
	models, err := s.gateway.GetAvailableModels(ctx, userID.String())
	if err != nil {
		return "", fmt.Errorf("failed to get available models: %w", err)
	}
	
	fmt.Printf("[LLM Service] Found %d available models\n", len(models))
	
	if len(models) == 0 {
		return "", ErrNoConnectionFound
	}
	
	// For now, return the first available connection
	// TODO: Implement proper connection selection based on hints
	if len(models) > 0 {
		// The model info should contain connection information
		// For now, we'll need to work with what we have
		fmt.Printf("[LLM Service] Using first available connection\n")
		return "auto-selected", nil // Placeholder - Gateway will handle routing
	}
	
	return "", ErrNoConnectionFound
}

// GetSupportedTasks returns a list of supported task types
func (s *Service) GetSupportedTasks() []TaskType {
	tasks := make([]TaskType, 0, len(s.taskHandlers))
	for task := range s.taskHandlers {
		tasks = append(tasks, task)
	}
	return tasks
}

// RegisterTaskHandler allows dynamic registration of task handlers
func (s *Service) RegisterTaskHandler(taskType TaskType, handler TaskHandler) {
	s.taskHandlers[taskType] = handler
	fmt.Printf("[LLM Service] Registered handler for task: %s\n", taskType)
}