package providers

// Capabilities defines what a provider/model can do
type Capabilities struct {
	Chat            bool     `json:"chat"`
	Streaming       bool     `json:"streaming"`
	FunctionCalling bool     `json:"function_calling"`
	Vision          bool     `json:"vision"`
	Embeddings      bool     `json:"embeddings"`
	AudioInput      bool     `json:"audio_input"`
	AudioOutput     bool     `json:"audio_output"`
	MaxContextSize  int      `json:"max_context_size"`
	OutputFormats   []string `json:"output_formats"` // text, json, code, markdown
}

// ModelCapabilities represents a model with its capabilities
type ModelCapabilities struct {
	ID            string       `json:"id"`
	Provider      string       `json:"provider"`
	DisplayName   string       `json:"display_name"`
	Description   string       `json:"description"`
	Capabilities  Capabilities `json:"capabilities"`
	PricingTier   string       `json:"pricing_tier"` // free, standard, premium
	ContextWindow int          `json:"context_window"`
	Available     bool         `json:"available"`
}

// Requirements defines what capabilities are needed for a request
type Requirements struct {
	NeedsFunctions  bool `json:"needs_functions"`
	NeedsVision     bool `json:"needs_vision"`
	NeedsStreaming  bool `json:"needs_streaming"`
	NeedsEmbeddings bool `json:"needs_embeddings"`
	MinContextSize  int  `json:"min_context_size"`
	OutputFormat    string `json:"output_format,omitempty"`
}

// ProviderCapabilities extends the Provider interface
type ProviderCapabilities interface {
	Provider
	GetCapabilities() Capabilities
	GetModelCapabilities(modelID string) *ModelCapabilities
}

// Default capabilities for known models
var DefaultModelCapabilities = map[string]ModelCapabilities{
	// OpenAI Models
	"gpt-4": {
		ID:          "gpt-4",
		Provider:    "openai",
		DisplayName: "GPT-4",
		Description: "Most capable OpenAI model for complex tasks",
		Capabilities: Capabilities{
			Chat:            true,
			Streaming:       true,
			FunctionCalling: true,
			Vision:          false,
			MaxContextSize:  8192,
			OutputFormats:   []string{"text", "json", "markdown", "code"},
		},
		PricingTier:   "premium",
		ContextWindow: 8192,
	},
	"gpt-4-vision-preview": {
		ID:          "gpt-4-vision-preview",
		Provider:    "openai",
		DisplayName: "GPT-4 Vision",
		Description: "GPT-4 with vision capabilities",
		Capabilities: Capabilities{
			Chat:            true,
			Streaming:       true,
			FunctionCalling: true,
			Vision:          true,
			MaxContextSize:  128000,
			OutputFormats:   []string{"text", "json", "markdown", "code"},
		},
		PricingTier:   "premium",
		ContextWindow: 128000,
	},
	"gpt-3.5-turbo": {
		ID:          "gpt-3.5-turbo",
		Provider:    "openai",
		DisplayName: "GPT-3.5 Turbo",
		Description: "Fast and efficient for most tasks",
		Capabilities: Capabilities{
			Chat:            true,
			Streaming:       true,
			FunctionCalling: true,
			Vision:          false,
			MaxContextSize:  16384,
			OutputFormats:   []string{"text", "json", "markdown", "code"},
		},
		PricingTier:   "standard",
		ContextWindow: 16384,
	},

	// Anthropic Models
	"claude-3-opus-20240229": {
		ID:          "claude-3-opus-20240229",
		Provider:    "anthropic",
		DisplayName: "Claude 3 Opus",
		Description: "Most capable Claude model",
		Capabilities: Capabilities{
			Chat:            true,
			Streaming:       true,
			FunctionCalling: true,
			Vision:          true,
			MaxContextSize:  200000,
			OutputFormats:   []string{"text", "json", "markdown", "code"},
		},
		PricingTier:   "premium",
		ContextWindow: 200000,
	},
	"claude-3-sonnet-20240229": {
		ID:          "claude-3-sonnet-20240229",
		Provider:    "anthropic",
		DisplayName: "Claude 3 Sonnet",
		Description: "Balanced performance and cost",
		Capabilities: Capabilities{
			Chat:            true,
			Streaming:       true,
			FunctionCalling: true,
			Vision:          true,
			MaxContextSize:  200000,
			OutputFormats:   []string{"text", "json", "markdown", "code"},
		},
		PricingTier:   "standard",
		ContextWindow: 200000,
	},
	"claude-3-haiku-20240307": {
		ID:          "claude-3-haiku-20240307",
		Provider:    "anthropic",
		DisplayName: "Claude 3 Haiku",
		Description: "Fast and cost-effective",
		Capabilities: Capabilities{
			Chat:            true,
			Streaming:       true,
			FunctionCalling: true,
			Vision:          true,
			MaxContextSize:  200000,
			OutputFormats:   []string{"text", "json", "markdown"},
		},
		PricingTier:   "economy",
		ContextWindow: 200000,
	},
}

// GetCapabilitiesForModel returns capabilities for a known model
func GetCapabilitiesForModel(modelID string) *ModelCapabilities {
	if cap, ok := DefaultModelCapabilities[modelID]; ok {
		return &cap
	}
	
	// Default capabilities for unknown models
	return &ModelCapabilities{
		ID:          modelID,
		DisplayName: modelID,
		Description: "Custom model",
		Capabilities: Capabilities{
			Chat:           true,
			Streaming:      true,
			MaxContextSize: 4096,
			OutputFormats:  []string{"text"},
		},
		PricingTier:   "standard",
		ContextWindow: 4096,
	}
}

// MeetsRequirements checks if capabilities meet requirements
func (c Capabilities) MeetsRequirements(req Requirements) bool {
	if req.NeedsFunctions && !c.FunctionCalling {
		return false
	}
	if req.NeedsVision && !c.Vision {
		return false
	}
	if req.NeedsStreaming && !c.Streaming {
		return false
	}
	if req.NeedsEmbeddings && !c.Embeddings {
		return false
	}
	if req.MinContextSize > c.MaxContextSize {
		return false
	}
	if req.OutputFormat != "" {
		found := false
		for _, format := range c.OutputFormats {
			if format == req.OutputFormat {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}