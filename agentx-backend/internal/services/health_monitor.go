package services

import (
	"context"
	"sync"
	"time"

	"github.com/agentx/agentx-backend/internal/providers"
)

// HealthStatus represents the health of a provider
type HealthStatus struct {
	Healthy       bool      `json:"healthy"`
	LastCheck     time.Time `json:"last_check"`
	LastError     string    `json:"last_error,omitempty"`
	ResponseTime  int64     `json:"response_time_ms"`
	ErrorCount    int       `json:"error_count"`
	SuccessCount  int       `json:"success_count"`
	ErrorRate     float64   `json:"error_rate"`
}

// HealthMonitor monitors provider health
type HealthMonitor struct {
	providers    *providers.Registry
	health       map[string]*HealthStatus
	mu           sync.RWMutex
	checkTicker  *time.Ticker
	stopChan     chan struct{}
}

// NewHealthMonitor creates a new health monitor
func NewHealthMonitor(providers *providers.Registry) *HealthMonitor {
	monitor := &HealthMonitor{
		providers: providers,
		health:    make(map[string]*HealthStatus),
		stopChan:  make(chan struct{}),
	}
	
	// Initialize health status for all providers
	for providerID := range providers.GetAll() {
		monitor.health[providerID] = &HealthStatus{
			Healthy:   true,
			LastCheck: time.Now(),
		}
	}
	
	// Start background health checks
	monitor.startHealthChecks()
	
	return monitor
}

// IsHealthy returns whether a provider is healthy
func (m *HealthMonitor) IsHealthy(providerID string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	status, exists := m.health[providerID]
	if !exists {
		return false
	}
	
	// Consider unhealthy if not checked recently
	if time.Since(status.LastCheck) > 5*time.Minute {
		return false
	}
	
	return status.Healthy
}

// GetHealth returns the health status for a provider
func (m *HealthMonitor) GetHealth(providerID string) *HealthStatus {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	if status, exists := m.health[providerID]; exists {
		// Return a copy to avoid race conditions
		statusCopy := *status
		return &statusCopy
	}
	
	return nil
}

// RecordSuccess records a successful request
func (m *HealthMonitor) RecordSuccess(providerID string, responseTime int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if status, exists := m.health[providerID]; exists {
		status.SuccessCount++
		status.ResponseTime = responseTime
		status.LastCheck = time.Now()
		status.Healthy = true
		status.LastError = ""
		m.updateErrorRate(status)
	}
}

// RecordError records a failed request
func (m *HealthMonitor) RecordError(providerID string, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if status, exists := m.health[providerID]; exists {
		status.ErrorCount++
		status.LastError = err.Error()
		status.LastCheck = time.Now()
		m.updateErrorRate(status)
		
		// Mark unhealthy if error rate is too high
		if status.ErrorRate > 0.5 && status.ErrorCount > 5 {
			status.Healthy = false
		}
	}
}

// updateErrorRate calculates the error rate
func (m *HealthMonitor) updateErrorRate(status *HealthStatus) {
	total := status.SuccessCount + status.ErrorCount
	if total > 0 {
		status.ErrorRate = float64(status.ErrorCount) / float64(total)
	}
}

// startHealthChecks starts background health monitoring
func (m *HealthMonitor) startHealthChecks() {
	m.checkTicker = time.NewTicker(1 * time.Minute)
	
	go func() {
		for {
			select {
			case <-m.checkTicker.C:
				m.performHealthChecks()
			case <-m.stopChan:
				return
			}
		}
	}()
}

// performHealthChecks checks all providers
func (m *HealthMonitor) performHealthChecks() {
	for providerID, provider := range m.providers.GetAll() {
		go m.checkProvider(providerID, provider)
	}
}

// checkProvider checks a single provider
func (m *HealthMonitor) checkProvider(providerID string, provider providers.Provider) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	
	start := time.Now()
	
	// Simple health check: try to get models
	_, err := provider.GetModels(ctx)
	
	responseTime := time.Since(start).Milliseconds()
	
	if err != nil {
		m.RecordError(providerID, err)
	} else {
		m.RecordSuccess(providerID, responseTime)
	}
}

// Stop stops the health monitor
func (m *HealthMonitor) Stop() {
	if m.checkTicker != nil {
		m.checkTicker.Stop()
	}
	close(m.stopChan)
}

// GetAllHealth returns health status for all providers
func (m *HealthMonitor) GetAllHealth() map[string]HealthStatus {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	// Create a copy of the health map
	healthCopy := make(map[string]HealthStatus)
	for k, v := range m.health {
		healthCopy[k] = *v
	}
	
	return healthCopy
}