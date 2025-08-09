package repository

import (
	"context"
	"encoding/json"
	"time"
)

// ContextMemory represents a stored memory item
type ContextMemory struct {
	ID           string          `db:"id" json:"id"`
	UserID       string          `db:"user_id" json:"user_id"`
	ProjectID    *string         `db:"project_id" json:"project_id,omitempty"`
	Namespace    string          `db:"namespace" json:"namespace"`
	Key          string          `db:"key" json:"key"`
	Value        json.RawMessage `db:"value" json:"value"`
	Importance   float32         `db:"importance" json:"importance"`
	AccessCount  int             `db:"access_count" json:"access_count"`
	LastAccessed time.Time       `db:"last_accessed" json:"last_accessed"`
	ExpiresAt    *time.Time      `db:"expires_at" json:"expires_at,omitempty"`
	Metadata     json.RawMessage `db:"metadata" json:"metadata"`
	CreatedAt    time.Time       `db:"created_at" json:"created_at"`
	UpdatedAt    time.Time       `db:"updated_at" json:"updated_at"`
}

// CanvasArtifact represents a canvas work artifact
type CanvasArtifact struct {
	ID            string          `db:"id" json:"id"`
	SessionID     string          `db:"session_id" json:"session_id"`
	UserID        string          `db:"user_id" json:"user_id"`
	Type          string          `db:"type" json:"type"`
	Title         *string         `db:"title" json:"title,omitempty"`
	Content       string          `db:"content" json:"content"`
	Language      *string         `db:"language" json:"language,omitempty"`
	Version       int             `db:"version" json:"version"`
	ParentVersion *string         `db:"parent_version" json:"parent_version,omitempty"`
	IsActive      bool            `db:"is_active" json:"is_active"`
	Metadata      json.RawMessage `db:"metadata" json:"metadata"`
	CreatedAt     time.Time       `db:"created_at" json:"created_at"`
	UpdatedAt     time.Time       `db:"updated_at" json:"updated_at"`
}

// UserPattern represents a detected user behavior pattern
type UserPattern struct {
	ID          string          `db:"id" json:"id"`
	UserID      string          `db:"user_id" json:"user_id"`
	PatternType string          `db:"pattern_type" json:"pattern_type"`
	PatternData json.RawMessage `db:"pattern_data" json:"pattern_data"`
	Confidence  float32         `db:"confidence" json:"confidence"`
	Frequency   int             `db:"frequency" json:"frequency"`
	LastSeen    time.Time       `db:"last_seen" json:"last_seen"`
	Metadata    json.RawMessage `db:"metadata" json:"metadata"`
	CreatedAt   time.Time       `db:"created_at" json:"created_at"`
	UpdatedAt   time.Time       `db:"updated_at" json:"updated_at"`
}

// ContextMemoryRef represents a link between memory and session/message
type ContextMemoryRef struct {
	ID             string    `db:"id" json:"id"`
	MemoryID       string    `db:"memory_id" json:"memory_id"`
	SessionID      *string   `db:"session_id" json:"session_id,omitempty"`
	MessageID      *string   `db:"message_id" json:"message_id,omitempty"`
	RelevanceScore float32   `db:"relevance_score" json:"relevance_score"`
	CreatedAt      time.Time `db:"created_at" json:"created_at"`
}

// ContextMemoryRepository defines the interface for context memory operations
type ContextMemoryRepository interface {
	// Memory operations
	CreateMemory(ctx context.Context, memory ContextMemory) (*ContextMemory, error)
	GetMemory(ctx context.Context, userID, namespace, key string) (*ContextMemory, error)
	UpdateMemory(ctx context.Context, memory ContextMemory) error
	DeleteMemory(ctx context.Context, userID, namespace, key string) error
	ListMemoriesByNamespace(ctx context.Context, userID, namespace string, limit int) ([]ContextMemory, error)
	ListRelevantMemories(ctx context.Context, userID, sessionID string, limit int) ([]ContextMemory, error)
	SearchMemories(ctx context.Context, userID, query string, limit int) ([]ContextMemory, error)
	GetProjectMemories(ctx context.Context, userID, projectID string, limit int) ([]ContextMemory, error)
	UpdateMemoryImportance(ctx context.Context, memoryID string, importance float32) error
	CleanupExpiredMemories(ctx context.Context) (int64, error)

	// Memory reference operations
	LinkMemoryToSession(ctx context.Context, memoryID, sessionID string, relevance float32) error
	LinkMemoryToMessage(ctx context.Context, memoryID, messageID string, relevance float32) error
	GetMemoryReferences(ctx context.Context, memoryID string) ([]ContextMemoryRef, error)
}

// CanvasArtifactRepository defines the interface for canvas artifact operations
type CanvasArtifactRepository interface {
	CreateArtifact(ctx context.Context, artifact CanvasArtifact) (*CanvasArtifact, error)
	GetArtifact(ctx context.Context, id string) (*CanvasArtifact, error)
	UpdateArtifact(ctx context.Context, artifact CanvasArtifact) error
	DeleteArtifact(ctx context.Context, id string) error
	ListArtifactsBySession(ctx context.Context, sessionID string) ([]CanvasArtifact, error)
	ListArtifactsByUser(ctx context.Context, userID string, limit int) ([]CanvasArtifact, error)
	GetArtifactVersions(ctx context.Context, originalID string) ([]CanvasArtifact, error)
	SetActiveArtifact(ctx context.Context, sessionID, artifactID string) error
}

// UserPatternRepository defines the interface for user pattern operations
type UserPatternRepository interface {
	CreatePattern(ctx context.Context, pattern UserPattern) (*UserPattern, error)
	GetPattern(ctx context.Context, id string) (*UserPattern, error)
	UpdatePattern(ctx context.Context, pattern UserPattern) error
	DeletePattern(ctx context.Context, id string) error
	ListPatternsByUser(ctx context.Context, userID string, limit int) ([]UserPattern, error)
	ListPatternsByType(ctx context.Context, userID, patternType string, limit int) ([]UserPattern, error)
	IncrementPatternFrequency(ctx context.Context, patternID string) error
	UpdatePatternConfidence(ctx context.Context, patternID string, confidence float32) error
}