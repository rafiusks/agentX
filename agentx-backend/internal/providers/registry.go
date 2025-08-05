package providers

import (
	"sync"
)

// Registry manages all available providers
type Registry struct {
	providers map[string]Provider
	mu        sync.RWMutex
}

// NewRegistry creates a new provider registry
func NewRegistry() *Registry {
	return &Registry{
		providers: make(map[string]Provider),
	}
}

// Register adds a provider to the registry
func (r *Registry) Register(id string, provider Provider) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.providers[id] = provider
}

// Get retrieves a provider by ID
func (r *Registry) Get(id string) Provider {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.providers[id]
}

// List returns all registered provider IDs
func (r *Registry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	ids := make([]string, 0, len(r.providers))
	for id := range r.providers {
		ids = append(ids, id)
	}
	return ids
}

// GetAll returns all registered providers
func (r *Registry) GetAll() map[string]Provider {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	// Return a copy to prevent external modification
	providers := make(map[string]Provider, len(r.providers))
	for k, v := range r.providers {
		providers[k] = v
	}
	return providers
}

// Has checks if a provider is registered
func (r *Registry) Has(id string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, exists := r.providers[id]
	return exists
}

// Unregister removes a provider from the registry
func (r *Registry) Unregister(id string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.providers, id)
}

