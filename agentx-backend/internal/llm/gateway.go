package llm

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// Gateway is the centralized service for all LLM interactions
type Gateway struct {
	providers      *ProviderManager
	config         *ConfigManager
	router         *Router
	middleware     []Middleware
	circuitBreaker *CircuitBreaker
	metrics        *MetricsCollector
	mu             sync.RWMutex
}

// GatewayOption is a functional option for configuring the Gateway
type GatewayOption func(*Gateway)

// WithMiddleware adds middleware to the gateway
func WithMiddleware(mw ...Middleware) GatewayOption {
	return func(g *Gateway) {
		g.middleware = append(g.middleware, mw...)
	}
}

// WithCircuitBreaker configures the circuit breaker
func WithCircuitBreaker(cb *CircuitBreaker) GatewayOption {
	return func(g *Gateway) {
		g.circuitBreaker = cb
	}
}

// WithMetrics configures metrics collection
func WithMetrics(m *MetricsCollector) GatewayOption {
	return func(g *Gateway) {
		g.metrics = m
	}
}

// NewGateway creates a new LLM Gateway
func NewGateway(opts ...GatewayOption) *Gateway {
	g := &Gateway{
		providers:      NewProviderManager(),
		config:         NewConfigManager(),
		router:         NewRouter(),
		middleware:     []Middleware{},
		circuitBreaker: NewCircuitBreaker(),
		metrics:        NewMetricsCollector(),
	}

	// Apply options
	for _, opt := range opts {
		opt(g)
	}

	// Setup default middleware pipeline if none provided
	if len(g.middleware) == 0 {
		g.setupDefaultMiddleware()
	}

	return g
}

// setupDefaultMiddleware configures the default middleware pipeline
func (g *Gateway) setupDefaultMiddleware() {
	g.middleware = []Middleware{
		NewLoggingMiddleware(),
		NewMetricsMiddleware(g.metrics),
		NewValidationMiddleware(),
		NewRateLimitMiddleware(),
		NewRetryMiddleware(),
	}
}

// Complete handles non-streaming completions
func (g *Gateway) Complete(ctx context.Context, req *Request) (*Response, error) {
	startTime := time.Now()

	// Validate request
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	// Apply middleware pre-processing
	for _, mw := range g.middleware {
		var err error
		ctx, req, err = mw.PreProcess(ctx, req)
		if err != nil {
			return nil, fmt.Errorf("middleware pre-process: %w", err)
		}
	}

	// Route to provider
	provider, routeInfo, err := g.router.Route(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("routing failed: %w", err)
	}

	// Log routing decision
	fmt.Printf("[Gateway] Routed request to provider=%s, model=%s, connection=%s\n",
		routeInfo.Provider, routeInfo.Model, routeInfo.ConnectionID)

	// Execute with circuit breaker
	var resp *Response
	cbKey := fmt.Sprintf("%s:%s", routeInfo.Provider, routeInfo.Model)
	
	err = g.circuitBreaker.Execute(cbKey, func() error {
		var execErr error
		resp, execErr = provider.Complete(ctx, req)
		return execErr
	})

	// Handle circuit breaker errors and try fallback
	if err != nil {
		if fallbackProvider := g.router.GetFallback(routeInfo.Provider); fallbackProvider != nil {
			fmt.Printf("[Gateway] Primary provider failed, trying fallback\n")
			resp, err = fallbackProvider.Complete(ctx, req)
			if err == nil {
				resp.Metadata.FallbackUsed = true
			}
		}
	}

	// Apply middleware post-processing
	for i := len(g.middleware) - 1; i >= 0; i-- {
		resp, err = g.middleware[i].PostProcess(ctx, req, resp, err)
	}

	// Add metadata and populate convenience fields
	if resp != nil {
		resp.Metadata.Provider = routeInfo.Provider
		resp.Metadata.Model = routeInfo.Model
		resp.Metadata.ConnectionID = routeInfo.ConnectionID
		resp.Metadata.LatencyMs = time.Since(startTime).Milliseconds()
		
		// Populate convenience fields for direct access
		resp.Content = resp.GetContent()
		resp.Role = resp.GetRole()
		resp.Provider = routeInfo.Provider
		
		// Debug: Log response structure
		fmt.Printf("[Gateway] Response debug - Content: '%s', Choices count: %d\n", resp.Content, len(resp.Choices))
		if len(resp.Choices) > 0 {
			fmt.Printf("[Gateway] First choice content: '%s'\n", resp.Choices[0].Message.Content)
		}
	}

	// Record metrics
	if g.metrics != nil {
		g.metrics.RecordRequest(routeInfo.Provider, routeInfo.Model, err == nil, time.Since(startTime))
		if resp != nil {
			g.metrics.RecordUsage(routeInfo.Provider, resp.Usage)
		}
	}

	return resp, err
}

// StreamComplete handles streaming completions
func (g *Gateway) StreamComplete(ctx context.Context, req *Request) (<-chan *StreamChunk, error) {
	// Validate request
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	// Apply middleware pre-processing
	for _, mw := range g.middleware {
		var err error
		ctx, req, err = mw.PreProcess(ctx, req)
		if err != nil {
			return nil, fmt.Errorf("middleware pre-process: %w", err)
		}
	}

	// Route to provider
	provider, routeInfo, err := g.router.Route(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("routing failed: %w", err)
	}

	// Log routing decision
	fmt.Printf("[Gateway] Streaming request routed to provider=%s, model=%s, connection=%s\n",
		routeInfo.Provider, routeInfo.Model, routeInfo.ConnectionID)

	// Get stream from provider
	providerStream, err := provider.StreamComplete(ctx, req)
	if err != nil {
		// Try fallback
		if fallbackProvider := g.router.GetFallback(routeInfo.Provider); fallbackProvider != nil {
			fmt.Printf("[Gateway] Primary provider stream failed, trying fallback\n")
			providerStream, err = fallbackProvider.StreamComplete(ctx, req)
		}
		if err != nil {
			return nil, err
		}
	}

	// Create output channel
	out := make(chan *StreamChunk)

	// Process stream in goroutine
	go func() {
		defer close(out)
		startTime := time.Now()
		var totalTokens int

		for chunk := range providerStream {
			// Add metadata to chunk
			chunk.Model = routeInfo.Model
			
			// Track metrics
			if g.metrics != nil && chunk.Usage != nil {
				totalTokens += chunk.Usage.TotalTokens
			}

			// Send chunk
			select {
			case out <- chunk:
			case <-ctx.Done():
				return
			}
		}

		// Record final metrics
		if g.metrics != nil {
			g.metrics.RecordRequest(routeInfo.Provider, routeInfo.Model, true, time.Since(startTime))
		}
	}()

	return out, nil
}

// RegisterProvider registers a provider for a user connection
func (g *Gateway) RegisterProvider(userID, connectionID string, config ProviderConfig) error {
	return g.providers.RegisterProvider(userID, connectionID, config)
}

// RemoveProvider removes a provider registration
func (g *Gateway) RemoveProvider(userID, connectionID string) error {
	return g.providers.RemoveProvider(userID, connectionID)
}

// GetAvailableModels returns all available models
func (g *Gateway) GetAvailableModels(ctx context.Context, userID string) ([]ModelInfo, error) {
	return g.providers.GetAvailableModels(ctx, userID)
}

// HealthCheck performs a health check on all providers
func (g *Gateway) HealthCheck(ctx context.Context) map[string]HealthStatus {
	return g.providers.HealthCheck(ctx)
}

// GetMetrics returns current metrics
func (g *Gateway) GetMetrics() map[string]interface{} {
	if g.metrics != nil {
		return g.metrics.GetSnapshot()
	}
	return nil
}

// Shutdown gracefully shuts down the gateway
func (g *Gateway) Shutdown(ctx context.Context) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	// Shutdown providers
	if err := g.providers.Shutdown(ctx); err != nil {
		return fmt.Errorf("failed to shutdown providers: %w", err)
	}

	// Flush metrics
	if g.metrics != nil {
		g.metrics.Flush()
	}

	return nil
}