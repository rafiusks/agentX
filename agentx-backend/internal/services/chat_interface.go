package services

import (
	"context"

	"github.com/google/uuid"
	"github.com/agentx/agentx-backend/internal/api/models"
	"github.com/agentx/agentx-backend/internal/repository"
)

// UnifiedChatInterface defines the contract for unified chat operations
// This interface allows us to swap implementations (old vs new) transparently
type UnifiedChatInterface interface {
	// Core chat operations
	Chat(ctx context.Context, req models.UnifiedChatRequest) (*models.UnifiedChatResponse, error)
	StreamChat(ctx context.Context, req models.UnifiedChatRequest) (<-chan models.UnifiedStreamChunk, error)
	GetAvailableModels(ctx context.Context) (*models.UnifiedModelsResponse, error)
	
	// Session management
	UpdateSessionTimestamp(ctx context.Context, userID uuid.UUID, sessionID string) error
	MaybeAutoLabelSession(ctx context.Context, userID uuid.UUID, sessionID string) error
	GenerateTitleForSession(ctx context.Context, userID string, messages []repository.Message, session *repository.Session, connectionID string) (string, error)
}

// Compile-time check that implementations satisfy the interface
var _ UnifiedChatInterface = (*UnifiedChatService)(nil)
var _ UnifiedChatInterface = (*UnifiedChatAdapter)(nil)
var _ UnifiedChatInterface = (*OrchestrationService)(nil)