package llm

import (
	"sync"
	"time"
)

// RateLimiter interface for rate limiting
type RateLimiter interface {
	Allow(key string) bool
	Reset(key string)
}

// TokenBucketLimiter implements token bucket rate limiting
type TokenBucketLimiter struct {
	buckets  map[string]*TokenBucket
	rate     int           // tokens per interval
	capacity int           // max tokens
	interval time.Duration // refill interval
	mu       sync.RWMutex
}

// TokenBucket represents a single token bucket
type TokenBucket struct {
	tokens    int
	lastRefill time.Time
	mu        sync.Mutex
}

// NewTokenBucketLimiter creates a new token bucket limiter
func NewTokenBucketLimiter(rate, capacity int) *TokenBucketLimiter {
	return &TokenBucketLimiter{
		buckets:  make(map[string]*TokenBucket),
		rate:     rate,
		capacity: capacity,
		interval: time.Minute, // tokens per minute
	}
}

// Allow checks if a request is allowed
func (l *TokenBucketLimiter) Allow(key string) bool {
	bucket := l.getOrCreateBucket(key)
	
	bucket.mu.Lock()
	defer bucket.mu.Unlock()
	
	// Refill tokens
	now := time.Now()
	elapsed := now.Sub(bucket.lastRefill)
	tokensToAdd := int(elapsed / l.interval) * l.rate
	
	if tokensToAdd > 0 {
		bucket.tokens = min(bucket.tokens+tokensToAdd, l.capacity)
		bucket.lastRefill = now
	}
	
	// Check if we have tokens
	if bucket.tokens > 0 {
		bucket.tokens--
		return true
	}
	
	return false
}

// Reset resets the rate limit for a key
func (l *TokenBucketLimiter) Reset(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	
	delete(l.buckets, key)
}

// getOrCreateBucket gets or creates a bucket for a key
func (l *TokenBucketLimiter) getOrCreateBucket(key string) *TokenBucket {
	l.mu.RLock()
	bucket, exists := l.buckets[key]
	l.mu.RUnlock()
	
	if exists {
		return bucket
	}
	
	l.mu.Lock()
	defer l.mu.Unlock()
	
	// Double-check after acquiring write lock
	if bucket, exists := l.buckets[key]; exists {
		return bucket
	}
	
	// Create new bucket
	bucket = &TokenBucket{
		tokens:     l.capacity,
		lastRefill: time.Now(),
	}
	
	l.buckets[key] = bucket
	return bucket
}

// SlidingWindowLimiter implements sliding window rate limiting
type SlidingWindowLimiter struct {
	windows  map[string]*SlidingWindow
	limit    int
	window   time.Duration
	mu       sync.RWMutex
}

// SlidingWindow represents a sliding window
type SlidingWindow struct {
	requests []time.Time
	mu       sync.Mutex
}

// NewSlidingWindowLimiter creates a new sliding window limiter
func NewSlidingWindowLimiter(limit int, window time.Duration) *SlidingWindowLimiter {
	return &SlidingWindowLimiter{
		windows: make(map[string]*SlidingWindow),
		limit:   limit,
		window:  window,
	}
}

// Allow checks if a request is allowed
func (l *SlidingWindowLimiter) Allow(key string) bool {
	window := l.getOrCreateWindow(key)
	
	window.mu.Lock()
	defer window.mu.Unlock()
	
	now := time.Now()
	cutoff := now.Add(-l.window)
	
	// Remove old requests
	validRequests := []time.Time{}
	for _, t := range window.requests {
		if t.After(cutoff) {
			validRequests = append(validRequests, t)
		}
	}
	window.requests = validRequests
	
	// Check if we're under the limit
	if len(window.requests) < l.limit {
		window.requests = append(window.requests, now)
		return true
	}
	
	return false
}

// Reset resets the rate limit for a key
func (l *SlidingWindowLimiter) Reset(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	
	delete(l.windows, key)
}

// getOrCreateWindow gets or creates a window for a key
func (l *SlidingWindowLimiter) getOrCreateWindow(key string) *SlidingWindow {
	l.mu.RLock()
	window, exists := l.windows[key]
	l.mu.RUnlock()
	
	if exists {
		return window
	}
	
	l.mu.Lock()
	defer l.mu.Unlock()
	
	// Double-check after acquiring write lock
	if window, exists := l.windows[key]; exists {
		return window
	}
	
	// Create new window
	window = &SlidingWindow{
		requests: []time.Time{},
	}
	
	l.windows[key] = window
	return window
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}