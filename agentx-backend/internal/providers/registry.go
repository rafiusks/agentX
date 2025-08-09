package providers

import (
	"fmt"
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
	fmt.Printf("[Registry.Register] Registered provider with key: %s (total providers: %d)\n", id, len(r.providers))
}

// Get retrieves a provider by ID
func (r *Registry) Get(id string) Provider {
	r.mu.RLock()
	defer r.mu.RUnlock()
	provider := r.providers[id]
	if provider == nil {
		fmt.Printf("[Registry.Get] Provider not found for key: %s (available keys: %d)\n", id, len(r.providers))
	} else {
		fmt.Printf("[Registry.Get] Found provider for key: %s\n", id)
	}
	return provider
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
	fmt.Printf("[Registry.Unregister] Unregistered provider with key: %s (remaining providers: %d)\n", id, len(r.providers)-1)
}

