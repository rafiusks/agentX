package llm

import (
	"context"
	"fmt"
	"time"
)

// Middleware interface for request/response processing
type Middleware interface {
	// PreProcess is called before the request is sent to the provider
	PreProcess(ctx context.Context, req *Request) (context.Context, *Request, error)
	
	// PostProcess is called after the response is received from the provider
	PostProcess(ctx context.Context, req *Request, resp *Response, err error) (*Response, error)
}

// BaseMiddleware provides a default implementation
type BaseMiddleware struct{}

func (m *BaseMiddleware) PreProcess(ctx context.Context, req *Request) (context.Context, *Request, error) {
	return ctx, req, nil
}

func (m *BaseMiddleware) PostProcess(ctx context.Context, req *Request, resp *Response, err error) (*Response, error) {
	return resp, err
}

// LoggingMiddleware logs requests and responses
type LoggingMiddleware struct {
	BaseMiddleware
}

func NewLoggingMiddleware() Middleware {
	return &LoggingMiddleware{}
}

func (m *LoggingMiddleware) PreProcess(ctx context.Context, req *Request) (context.Context, *Request, error) {
	fmt.Printf("[LLM Request] user=%s, session=%s, model=%s, messages=%d\n",
		req.UserID, req.SessionID, req.Model, len(req.Messages))
	return ctx, req, nil
}

func (m *LoggingMiddleware) PostProcess(ctx context.Context, req *Request, resp *Response, err error) (*Response, error) {
	if err != nil {
		fmt.Printf("[LLM Response Error] user=%s, error=%v\n", req.UserID, err)
	} else if resp != nil {
		fmt.Printf("[LLM Response] user=%s, tokens=%d, latency=%dms\n",
			req.UserID, resp.Usage.TotalTokens, resp.Metadata.LatencyMs)
	}
	return resp, err
}

// ValidationMiddleware validates requests
type ValidationMiddleware struct {
	BaseMiddleware
}

func NewValidationMiddleware() Middleware {
	return &ValidationMiddleware{}
}

func (m *ValidationMiddleware) PreProcess(ctx context.Context, req *Request) (context.Context, *Request, error) {
	// Validate request
	if err := req.Validate(); err != nil {
		return ctx, req, fmt.Errorf("validation failed: %w", err)
	}
	
	// Ensure metadata is initialized
	if req.Metadata == nil {
		req.Metadata = make(map[string]interface{})
	}
	
	return ctx, req, nil
}

// MetricsMiddleware collects metrics
type MetricsMiddleware struct {
	BaseMiddleware
	collector *MetricsCollector
}

func NewMetricsMiddleware(collector *MetricsCollector) Middleware {
	return &MetricsMiddleware{collector: collector}
}

func (m *MetricsMiddleware) PreProcess(ctx context.Context, req *Request) (context.Context, *Request, error) {
	// Add start time to context
	ctx = context.WithValue(ctx, "start_time", time.Now())
	return ctx, req, nil
}

func (m *MetricsMiddleware) PostProcess(ctx context.Context, req *Request, resp *Response, err error) (*Response, error) {
	// Get start time from context
	if startTime, ok := ctx.Value("start_time").(time.Time); ok {
		duration := time.Since(startTime)
		
		if m.collector != nil && resp != nil {
			m.collector.RecordLatency(resp.Metadata.Provider, duration)
			m.collector.RecordTokens(resp.Metadata.Provider, resp.Usage.TotalTokens)
		}
	}
	
	return resp, err
}

// RateLimitMiddleware enforces rate limits
type RateLimitMiddleware struct {
	BaseMiddleware
	limiter RateLimiter
}

func NewRateLimitMiddleware() Middleware {
	return &RateLimitMiddleware{
		limiter: NewTokenBucketLimiter(100, 100), // 100 requests per minute
	}
}

func (m *RateLimitMiddleware) PreProcess(ctx context.Context, req *Request) (context.Context, *Request, error) {
	// Check rate limit for user
	if !m.limiter.Allow(req.UserID) {
		return ctx, req, fmt.Errorf("rate limit exceeded for user %s", req.UserID)
	}
	return ctx, req, nil
}

// RetryMiddleware handles retries with exponential backoff
type RetryMiddleware struct {
	BaseMiddleware
	maxRetries int
	baseDelay  time.Duration
}

func NewRetryMiddleware() Middleware {
	return &RetryMiddleware{
		maxRetries: 3,
		baseDelay:  time.Second,
	}
}

func (m *RetryMiddleware) PostProcess(ctx context.Context, req *Request, resp *Response, err error) (*Response, error) {
	// Only retry on certain errors
	if err != nil && m.shouldRetry(err) {
		retries := 0
		if r, ok := req.Metadata["retries"].(int); ok {
			retries = r
		}
		
		if retries < m.maxRetries {
			req.Metadata["retries"] = retries + 1
			// Note: Actual retry logic would be handled at Gateway level
			// This just marks that a retry should be attempted
			return resp, fmt.Errorf("retry_needed: %w", err)
		}
	}
	
	return resp, err
}

func (m *RetryMiddleware) shouldRetry(err error) bool {
	// Check if error is retryable (network errors, rate limits, etc.)
	errStr := err.Error()
	return contains(errStr, "timeout") || 
	       contains(errStr, "connection") ||
	       contains(errStr, "rate limit")
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && s[:len(substr)] == substr
}