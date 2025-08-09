package services

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/agentx/agentx-backend/internal/api/models"
	"github.com/agentx/agentx-backend/internal/llm"
	"github.com/agentx/agentx-backend/internal/providers"
)

func TestUnifiedChatAdapter_ImplementsInterface(t *testing.T) {
	// This test verifies that UnifiedChatAdapter implements UnifiedChatInterface
	var _ UnifiedChatInterface = (*UnifiedChatAdapter)(nil)
}

func TestUnifiedChatAdapter_ConvertRequest(t *testing.T) {
	adapter := &UnifiedChatAdapter{}
	
	// Test request conversion
	unifiedReq := models.UnifiedChatRequest{
		Messages: []providers.Message{
			{Role: "user", Content: "Hello"},
		},
		Temperature: floatPtr(0.7),
		MaxTokens:   intPtr(100),
		Preferences: models.Preferences{
			Provider:     "openai",
			Model:        "gpt-4",
			ConnectionID: "user123:conn456",
		},
	}
	
	gatewayReq := adapter.convertToGatewayRequest(unifiedReq, "user123")
	
	assert.Equal(t, 1, len(gatewayReq.Messages))
	assert.Equal(t, "user", gatewayReq.Messages[0].Role)
	assert.Equal(t, "Hello", gatewayReq.Messages[0].Content)
	assert.Equal(t, float32(0.7), *gatewayReq.Temperature)
	assert.Equal(t, 100, *gatewayReq.MaxTokens)
	assert.Equal(t, "conn456", gatewayReq.Preferences.ConnectionID) // Should strip user prefix
	assert.Equal(t, "user123", gatewayReq.UserID)
}

func TestUnifiedChatAdapter_ConvertResponse(t *testing.T) {
	adapter := &UnifiedChatAdapter{}
	
	// Test response conversion
	gatewayResp := &llm.Response{
		ID:      "resp-123",
		Role:    "assistant",
		Content: "Hello back!",
		Usage: llm.Usage{
			PromptTokens:     10,
			CompletionTokens: 5,
			TotalTokens:      15,
		},
		Provider: "openai",
		Model:    "gpt-4",
		Metadata: llm.Metadata{
			LatencyMs: 100,
		},
	}
	
	unifiedResp := adapter.convertFromGatewayResponse(gatewayResp)
	
	assert.Equal(t, "resp-123", unifiedResp.ID)
	assert.Equal(t, "assistant", unifiedResp.Role)
	assert.Equal(t, "Hello back!", unifiedResp.Content)
	assert.Equal(t, 10, unifiedResp.Usage.PromptTokens)
	assert.Equal(t, 5, unifiedResp.Usage.CompletionTokens)
	assert.Equal(t, 15, unifiedResp.Usage.TotalTokens)
	assert.Equal(t, "openai", unifiedResp.Metadata.Provider)
	assert.Equal(t, "gpt-4", unifiedResp.Metadata.Model)
	assert.Equal(t, int64(100), unifiedResp.Metadata.LatencyMs)
}

func TestUnifiedChatAdapter_ExtractUserID(t *testing.T) {
	adapter := &UnifiedChatAdapter{}
	
	tests := []struct {
		name         string
		connectionID string
		expected     string
	}{
		{
			name:         "Valid format with user and connection",
			connectionID: "user123:conn456",
			expected:     "user123",
		},
		{
			name:         "Just connection ID",
			connectionID: "conn456",
			expected:     "",
		},
		{
			name:         "Empty connection ID",
			connectionID: "",
			expected:     "",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := adapter.extractUserID(context.Background(), tt.connectionID)
			assert.Equal(t, tt.expected, result)
		})
	}
}