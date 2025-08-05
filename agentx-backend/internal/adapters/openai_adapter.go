package adapters

import (
	"github.com/agentx/agentx-backend/internal/api/models"
	"github.com/agentx/agentx-backend/internal/providers"
)

// OpenAIAdapter handles OpenAI-specific adaptations
type OpenAIAdapter struct {
	BaseAdapter
}

// NewOpenAIAdapter creates a new OpenAI adapter
func NewOpenAIAdapter() *OpenAIAdapter {
	return &OpenAIAdapter{
		BaseAdapter: NewBaseAdapter("openai"),
	}
}

// NormalizeRequest converts unified request to OpenAI format
func (a *OpenAIAdapter) NormalizeRequest(req models.UnifiedChatRequest) (providers.CompletionRequest, error) {
	// Use base implementation
	providerReq, err := a.BaseAdapter.NormalizeRequest(req)
	if err != nil {
		return providerReq, err
	}
	
	// OpenAI-specific adjustments
	// Handle images for vision models
	if len(req.Images) > 0 {
		// Convert images to content format
		for i, msg := range providerReq.Messages {
			if msg.Role == "user" {
				// OpenAI expects images in a specific content format
				contents := []interface{}{
					map[string]string{"type": "text", "text": msg.Content},
				}
				
				for _, img := range req.Images {
					imageContent := map[string]interface{}{
						"type": "image_url",
						"image_url": map[string]string{
							"url": img.URL,
						},
					}
					if img.Base64 != "" {
						imageContent["image_url"] = map[string]string{
							"url": "data:" + img.Type + ";base64," + img.Base64,
						}
					}
					contents = append(contents, imageContent)
				}
				
				// Update message with multi-content format
				providerReq.Messages[i].Content = ""
				providerReq.Messages[i].ContentArray = contents
			}
		}
	}
	
	return providerReq, nil
}

// NormalizeResponse converts OpenAI response to unified format
func (a *OpenAIAdapter) NormalizeResponse(resp *providers.CompletionResponse) (*models.UnifiedChatResponse, error) {
	unifiedResp, err := a.BaseAdapter.NormalizeResponse(resp)
	if err != nil {
		return nil, err
	}
	
	// OpenAI-specific metadata
	unifiedResp.Metadata = models.ResponseMetadata{
		Provider: "openai",
		Model:    resp.Model,
	}
	
	// Calculate estimated cost
	unifiedResp.Usage.EstimatedCost = EstimateCost(unifiedResp.Usage, "openai", resp.Model)
	
	return unifiedResp, nil
}

// NormalizeStreamChunk converts OpenAI stream chunk to unified format
func (a *OpenAIAdapter) NormalizeStreamChunk(chunk providers.StreamChunk) (*models.UnifiedStreamChunk, error) {
	unified, err := a.BaseAdapter.NormalizeStreamChunk(chunk)
	if err != nil {
		return nil, err
	}
	
	// Add OpenAI-specific metadata
	if unified.Type == "meta" && unified.Metadata != nil {
		unified.Metadata.Provider = "openai"
	}
	
	return unified, nil
}

// NormalizeError converts OpenAI errors to unified format
func (a *OpenAIAdapter) NormalizeError(err error, provider string) *models.UnifiedError {
	unified := a.BaseAdapter.NormalizeError(err, provider)
	
	// OpenAI-specific error code extraction
	unified.Code = ExtractErrorCode(err)
	
	// Add fallback suggestions for common OpenAI errors
	if unified.Type == models.ErrorTypeRateLimit {
		unified.Fallback = &models.Fallback{
			Provider: "anthropic",
			Model:    "claude-3-haiku-20240307",
			Reason:   "OpenAI rate limit reached, falling back to Anthropic",
		}
	} else if unified.Type == models.ErrorTypeModelLimit {
		unified.Fallback = &models.Fallback{
			Provider: "openai",
			Model:    "gpt-3.5-turbo-16k",
			Reason:   "Context too long for requested model, trying longer context model",
		}
	}
	
	return unified
}