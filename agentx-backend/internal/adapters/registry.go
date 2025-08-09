package adapters

import (
	"fmt"
	"sync"
)

// Registry manages provider adapters
type Registry struct {
	adapters map[string]Adapter
	mu       sync.RWMutex
}

// NewRegistry creates a new adapter registry
func NewRegistry() *Registry {
	r := &Registry{
		adapters: make(map[string]Adapter),
	}
	
	// Register default adapters
	r.Register("openai", NewOpenAIAdapter())
	r.Register("anthropic", NewAnthropicAdapter())
	r.Register("ollama", NewOllamaAdapter())
	r.Register("openai-compatible", NewOpenAICompatibleAdapter())
	
	return r
}

// Register adds an adapter to the registry
func (r *Registry) Register(providerType string, adapter Adapter) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.adapters[providerType] = adapter
}

// Get retrieves an adapter by provider type
func (r *Registry) Get(providerType string) (Adapter, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	adapter, ok := r.adapters[providerType]
	if !ok {
		// Fallback to OpenAI-compatible adapter for unknown providers
		if compatAdapter, exists := r.adapters["openai-compatible"]; exists {
			return compatAdapter, nil
		}
		return nil, fmt.Errorf("no adapter found for provider type: %s", providerType)
	}
	
	return adapter, nil
}

// GetOrDefault retrieves an adapter or returns a default
func (r *Registry) GetOrDefault(providerType string) Adapter {
	if r == nil {
		return NewOpenAICompatibleAdapter()
	}
	
	adapter, err := r.Get(providerType)
	if err != nil {
		// Return OpenAI-compatible adapter as default
		return NewOpenAICompatibleAdapter()
	}
	
	if adapter == nil {
		return NewOpenAICompatibleAdapter()
	}
	
	return adapter
}

// List returns all registered provider types
func (r *Registry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	types := make([]string, 0, len(r.adapters))
	for t := range r.adapters {
		types = append(types, t)
	}
	return types
}