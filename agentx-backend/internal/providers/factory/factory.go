package factory

import (
	"fmt"

	"github.com/agentx/agentx-backend/internal/config"
	"github.com/agentx/agentx-backend/internal/providers"
	"github.com/agentx/agentx-backend/internal/providers/anthropic"
	"github.com/agentx/agentx-backend/internal/providers/local"
	"github.com/agentx/agentx-backend/internal/providers/openai"
)

// CreateProvider creates a provider instance based on configuration
func CreateProvider(id string, cfg config.ProviderConfig) (providers.Provider, error) {
	switch cfg.Type {
	case "openai":
		return openai.NewProvider(id, cfg)
	case "anthropic":
		return anthropic.NewProvider(id, cfg)
	case "openai-compatible":
		return local.NewOpenAICompatibleProvider(id, cfg)
	case "ollama":
		// Ollama is OpenAI-compatible
		return local.NewOpenAICompatibleProvider(id, cfg)
	default:
		return nil, fmt.Errorf("unknown provider type: %s", cfg.Type)
	}
}