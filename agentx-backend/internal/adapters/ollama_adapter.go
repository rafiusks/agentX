package adapters

import (
	"github.com/agentx/agentx-backend/internal/api/models"
	"github.com/agentx/agentx-backend/internal/providers"
)

// OllamaAdapter handles Ollama-specific adaptations
type OllamaAdapter struct {
	BaseAdapter
}

// NewOllamaAdapter creates a new Ollama adapter
func NewOllamaAdapter() *OllamaAdapter {
	return &OllamaAdapter{
		BaseAdapter: NewBaseAdapter("ollama"),
	}
}

// NormalizeRequest converts unified request to Ollama format
func (a *OllamaAdapter) NormalizeRequest(req models.UnifiedChatRequest) (providers.CompletionRequest, error) {
	providerReq, err := a.BaseAdapter.NormalizeRequest(req)
	if err != nil {
		return providerReq, err
	}
	
	// Ollama-specific adjustments
	// Ollama doesn't support some advanced features, so we need to adapt
	
	// Remove unsupported features
	providerReq.Functions = nil
	providerReq.Tools = nil
	providerReq.ToolChoice = nil
	
	// Handle images for multimodal models (like llava)
	if len(req.Images) > 0 {
		// Ollama expects images in a specific format
		for i, msg := range providerReq.Messages {
			if msg.Role == "user" {
				// For Ollama, we'll append image data to the message
				// This is a simplified approach - real implementation might need model-specific handling
				if len(req.Images) > 0 && req.Images[0].Base64 != "" {
					// Ollama typically handles images differently per model
					// For now, we'll just note that images are present
					providerReq.Messages[i].Content += "\n[Image provided]"
				}
			}
		}
	}
	
	// Ensure model is specified (required for Ollama)
	if providerReq.Model == "" {
		providerReq.Model = "llama2" // Default model
	}
	
	return providerReq, nil
}

// NormalizeResponse converts Ollama response to unified format
func (a *OllamaAdapter) NormalizeResponse(resp *providers.CompletionResponse) (*models.UnifiedChatResponse, error) {
	unifiedResp, err := a.BaseAdapter.NormalizeResponse(resp)
	if err != nil {
		return nil, err
	}
	
	// Ollama-specific metadata
	unifiedResp.Metadata = models.ResponseMetadata{
		Provider: "ollama",
		Model:    resp.Model,
	}
	
	// Local models have no cost
	unifiedResp.Usage.EstimatedCost = 0
	
	// Add local processing indicator
	unifiedResp.Metadata.RoutingReason = "Local model (Ollama) - privacy-focused, no external API calls"
	
	return unifiedResp, nil
}

// NormalizeStreamChunk converts Ollama stream chunk to unified format
func (a *OllamaAdapter) NormalizeStreamChunk(chunk providers.StreamChunk) (*models.UnifiedStreamChunk, error) {
	unified, err := a.BaseAdapter.NormalizeStreamChunk(chunk)
	if err != nil {
		return nil, err
	}
	
	// Add Ollama-specific metadata
	if unified.Type == "meta" && unified.Metadata != nil {
		unified.Metadata.Provider = "ollama"
	}
	
	// Ollama doesn't support function calling, so no special handling needed
	
	return unified, nil
}

// NormalizeError converts Ollama errors to unified format
func (a *OllamaAdapter) NormalizeError(err error, provider string) *models.UnifiedError {
	unified := a.BaseAdapter.NormalizeError(err, provider)
	
	// Ollama-specific error handling
	unified.Code = ExtractErrorCode(err)
	
	// Add fallback suggestions for common Ollama errors
	if unified.Type == models.ErrorTypeNetwork {
		// Ollama connection error might mean the server isn't running
		unified.Message = "Ollama server connection failed. Ensure Ollama is running locally."
		unified.Fallback = &models.Fallback{
			Provider: "openai",
			Model:    "gpt-3.5-turbo",
			Reason:   "Local Ollama server unavailable, falling back to cloud provider",
		}
	} else if unified.Type == models.ErrorTypeModelLimit {
		// Context too long for local model
		unified.Fallback = &models.Fallback{
			Provider: "anthropic",
			Model:    "claude-3-haiku-20240307",
			Reason:   "Local model context limit exceeded, using cloud model with larger context",
		}
	}
	
	return unified
}