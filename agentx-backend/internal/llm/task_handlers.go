package llm

import (
	"context"
	"fmt"
	"strings"
)

// TitleGenerationHandler handles title generation tasks
type TitleGenerationHandler struct {
	service *Service
}

// Handle processes title generation requests
func (h *TitleGenerationHandler) Handle(ctx context.Context, req CompletionRequest) (*CompletionResponse, error) {
	fmt.Printf("[TitleHandler] Starting title generation\n")
	
	// Extract session ID from context
	sessionID, ok := req.Context["session_id"].(string)
	if !ok || sessionID == "" {
		return nil, ErrSessionIDRequired
	}
	
	userID, _ := req.Context["user_id"].(string)
	fmt.Printf("[TitleHandler] Processing session: %s for user: %s\n", sessionID, userID)
	
	// Get session messages from session service
	sessionMessages, err := h.service.sessionProvider.GetMessages(ctx, sessionID)
	if err != nil {
		fmt.Printf("[TitleHandler] Failed to get session messages: %v\n", err)
		// Fallback to empty conversation for now
		sessionMessages = []SessionMessage{}
	}
	
	// Convert session messages to strings for prompt building and limit to recent messages
	maxMessages := 10 // Only use last 10 messages for title generation
	startIdx := 0
	if len(sessionMessages) > maxMessages {
		startIdx = len(sessionMessages) - maxMessages
	}
	
	recentMessages := sessionMessages[startIdx:]
	messages := make([]string, len(recentMessages))
	
	for i, msg := range recentMessages {
		// Truncate very long messages
		content := msg.Content
		if len(content) > 500 {
			content = content[:497] + "..."
		}
		messages[i] = fmt.Sprintf("%s: %s", msg.Role, content)
	}
	
	if len(messages) == 0 {
		fmt.Printf("[TitleHandler] No messages found for session: %s\n", sessionID)
		return nil, fmt.Errorf("no messages found in session %s", sessionID)
	}
	
	fmt.Printf("[TitleHandler] Found %d messages for session: %s (using last %d for title generation)\n", 
		len(sessionMessages), sessionID, len(messages))
	
	// Build prompt
	prompt := h.buildTitlePrompt(messages, req.Parameters)
	fmt.Printf("[TitleHandler] Built prompt - System: %d chars, User: %d chars\n", 
		len(prompt.System), len(prompt.User))
	
	// Create gateway request
	gatewayReq := &Request{
		Messages: []Message{
			{
				Role:    "system",
				Content: prompt.System,
			},
			{
				Role:    "user", 
				Content: prompt.User,
			},
		},
		MaxTokens:   h.defaultInt(req.Parameters.MaxTokens, 50),
		Temperature: h.defaultFloat(req.Parameters.Temperature, 0.7),
		UserID:      userID,
	}
	
	// Set connection preference if specified
	if req.ConnectionID != "" && req.ConnectionID != "auto-selected" {
		gatewayReq.Preferences.ConnectionID = req.ConnectionID
	}
	
	fmt.Printf("[TitleHandler] Sending request to Gateway\n")
	
	// Send to gateway
	resp, err := h.service.gateway.Complete(ctx, gatewayReq)
	if err != nil {
		fmt.Printf("[TitleHandler] Gateway request failed: %v\n", err)
		return nil, fmt.Errorf("gateway request failed: %w", err)
	}
	
	fmt.Printf("[TitleHandler] Gateway response received: '%s'\n", resp.Content)
	
	// Clean up the response
	title := strings.TrimSpace(resp.Content)
	title = strings.Trim(title, "\"'")
	
	if len(title) > 100 {
		title = title[:97] + "..."
	}
	
	// If title is too short or empty, create a fallback
	if len(title) < 3 {
		title = "Chat " + sessionID[:8]
		fmt.Printf("[TitleHandler] Using fallback title: %s\n", title)
	}
	
	// Format response
	return &CompletionResponse{
		Result:       title,
		Provider:     resp.Provider,
		ConnectionID: req.ConnectionID,
		Metadata: map[string]interface{}{
			"session_id": sessionID,
			"task":       "title_generation",
		},
		Usage: &TaskUsage{
			PromptTokens:     resp.Usage.PromptTokens,
			CompletionTokens: resp.Usage.CompletionTokens,
			TotalTokens:      resp.Usage.TotalTokens,
		},
	}, nil
}

// ValidateRequest validates title generation requests
func (h *TitleGenerationHandler) ValidateRequest(req CompletionRequest) error {
	if req.Context == nil || req.Context["session_id"] == nil {
		return ErrSessionIDRequired
	}
	return nil
}

// buildTitlePrompt constructs the prompt for title generation
func (h *TitleGenerationHandler) buildTitlePrompt(messages []string, params Parameters) struct{ System, User string } {
	systemPrompt := params.SystemPrompt
	if systemPrompt == "" {
		systemPrompt = "You are a helpful assistant that generates concise, descriptive titles for conversations. Generate a title that captures the main topic or purpose of the conversation in 5-10 words maximum. Respond with ONLY the title, nothing else."
	}
	
	// Convert messages to conversation string
	conversation := strings.Join(messages, "\n")
	
	userPrompt := params.UserPrompt
	if userPrompt == "" {
		userPrompt = fmt.Sprintf(`Generate a concise, descriptive title for this conversation. The title should:
- Be 3-7 words maximum
- Capture the main topic or purpose of the conversation
- Be clear and specific
- Not include phrases like "Chat about" or "Discussion of"
- Not use quotation marks

Conversation:
%s

Title:`, conversation)
	}
	
	return struct{ System, User string }{
		System: systemPrompt,
		User:   userPrompt,
	}
}

// GenericHandler handles custom/generic LLM tasks
type GenericHandler struct {
	service *Service
}

// Handle processes generic completion requests
func (h *GenericHandler) Handle(ctx context.Context, req CompletionRequest) (*CompletionResponse, error) {
	fmt.Printf("[GenericHandler] Processing custom task\n")
	
	// Build gateway request from parameters
	messages := []Message{}
	
	if req.Parameters.SystemPrompt != "" {
		messages = append(messages, Message{
			Role:    "system",
			Content: req.Parameters.SystemPrompt,
		})
	}
	
	if req.Parameters.UserPrompt != "" {
		messages = append(messages, Message{
			Role:    "user",
			Content: req.Parameters.UserPrompt,
		})
	}
	
	if len(messages) == 0 {
		return nil, ErrPromptRequired
	}
	
	userID, _ := req.Context["user_id"].(string)
	
	gatewayReq := &Request{
		Messages:    messages,
		MaxTokens:   h.defaultInt(req.Parameters.MaxTokens, 1000),
		Temperature: h.defaultFloat(req.Parameters.Temperature, 0.7),
		TopP:        req.Parameters.TopP,
		UserID:      userID,
	}
	
	if req.ConnectionID != "" && req.ConnectionID != "auto-selected" {
		gatewayReq.Preferences.ConnectionID = req.ConnectionID
	}
	
	resp, err := h.service.gateway.Complete(ctx, gatewayReq)
	if err != nil {
		return nil, err
	}
	
	return &CompletionResponse{
		Result:       resp.Content,
		Provider:     resp.Provider,
		ConnectionID: req.ConnectionID,
		Metadata:     req.Context,
		Usage: &TaskUsage{
			PromptTokens:     resp.Usage.PromptTokens,
			CompletionTokens: resp.Usage.CompletionTokens,
			TotalTokens:      resp.Usage.TotalTokens,
		},
	}, nil
}

// ValidateRequest validates generic requests
func (h *GenericHandler) ValidateRequest(req CompletionRequest) error {
	if req.Parameters.SystemPrompt == "" && req.Parameters.UserPrompt == "" {
		return ErrPromptRequired
	}
	return nil
}

// Helper methods
func (h *TitleGenerationHandler) defaultInt(val *int, def int) *int {
	if val == nil {
		return &def
	}
	return val
}

func (h *TitleGenerationHandler) defaultFloat(val *float32, def float32) *float32 {
	if val == nil {
		return &def
	}
	return val
}

func (h *GenericHandler) defaultInt(val *int, def int) *int {
	if val == nil {
		return &def
	}
	return val
}

func (h *GenericHandler) defaultFloat(val *float32, def float32) *float32 {
	if val == nil {
		return &def
	}
	return val
}