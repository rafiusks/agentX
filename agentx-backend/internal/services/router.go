package services

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/agentx/agentx-backend/internal/api/models"
	"github.com/agentx/agentx-backend/internal/providers"
)

// RequestRouter intelligently routes requests to appropriate providers
type RequestRouter struct {
	providers      *providers.Registry
	configService  *ConfigService
	healthMonitor  *HealthMonitor
}

// GetHealthMonitor returns the health monitor
func (r *RequestRouter) GetHealthMonitor() *HealthMonitor {
	return r.healthMonitor
}

// NewRequestRouter creates a new request router
func NewRequestRouter(providers *providers.Registry, configService *ConfigService) *RequestRouter {
	var healthMon *HealthMonitor
	if providers != nil {
		healthMon = NewHealthMonitor(providers)
	}
	return &RequestRouter{
		providers:     providers,
		configService: configService,
		healthMonitor: healthMon,
	}
}

// RouteRequest determines the best provider and model for a request
func (r *RequestRouter) RouteRequest(ctx context.Context, req models.UnifiedChatRequest) (string, string, error) {
	// Check for nil router
	if r == nil {
		fmt.Printf("[RequestRouter.RouteRequest] Router is nil\n")
		return "", "", fmt.Errorf("router is nil")
	}
	
	// Check for nil providers registry
	if r.providers == nil {
		fmt.Printf("[RequestRouter.RouteRequest] Providers registry is nil\n")
		return "", "", fmt.Errorf("providers registry is nil")
	}
	// Handle connection_id if specified
	if req.Preferences.ConnectionID != "" {
		fmt.Printf("[RequestRouter.RouteRequest] Looking for connection: %s\n", req.Preferences.ConnectionID)
		
		// List all available providers
		allProviders := r.providers.GetAll()
		fmt.Printf("[RequestRouter.RouteRequest] Total providers available: %d\n", len(allProviders))
		for key := range allProviders {
			fmt.Printf("[RequestRouter.RouteRequest] Provider key: %s\n", key)
		}
		
		// Build the correct registry key: userID:connectionID
		// First try with just the connection ID (backwards compatibility)
		provider := r.providers.Get(req.Preferences.ConnectionID)
		
		// If not found, try to find it in the registry by scanning for matching connection IDs
		if provider == nil {
			// Look for any key ending with :connectionID
			for key, p := range allProviders {
				parts := strings.Split(key, ":")
				if len(parts) == 2 && parts[1] == req.Preferences.ConnectionID {
					fmt.Printf("[RequestRouter.RouteRequest] Found provider with key: %s\n", key)
					provider = p
					req.Preferences.ConnectionID = key // Update to full key
					break
				}
			}
		}
		
		if provider == nil {
			fmt.Printf("[RequestRouter.RouteRequest] Provider not found for connection: %s, will try fallback\n", req.Preferences.ConnectionID)
			// Don't fail immediately, clear the connection ID and fall through to default routing
			req.Preferences.ConnectionID = ""
		} else {
			// Get default model for this provider
			models, err := provider.GetModels(ctx)
			if err != nil || len(models) == 0 {
				// Fallback to a default model name
				return req.Preferences.ConnectionID, "default", nil
			}
			return req.Preferences.ConnectionID, models[0].ID, nil
		}
	}
	
	// 1. Analyze requirements from the request
	requirements := r.analyzeRequirements(req)

	// 2. Get available providers and their models
	candidates := r.findCapableProviders(requirements)
	if len(candidates) == 0 {
		// Try to get ANY available provider as a last resort
		allProviders := r.providers.GetAll()
		fmt.Printf("[RequestRouter.RouteRequest] No capable providers found, checking all %d providers\n", len(allProviders))
		
		for providerID, provider := range allProviders {
			fmt.Printf("[RequestRouter.RouteRequest] Trying provider: %s\n", providerID)
			// Just use the first available provider
			models, err := provider.GetModels(context.Background())
			if err == nil && len(models) > 0 {
				return providerID, models[0].ID, nil
			}
		}
		
		return "", "", fmt.Errorf("no providers available for requirements: %+v", requirements)
	}

	// 3. Apply user preferences
	selected := r.applyPreferences(candidates, req.Preferences)

	// Check if we got a valid selection
	if selected.Provider == "" {
		return "", "", fmt.Errorf("no provider could be selected")
	}

	// 4. Check health status
	if r.healthMonitor != nil && selected.Provider != "" && !r.healthMonitor.IsHealthy(selected.Provider) {
		// Try to find a healthy alternative
		for _, candidate := range candidates {
			if r.healthMonitor.IsHealthy(candidate.Provider) {
				selected = candidate
				break
			}
		}
	}

	return selected.Provider, selected.Model, nil
}

// RoutingCandidate represents a potential provider/model combination
type RoutingCandidate struct {
	Provider     string
	Model        string
	Capabilities providers.ModelCapabilities
	Score        float64
}

// analyzeRequirements extracts requirements from the request
func (r *RequestRouter) analyzeRequirements(req models.UnifiedChatRequest) providers.Requirements {
	requirements := req.Requirements

	// Auto-detect requirements from request content
	if len(req.Functions) > 0 || len(req.Tools) > 0 {
		requirements.NeedsFunctions = true
	}

	if len(req.Images) > 0 {
		requirements.NeedsVision = true
	}

	// Calculate minimum context size needed
	contextSize := 0
	for _, msg := range req.Messages {
		contextSize += len(msg.Content) / 4 // Rough token estimate
	}
	if contextSize > requirements.MinContextSize {
		requirements.MinContextSize = contextSize
	}

	// Set output format
	if req.ResponseFormat != "" {
		requirements.OutputFormat = req.ResponseFormat
	}

	return requirements
}

// findCapableProviders returns providers that can handle the requirements
func (r *RequestRouter) findCapableProviders(requirements providers.Requirements) []RoutingCandidate {
	var candidates []RoutingCandidate

	allProviders := r.providers.GetAll()
	for providerID, provider := range allProviders {
		// Skip unhealthy providers (if health monitor is available)
		if r.healthMonitor != nil && !r.healthMonitor.IsHealthy(providerID) {
			continue
		}

		// Check each model's capabilities
		models, err := provider.GetModels(context.Background())
		if err != nil {
			continue
		}

		for _, model := range models {
			modelCaps := providers.GetCapabilitiesForModel(model.ID)
			if modelCaps.Capabilities.MeetsRequirements(requirements) {
				candidates = append(candidates, RoutingCandidate{
					Provider:     providerID,
					Model:        model.ID,
					Capabilities: *modelCaps,
					Score:        0,
				})
			}
		}
	}

	return candidates
}

// applyPreferences scores and selects based on user preferences
func (r *RequestRouter) applyPreferences(candidates []RoutingCandidate, prefs models.Preferences) RoutingCandidate {
	// Check for empty candidates
	if len(candidates) == 0 {
		return RoutingCandidate{} // Return empty candidate
	}

	// Direct provider/model override
	if prefs.Provider != "" && prefs.Model != "" {
		for _, c := range candidates {
			if c.Provider == prefs.Provider && c.Model == prefs.Model {
				return c
			}
		}
	}

	// Score each candidate
	for i := range candidates {
		candidates[i].Score = r.scoreCandidate(candidates[i], prefs)
	}

	// Sort by score (highest first)
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].Score > candidates[j].Score
	})

	return candidates[0]
}

// scoreCandidate calculates a score based on preferences
func (r *RequestRouter) scoreCandidate(candidate RoutingCandidate, prefs models.Preferences) float64 {
	score := 100.0

	// Speed preference
	switch prefs.Speed {
	case "fast":
		if strings.Contains(strings.ToLower(candidate.Model), "turbo") ||
		   strings.Contains(strings.ToLower(candidate.Model), "haiku") {
			score += 20
		}
	case "quality":
		if strings.Contains(strings.ToLower(candidate.Model), "gpt-4") ||
		   strings.Contains(strings.ToLower(candidate.Model), "opus") {
			score += 20
		}
	}

	// Cost preference
	switch prefs.Cost {
	case "economy":
		if candidate.Capabilities.PricingTier == "economy" || 
		   candidate.Capabilities.PricingTier == "free" {
			score += 30
		}
	case "premium":
		if candidate.Capabilities.PricingTier == "premium" {
			score += 10
		}
	}

	// Privacy preference
	switch prefs.Privacy {
	case "local":
		if candidate.Provider == "ollama" || candidate.Provider == "lm-studio" {
			score += 50
		}
	case "cloud":
		if candidate.Provider == "openai" || candidate.Provider == "anthropic" {
			score += 10
		}
	}

	// Provider preference (partial match)
	if prefs.Provider != "" && candidate.Provider == prefs.Provider {
		score += 25
	}

	// Model preference (partial match)
	if prefs.Model != "" && strings.Contains(candidate.Model, prefs.Model) {
		score += 15
	}

	// Health status bonus - check for nil health monitor
	if r.healthMonitor != nil {
		health := r.healthMonitor.GetHealth(candidate.Provider)
		if health != nil {
			score += float64(100 - health.ErrorRate*100) * 0.1 // Up to 10 points for reliability
		}
	}

	return score
}

// GetRoutingMetadata returns metadata about the routing decision
func (r *RequestRouter) GetRoutingMetadata(provider, model string, prefs models.Preferences) models.ResponseMetadata {
	reason := "Selected based on "
	reasons := []string{}

	if prefs.Provider != "" && prefs.Model != "" {
		reasons = append(reasons, "explicit selection")
	} else {
		if prefs.Speed != "" {
			reasons = append(reasons, fmt.Sprintf("%s speed", prefs.Speed))
		}
		if prefs.Cost != "" {
			reasons = append(reasons, fmt.Sprintf("%s cost", prefs.Cost))
		}
		if prefs.Privacy != "" {
			reasons = append(reasons, fmt.Sprintf("%s privacy", prefs.Privacy))
		}
	}

	if len(reasons) == 0 {
		reason = "Default routing"
	} else {
		reason += strings.Join(reasons, ", ")
	}

	return models.ResponseMetadata{
		Provider:      provider,
		Model:         model,
		RoutingReason: reason,
	}
}