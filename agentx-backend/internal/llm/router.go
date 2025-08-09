package llm

import (
	"context"
	"fmt"
	"sync"
)

// Router handles intelligent routing of requests to providers
type Router struct {
	providers     *ProviderManager
	config        *ConfigManager
	routingRules  []RoutingRule
	fallbacks     map[string]string // provider -> fallback provider
	loadBalancer  LoadBalancer
	mu            sync.RWMutex
}

// RouteInfo contains information about routing decision
type RouteInfo struct {
	Provider     string
	Model        string
	ConnectionID string
	Reason       string
	Score        float64
}

// RoutingRule defines a rule for routing requests
type RoutingRule struct {
	Name      string
	Priority  int
	Condition func(*Request) bool
	Action    func(*Request) *RouteInfo
}

// LoadBalancer interface for load balancing strategies
type LoadBalancer interface {
	SelectProvider(providers []string, req *Request) string
}

// NewRouter creates a new router
func NewRouter() *Router {
	return &Router{
		fallbacks:    make(map[string]string),
		loadBalancer: &RoundRobinBalancer{},
	}
}

// SetProviderManager sets the provider manager
func (r *Router) SetProviderManager(pm *ProviderManager) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.providers = pm
}

// SetConfigManager sets the config manager
func (r *Router) SetConfigManager(cm *ConfigManager) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.config = cm
}

// Route determines the best provider for a request
func (r *Router) Route(ctx context.Context, req *Request) (Provider, *RouteInfo, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Priority 1: Use explicit connection ID if provided
	if req.ConnectionID != "" {
		provider, err := r.providers.GetProvider(req.UserID, req.ConnectionID)
		if err == nil {
			return provider, &RouteInfo{
				Provider:     req.ConnectionID,
				ConnectionID: req.ConnectionID,
				Model:        req.Model,
				Reason:       "explicit connection ID",
			}, nil
		}
	}

	// Priority 2: Use preferences if specified
	if req.Preferences.ConnectionID != "" {
		provider, err := r.providers.GetProvider(req.UserID, req.Preferences.ConnectionID)
		if err == nil {
			model := req.Model
			if model == "" {
				model = req.Preferences.Model
			}
			return provider, &RouteInfo{
				Provider:     req.Preferences.Provider,
				ConnectionID: req.Preferences.ConnectionID,
				Model:        model,
				Reason:       "user preferences",
			}, nil
		}
	}

	// Priority 3: Apply routing rules
	for _, rule := range r.routingRules {
		if rule.Condition(req) {
			if info := rule.Action(req); info != nil {
				key := fmt.Sprintf("%s:%s", req.UserID, info.ConnectionID)
				provider, err := r.providers.GetProviderByKey(key)
				if err == nil {
					return provider, info, nil
				}
			}
		}
	}

	// Priority 4: Find best provider based on requirements
	bestProvider, bestInfo := r.findBestProvider(ctx, req)
	if bestProvider != nil {
		return bestProvider, bestInfo, nil
	}

	// Priority 5: Get default provider for user
	userProviders := r.providers.GetUserProviders(req.UserID)
	if len(userProviders) > 0 {
		// Take the first available provider
		for key, provider := range userProviders {
			return provider, &RouteInfo{
				Provider:     key,
				ConnectionID: key,
				Model:        req.Model,
				Reason:       "default user provider",
			}, nil
		}
	}

	return nil, nil, fmt.Errorf("no suitable provider found for user %s", req.UserID)
}

// findBestProvider finds the best provider based on requirements
func (r *Router) findBestProvider(ctx context.Context, req *Request) (Provider, *RouteInfo) {
	userProviders := r.providers.GetUserProviders(req.UserID)
	if len(userProviders) == 0 {
		return nil, nil
	}

	var bestProvider Provider
	var bestInfo *RouteInfo
	bestScore := -1.0

	for key, provider := range userProviders {
		score := r.scoreProvider(provider, req)
		if score > bestScore {
			bestScore = score
			bestProvider = provider
			bestInfo = &RouteInfo{
				Provider:     key,
				ConnectionID: key,
				Model:        req.Model,
				Score:        score,
				Reason:       "best match for requirements",
			}
		}
	}

	return bestProvider, bestInfo
}

// scoreProvider scores a provider based on request requirements
func (r *Router) scoreProvider(provider Provider, req *Request) float64 {
	score := 0.0
	caps := provider.GetCapabilities()

	// Check streaming support
	if req.Stream && caps.Streaming {
		score += 10.0
	} else if req.Stream && !caps.Streaming {
		return -1.0 // Disqualify
	}

	// Check tool/function support
	if len(req.Tools) > 0 && caps.FunctionCalling {
		score += 10.0
	} else if len(req.Tools) > 0 && !caps.FunctionCalling {
		return -1.0 // Disqualify
	}

	// Check model support
	if req.Model != "" {
		for _, model := range caps.SupportedModels {
			if model == req.Model {
				score += 20.0
				break
			}
		}
	}

	// Check max tokens
	if req.MaxTokens != nil && *req.MaxTokens <= caps.MaxTokens {
		score += 5.0
	}

	// Check capabilities match
	for _, reqCap := range req.Preferences.Capabilities {
		switch reqCap {
		case "vision":
			if caps.Vision {
				score += 5.0
			}
		case "audio":
			if caps.AudioInput || caps.AudioOutput {
				score += 5.0
			}
		}
	}

	return score
}

// GetFallback returns the fallback provider for a given provider
func (r *Router) GetFallback(providerKey string) Provider {
	r.mu.RLock()
	defer r.mu.RUnlock()

	fallbackKey, exists := r.fallbacks[providerKey]
	if !exists {
		return nil
	}

	provider, err := r.providers.GetProviderByKey(fallbackKey)
	if err != nil {
		return nil
	}

	return provider
}

// SetFallback sets a fallback provider
func (r *Router) SetFallback(primary, fallback string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.fallbacks[primary] = fallback
}

// AddRoutingRule adds a routing rule
func (r *Router) AddRoutingRule(rule RoutingRule) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.routingRules = append(r.routingRules, rule)
	
	// Sort by priority
	for i := 0; i < len(r.routingRules)-1; i++ {
		for j := i + 1; j < len(r.routingRules); j++ {
			if r.routingRules[i].Priority < r.routingRules[j].Priority {
				r.routingRules[i], r.routingRules[j] = r.routingRules[j], r.routingRules[i]
			}
		}
	}
}

// RoundRobinBalancer implements round-robin load balancing
type RoundRobinBalancer struct {
	current int
	mu      sync.Mutex
}

// SelectProvider selects a provider using round-robin
func (b *RoundRobinBalancer) SelectProvider(providers []string, req *Request) string {
	if len(providers) == 0 {
		return ""
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	selected := providers[b.current%len(providers)]
	b.current++
	return selected
}