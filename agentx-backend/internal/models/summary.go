package models

import "time"

// SessionSummary represents a summary of conversation messages
type SessionSummary struct {
	ID             string    `json:"id"`
	SessionID      string    `json:"session_id"`
	UserID         string    `json:"user_id"`
	SummaryText    string    `json:"summary_text"`
	MessageCount   int       `json:"message_count"`
	StartMessageID string    `json:"start_message_id,omitempty"`
	EndMessageID   string    `json:"end_message_id,omitempty"`
	TokensSaved    int       `json:"tokens_saved"`
	ModelUsed      string    `json:"model_used"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}