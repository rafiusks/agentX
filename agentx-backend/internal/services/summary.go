package services

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/agentx/agentx-backend/internal/db"
	"github.com/agentx/agentx-backend/internal/llm"
	"github.com/agentx/agentx-backend/internal/models"
	"github.com/agentx/agentx-backend/internal/repository"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type SummaryService struct {
	db         *db.Database
	llmService *LLMService
}

func NewSummaryService(database *db.Database, llmService *LLMService) *SummaryService {
	return &SummaryService{
		db:         database,
		llmService: llmService,
	}
}

// GenerateSessionSummary generates a summary of recent messages in a session
func (s *SummaryService) GenerateSessionSummary(ctx context.Context, userID, sessionID string, messageCount int) (*models.SessionSummary, error) {
	// Get recent messages from the session
	query := `
		SELECT id, role, content, created_at 
		FROM messages 
		WHERE session_id = $1 AND user_id = $2
		ORDER BY created_at DESC
		LIMIT $3
	`
	
	// Convert Pool to sqlx.DB
	sqlDB, ok := s.db.Pool.(*sqlx.DB)
	if !ok {
		return nil, fmt.Errorf("database connection type mismatch")
	}
	
	rows, err := sqlDB.QueryContext(ctx, query, sessionID, userID, messageCount)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch messages: %w", err)
	}
	defer rows.Close()

	var messages []repository.Message
	var firstMessageID, lastMessageID string
	
	for rows.Next() {
		var msg repository.Message
		err := rows.Scan(&msg.ID, &msg.Role, &msg.Content, &msg.CreatedAt)
		if err != nil {
			continue
		}
		messages = append(messages, msg)
		
		if firstMessageID == "" {
			firstMessageID = msg.ID
		}
		lastMessageID = msg.ID
	}

	if len(messages) == 0 {
		return nil, fmt.Errorf("no messages found to summarize")
	}

	// Reverse to chronological order
	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}

	// Build conversation text
	var conversationBuilder strings.Builder
	for _, msg := range messages {
		conversationBuilder.WriteString(fmt.Sprintf("%s: %s\n\n", strings.Title(msg.Role), msg.Content))
	}

	// Generate summary using LLM
	summaryPrompt := fmt.Sprintf(`Create a concise summary of this conversation that can replace the full messages in an LLM context.

Include ONLY the essential information needed for context continuity:
- Key facts established
- Important decisions made  
- Critical technical details (errors, code snippets if crucial)
- Current task/question being worked on
- User preferences discovered

Format as a brief narrative (max 200 words) that provides context for continuing the conversation.

Conversation to summarize:
%s

Summary:`, conversationBuilder.String())

	// Use the LLM service to generate summary
	summaryReq := &llm.CompletionRequest{
		Task: "summarize",
		Context: map[string]interface{}{
			"session_id": sessionID,
			"messages":   summaryPrompt,
		},
		Parameters: llm.Parameters{
			Temperature: floatPtr(0.7),
			MaxTokens:   intPtr(500),
		},
	}

	response, err := s.llmService.Complete(ctx, summaryReq)
	if err != nil {
		return nil, fmt.Errorf("failed to generate summary: %w", err)
	}

	// Calculate tokens saved (approximate)
	originalTokens := len(conversationBuilder.String()) / 4 // Rough estimate
	summaryTokens := len(response.Result) / 4
	tokensSaved := originalTokens - summaryTokens

	// Save summary to database
	summaryID := uuid.New()
	insertQuery := `
		INSERT INTO session_summaries 
		(id, session_id, user_id, summary_text, message_count, 
		 start_message_id, end_message_id, tokens_saved, model_used)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING created_at
	`
	
	var createdAt time.Time
	err = sqlDB.QueryRowContext(ctx, insertQuery,
		summaryID,
		sessionID,
		userID,
		response.Result,
		len(messages),
		lastMessageID,  // oldest message
		firstMessageID, // newest message
		tokensSaved,
		response.Provider,
	).Scan(&createdAt)
	
	if err != nil {
		return nil, fmt.Errorf("failed to save summary: %w", err)
	}

	return &models.SessionSummary{
		ID:             summaryID.String(),
		SessionID:      sessionID,
		UserID:         userID,
		SummaryText:    response.Result,
		MessageCount:   len(messages),
		StartMessageID: lastMessageID,
		EndMessageID:   firstMessageID,
		TokensSaved:    tokensSaved,
		ModelUsed:      response.Provider,
		CreatedAt:      createdAt,
	}, nil
}

// GetSessionSummaries retrieves summaries for a session
func (s *SummaryService) GetSessionSummaries(ctx context.Context, userID, sessionID string) ([]*models.SessionSummary, error) {
	// Convert Pool to sqlx.DB
	sqlDB, ok := s.db.Pool.(*sqlx.DB)
	if !ok {
		return nil, fmt.Errorf("database connection type mismatch")
	}
	
	query := `
		SELECT id, session_id, user_id, summary_text, message_count,
		       start_message_id, end_message_id, tokens_saved, model_used, created_at
		FROM session_summaries
		WHERE session_id = $1 AND user_id = $2
		ORDER BY created_at DESC
	`
	
	rows, err := sqlDB.QueryContext(ctx, query, sessionID, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch summaries: %w", err)
	}
	defer rows.Close()

	var summaries []*models.SessionSummary
	for rows.Next() {
		var summary models.SessionSummary
		err := rows.Scan(
			&summary.ID,
			&summary.SessionID,
			&summary.UserID,
			&summary.SummaryText,
			&summary.MessageCount,
			&summary.StartMessageID,
			&summary.EndMessageID,
			&summary.TokensSaved,
			&summary.ModelUsed,
			&summary.CreatedAt,
		)
		if err != nil {
			continue
		}
		summaries = append(summaries, &summary)
	}

	return summaries, nil
}

// AutoGenerateSummary checks if a session needs summarization and generates it
func (s *SummaryService) AutoGenerateSummary(ctx context.Context, userID, sessionID string) error {
	// Convert Pool to sqlx.DB
	sqlDB, ok := s.db.Pool.(*sqlx.DB)
	if !ok {
		return fmt.Errorf("database connection type mismatch")
	}
	
	// Check message count since last summary
	var messageCount int
	countQuery := `
		SELECT COUNT(*) 
		FROM messages m
		WHERE m.session_id = $1 AND m.user_id = $2
		AND NOT EXISTS (
			SELECT 1 FROM session_summaries s
			WHERE s.session_id = m.session_id
			AND m.created_at <= (
				SELECT created_at FROM messages WHERE id = s.end_message_id
			)
		)
	`
	
	err := sqlDB.QueryRowContext(ctx, countQuery, sessionID, userID).Scan(&messageCount)
	if err != nil {
		return fmt.Errorf("failed to count messages: %w", err)
	}

	// Generate summary if we have enough new messages (e.g., every 20 messages)
	if messageCount >= 20 {
		_, err = s.GenerateSessionSummary(ctx, userID, sessionID, 20)
		if err != nil {
			return fmt.Errorf("failed to generate summary: %w", err)
		}
	}

	return nil
}

// Helper functions moved to avoid redeclaration