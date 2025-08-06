package audit

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/agentx/agentx-backend/internal/models"
)

// EventType represents the type of audit event
type EventType string

const (
	EventLogin           EventType = "user.login"
	EventLogout          EventType = "user.logout"
	EventSignup          EventType = "user.signup"
	EventPasswordChange  EventType = "user.password_change"
	EventProfileUpdate   EventType = "user.profile_update"
	EventAPIKeyCreate    EventType = "api_key.create"
	EventAPIKeyRevoke    EventType = "api_key.revoke"
	EventAPIKeyUse       EventType = "api_key.use"
	EventConnectionCreate EventType = "connection.create"
	EventConnectionUpdate EventType = "connection.update"
	EventConnectionDelete EventType = "connection.delete"
	EventChatMessage     EventType = "chat.message"
	EventSessionCreate   EventType = "session.create"
	EventSessionDelete   EventType = "session.delete"
	EventAdminAction     EventType = "admin.action"
)

// Logger defines the interface for audit logging
type Logger interface {
	Log(ctx context.Context, event *Event) error
	GetUserEvents(ctx context.Context, userID uuid.UUID, limit int) ([]*Event, error)
	GetSystemEvents(ctx context.Context, limit int) ([]*Event, error)
}

// Event represents an audit event
type Event struct {
	ID        uuid.UUID              `json:"id"`
	EventType EventType              `json:"event_type"`
	UserID    *uuid.UUID             `json:"user_id,omitempty"`
	IPAddress string                 `json:"ip_address,omitempty"`
	UserAgent string                 `json:"user_agent,omitempty"`
	Resource  string                 `json:"resource,omitempty"`
	Action    string                 `json:"action,omitempty"`
	Result    string                 `json:"result,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt time.Time              `json:"created_at"`
}

// Repository defines the interface for audit log persistence
type Repository interface {
	Log(ctx context.Context, log *models.AuditLog) error
	GetByUserID(ctx context.Context, userID uuid.UUID, limit int) ([]*models.AuditLog, error)
	GetRecentLogs(ctx context.Context, limit int) ([]*models.AuditLog, error)
}

// Service implements the audit logger
type Service struct {
	repo Repository
}

// NewService creates a new audit service
func NewService(repo Repository) *Service {
	return &Service{
		repo: repo,
	}
}

// Log records an audit event
func (s *Service) Log(ctx context.Context, event *Event) error {
	// Create audit log model
	log := &models.AuditLog{
		ID:           event.ID,
		UserID:       event.UserID,
		Action:       event.Action,
		ResourceType: event.Resource,
		IPAddress:    event.IPAddress,
		UserAgent:    event.UserAgent,
		Metadata:     models.JSONB(event.Metadata),
		Status:       event.Result,
		CreatedAt:    event.CreatedAt,
	}

	// Save to database
	return s.repo.Log(ctx, log)
}

// GetUserEvents retrieves audit events for a specific user
func (s *Service) GetUserEvents(ctx context.Context, userID uuid.UUID, limit int) ([]*Event, error) {
	logs, err := s.repo.GetByUserID(ctx, userID, limit)
	if err != nil {
		return nil, err
	}

	events := make([]*Event, len(logs))
	for i, log := range logs {
		metadata := map[string]interface{}(log.Metadata)

		events[i] = &Event{
			ID:        log.ID,
			EventType: EventType(log.Action),
			UserID:    log.UserID,
			IPAddress: log.IPAddress,
			UserAgent: log.UserAgent,
			Resource:  log.ResourceType,
			Action:    log.Action,
			Result:    log.Status,
			Metadata:  metadata,
			CreatedAt: log.CreatedAt,
		}
	}

	return events, nil
}

// GetSystemEvents retrieves system-wide audit events
func (s *Service) GetSystemEvents(ctx context.Context, limit int) ([]*Event, error) {
	logs, err := s.repo.GetRecentLogs(ctx, limit)
	if err != nil {
		return nil, err
	}

	events := make([]*Event, len(logs))
	for i, log := range logs {
		metadata := map[string]interface{}(log.Metadata)

		events[i] = &Event{
			ID:        log.ID,
			EventType: EventType(log.Action),
			UserID:    log.UserID,
			IPAddress: log.IPAddress,
			UserAgent: log.UserAgent,
			Resource:  log.ResourceType,
			Action:    log.Action,
			Result:    log.Status,
			Metadata:  metadata,
			CreatedAt: log.CreatedAt,
		}
	}

	return events, nil
}

// Helper function to create an event
func NewEvent(eventType EventType, userID *uuid.UUID, ipAddress, userAgent string) *Event {
	return &Event{
		ID:        uuid.New(),
		EventType: eventType,
		UserID:    userID,
		IPAddress: ipAddress,
		UserAgent: userAgent,
		CreatedAt: time.Now(),
		Metadata:  make(map[string]interface{}),
	}
}