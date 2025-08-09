package llm

import (
	"fmt"
	"sync"
	"time"
)

// CircuitBreaker implements the circuit breaker pattern
type CircuitBreaker struct {
	breakers map[string]*Breaker
	mu       sync.RWMutex
}

// Breaker represents a single circuit breaker
type Breaker struct {
	failureThreshold uint32
	successThreshold uint32
	timeout          time.Duration
	
	failures    uint32
	successes   uint32
	lastFailure time.Time
	state       BreakerState
	mu          sync.RWMutex
}

// BreakerState represents the circuit breaker state
type BreakerState int

const (
	StateClosed BreakerState = iota
	StateOpen
	StateHalfOpen
)

// NewCircuitBreaker creates a new circuit breaker
func NewCircuitBreaker() *CircuitBreaker {
	return &CircuitBreaker{
		breakers: make(map[string]*Breaker),
	}
}

// Execute executes a function with circuit breaker protection
func (cb *CircuitBreaker) Execute(key string, fn func() error) error {
	breaker := cb.getOrCreateBreaker(key)
	
	// Check breaker state
	state := breaker.getState()
	if state == StateOpen {
		return fmt.Errorf("circuit breaker is open for %s", key)
	}
	
	// Execute function
	err := fn()
	
	// Update breaker state based on result
	if err != nil {
		breaker.recordFailure()
	} else {
		breaker.recordSuccess()
	}
	
	return err
}

// getOrCreateBreaker gets or creates a breaker for a key
func (cb *CircuitBreaker) getOrCreateBreaker(key string) *Breaker {
	cb.mu.RLock()
	breaker, exists := cb.breakers[key]
	cb.mu.RUnlock()
	
	if exists {
		return breaker
	}
	
	cb.mu.Lock()
	defer cb.mu.Unlock()
	
	// Double-check after acquiring write lock
	if breaker, exists := cb.breakers[key]; exists {
		return breaker
	}
	
	// Create new breaker with default settings
	breaker = &Breaker{
		failureThreshold: 5,
		successThreshold: 2,
		timeout:          30 * time.Second,
		state:            StateClosed,
	}
	
	cb.breakers[key] = breaker
	return breaker
}

// getState returns the current state of the breaker
func (b *Breaker) getState() BreakerState {
	b.mu.RLock()
	defer b.mu.RUnlock()
	
	// Check if we should transition from Open to HalfOpen
	if b.state == StateOpen {
		if time.Since(b.lastFailure) > b.timeout {
			b.mu.RUnlock()
			b.mu.Lock()
			b.state = StateHalfOpen
			b.failures = 0
			b.successes = 0
			b.mu.Unlock()
			b.mu.RLock()
		}
	}
	
	return b.state
}

// recordFailure records a failure
func (b *Breaker) recordFailure() {
	b.mu.Lock()
	defer b.mu.Unlock()
	
	b.failures++
	b.lastFailure = time.Now()
	
	switch b.state {
	case StateClosed:
		if b.failures >= b.failureThreshold {
			b.state = StateOpen
			fmt.Printf("[CircuitBreaker] Opening circuit breaker after %d failures\n", b.failures)
		}
	case StateHalfOpen:
		b.state = StateOpen
		fmt.Printf("[CircuitBreaker] Re-opening circuit breaker after failure in half-open state\n")
	}
}

// recordSuccess records a success
func (b *Breaker) recordSuccess() {
	b.mu.Lock()
	defer b.mu.Unlock()
	
	b.successes++
	
	switch b.state {
	case StateClosed:
		b.failures = 0 // Reset failure count on success
	case StateHalfOpen:
		if b.successes >= b.successThreshold {
			b.state = StateClosed
			b.failures = 0
			b.successes = 0
			fmt.Printf("[CircuitBreaker] Closing circuit breaker after %d successes\n", b.successes)
		}
	}
}

// GetState returns the state of a specific breaker
func (cb *CircuitBreaker) GetState(key string) BreakerState {
	cb.mu.RLock()
	breaker, exists := cb.breakers[key]
	cb.mu.RUnlock()
	
	if !exists {
		return StateClosed
	}
	
	return breaker.getState()
}

// Reset resets a specific breaker
func (cb *CircuitBreaker) Reset(key string) {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	
	if breaker, exists := cb.breakers[key]; exists {
		breaker.mu.Lock()
		breaker.state = StateClosed
		breaker.failures = 0
		breaker.successes = 0
		breaker.mu.Unlock()
	}
}

// ResetAll resets all breakers
func (cb *CircuitBreaker) ResetAll() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	
	for _, breaker := range cb.breakers {
		breaker.mu.Lock()
		breaker.state = StateClosed
		breaker.failures = 0
		breaker.successes = 0
		breaker.mu.Unlock()
	}
}