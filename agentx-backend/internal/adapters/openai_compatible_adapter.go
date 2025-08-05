package adapters

import (
	"github.com/agentx/agentx-backend/internal/api/models"
	"github.com/agentx/agentx-backend/internal/providers"
)

// OpenAICompatibleAdapter handles OpenAI-compatible providers (LM Studio, LocalAI, etc.)
type OpenAICompatibleAdapter struct {
	BaseAdapter
}

// NewOpenAICompatibleAdapter creates a new OpenAI-compatible adapter
func NewOpenAICompatibleAdapter() *OpenAICompatibleAdapter {
	return &OpenAICompatibleAdapter{
		BaseAdapter: NewBaseAdapter("openai-compatible"),
	}
}

// NormalizeRequest converts unified request to OpenAI-compatible format
func (a *OpenAICompatibleAdapter) NormalizeRequest(req models.UnifiedChatRequest) (providers.CompletionRequest, error) {
	// Most OpenAI-compatible providers support the same format as OpenAI
	providerReq, err := a.BaseAdapter.NormalizeRequest(req)
	if err != nil {
		return providerReq, err
	}
	
	// OpenAI-compatible providers might have varying feature support
	// We'll keep most features but note that some might not work
	
	// Handle images conservatively - not all providers support vision
	if len(req.Images) > 0 {
		// Check if we should try to send images
		// For now, we'll attempt to send them in OpenAI format
		for i, msg := range providerReq.Messages {
			if msg.Role == "user" && len(req.Images) > 0 {
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
				
				providerReq.Messages[i].Content = ""
				providerReq.Messages[i].ContentArray = contents
			}
		}
	}
	
	return providerReq, nil
}

// NormalizeResponse converts OpenAI-compatible response to unified format
func (a *OpenAICompatibleAdapter) NormalizeResponse(resp *providers.CompletionResponse) (*models.UnifiedChatResponse, error) {
	unifiedResp, err := a.BaseAdapter.NormalizeResponse(resp)
	if err != nil {
		return nil, err
	}
	
	// Generic metadata for OpenAI-compatible providers
	unifiedResp.Metadata = models.ResponseMetadata{
		Provider: "openai-compatible",
		Model:    resp.Model,
	}
	
	// Most OpenAI-compatible providers are local or self-hosted
	unifiedResp.Usage.EstimatedCost = 0
	
	return unifiedResp, nil
}

// NormalizeStreamChunk converts OpenAI-compatible stream chunk to unified format
func (a *OpenAICompatibleAdapter) NormalizeStreamChunk(chunk providers.StreamChunk) (*models.UnifiedStreamChunk, error) {
	unified, err := a.BaseAdapter.NormalizeStreamChunk(chunk)
	if err != nil {
		return nil, err
	}
	
	// Add generic metadata
	if unified.Type == "meta" && unified.Metadata != nil {
		unified.Metadata.Provider = "openai-compatible"
	}
	
	return unified, nil
}

// NormalizeError converts OpenAI-compatible errors to unified format
func (a *OpenAICompatibleAdapter) NormalizeError(err error, provider string) *models.UnifiedError {
	unified := a.BaseAdapter.NormalizeError(err, provider)
	
	// Generic error handling for OpenAI-compatible providers
	unified.Code = ExtractErrorCode(err)
	
	// Fallback suggestions for OpenAI-compatible errors
	if unified.Type == models.ErrorTypeNetwork {
		unified.Fallback = &models.Fallback{
			Provider: "openai",
			Model:    "gpt-3.5-turbo",
			Reason:   "OpenAI-compatible provider unavailable, falling back to OpenAI",
		}
	} else if unified.Type == models.ErrorTypeCapability {
		// Feature not supported by this provider
		unified.Message = "This OpenAI-compatible provider may not support the requested feature"
		unified.Fallback = &models.Fallback{
			Provider: "openai",
			Model:    "gpt-4",
			Reason:   "Feature not supported by provider, using OpenAI with full feature support",
		}
	}
	
	return unified
}