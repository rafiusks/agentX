package adapters

import (
	"github.com/agentx/agentx-backend/internal/api/models"
	"github.com/agentx/agentx-backend/internal/providers"
)

// AnthropicAdapter handles Anthropic-specific adaptations
type AnthropicAdapter struct {
	BaseAdapter
}

// NewAnthropicAdapter creates a new Anthropic adapter
func NewAnthropicAdapter() *AnthropicAdapter {
	return &AnthropicAdapter{
		BaseAdapter: NewBaseAdapter("anthropic"),
	}
}

// NormalizeRequest converts unified request to Anthropic format
func (a *AnthropicAdapter) NormalizeRequest(req models.UnifiedChatRequest) (providers.CompletionRequest, error) {
	providerReq, err := a.BaseAdapter.NormalizeRequest(req)
	if err != nil {
		return providerReq, err
	}
	
	// Anthropic-specific adjustments
	// Handle system messages (Anthropic uses a separate system parameter)
	// This is already handled in the Anthropic provider, but we can validate here
	
	// Handle images for vision models
	if len(req.Images) > 0 {
		// Anthropic expects images in message content
		for i, msg := range providerReq.Messages {
			if msg.Role == "user" {
				// Create content array with text and images
				contents := []interface{}{
					map[string]string{"type": "text", "text": msg.Content},
				}
				
				for _, img := range req.Images {
					imageContent := map[string]interface{}{
						"type": "image",
						"source": map[string]string{
							"type": "base64",
							"media_type": img.Type,
							"data": img.Base64,
						},
					}
					if img.URL != "" {
						// Anthropic doesn't support URLs directly, would need to fetch and convert
						// For now, skip URL images or implement fetching logic
						continue
					}
					contents = append(contents, imageContent)
				}
				
				// Update message with multi-content format
				providerReq.Messages[i].Content = ""
				providerReq.Messages[i].ContentArray = contents
			}
		}
	}
	
	// Default max tokens if not specified (Anthropic requires this)
	if providerReq.MaxTokens == nil {
		maxTokens := 4096
		providerReq.MaxTokens = &maxTokens
	}
	
	return providerReq, nil
}

// NormalizeResponse converts Anthropic response to unified format
func (a *AnthropicAdapter) NormalizeResponse(resp *providers.CompletionResponse) (*models.UnifiedChatResponse, error) {
	unifiedResp, err := a.BaseAdapter.NormalizeResponse(resp)
	if err != nil {
		return nil, err
	}
	
	// Anthropic-specific metadata
	unifiedResp.Metadata = models.ResponseMetadata{
		Provider: "anthropic",
		Model:    resp.Model,
	}
	
	// Calculate estimated cost
	unifiedResp.Usage.EstimatedCost = EstimateCost(unifiedResp.Usage, "anthropic", resp.Model)
	
	return unifiedResp, nil
}

// NormalizeStreamChunk converts Anthropic stream chunk to unified format
func (a *AnthropicAdapter) NormalizeStreamChunk(chunk providers.StreamChunk) (*models.UnifiedStreamChunk, error) {
	unified, err := a.BaseAdapter.NormalizeStreamChunk(chunk)
	if err != nil {
		return nil, err
	}
	
	// Add Anthropic-specific metadata
	if unified.Type == "meta" && unified.Metadata != nil {
		unified.Metadata.Provider = "anthropic"
	}
	
	// Anthropic sends tool use in a specific format
	if chunk.ToolCalls != nil && len(chunk.ToolCalls) > 0 {
		// Anthropic streams tool calls differently
		unified.Type = "tool_use"
	}
	
	return unified, nil
}

// NormalizeError converts Anthropic errors to unified format
func (a *AnthropicAdapter) NormalizeError(err error, provider string) *models.UnifiedError {
	unified := a.BaseAdapter.NormalizeError(err, provider)
	
	// Anthropic-specific error code extraction
	unified.Code = ExtractErrorCode(err)
	
	// Add fallback suggestions for common Anthropic errors
	if unified.Type == models.ErrorTypeRateLimit {
		unified.Fallback = &models.Fallback{
			Provider: "openai",
			Model:    "gpt-3.5-turbo",
			Reason:   "Anthropic rate limit reached, falling back to OpenAI",
		}
	} else if unified.Type == models.ErrorTypeModelLimit {
		// Anthropic has very large context windows, so this is rare
		unified.Fallback = &models.Fallback{
			Provider: "anthropic",
			Model:    "claude-3-haiku-20240307",
			Reason:   "Switching to more efficient model for large context",
		}
	}
	
	return unified
}