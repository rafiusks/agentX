package llm

import (
	"fmt"
	"sync"
	"time"
)

// MetricsCollector collects metrics for LLM operations
type MetricsCollector struct {
	requests   map[string]int64
	tokens     map[string]int64
	costs      map[string]float64
	latencies  map[string][]time.Duration
	errors     map[string]int64
	mu         sync.RWMutex
}

// NewMetricsCollector creates a new metrics collector
func NewMetricsCollector() *MetricsCollector {
	return &MetricsCollector{
		requests:  make(map[string]int64),
		tokens:    make(map[string]int64),
		costs:     make(map[string]float64),
		latencies: make(map[string][]time.Duration),
		errors:    make(map[string]int64),
	}
}

// RecordRequest records a request
func (mc *MetricsCollector) RecordRequest(provider, model string, success bool, latency time.Duration) {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	
	key := provider + ":" + model
	mc.requests[key]++
	
	if !success {
		mc.errors[key]++
	}
	
	mc.latencies[key] = append(mc.latencies[key], latency)
	
	// Keep only last 100 latencies
	if len(mc.latencies[key]) > 100 {
		mc.latencies[key] = mc.latencies[key][1:]
	}
}

// RecordTokens records token usage
func (mc *MetricsCollector) RecordTokens(provider string, tokens int) {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	
	mc.tokens[provider] += int64(tokens)
}

// RecordCost records cost
func (mc *MetricsCollector) RecordCost(provider string, cost float64) {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	
	mc.costs[provider] += cost
}

// RecordLatency records latency
func (mc *MetricsCollector) RecordLatency(provider string, latency time.Duration) {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	
	mc.latencies[provider] = append(mc.latencies[provider], latency)
	
	// Keep only last 100 latencies
	if len(mc.latencies[provider]) > 100 {
		mc.latencies[provider] = mc.latencies[provider][1:]
	}
}

// RecordUsage records usage statistics
func (mc *MetricsCollector) RecordUsage(provider string, usage Usage) {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	
	mc.tokens[provider] += int64(usage.TotalTokens)
	if usage.EstimatedCost > 0 {
		mc.costs[provider] += usage.EstimatedCost
	}
}

// RecordChunk records a streaming chunk
func (mc *MetricsCollector) RecordChunk(provider string, chunk *StreamChunk) {
	if chunk.Usage != nil {
		mc.RecordUsage(provider, *chunk.Usage)
	}
}

// GetSnapshot returns a snapshot of current metrics
func (mc *MetricsCollector) GetSnapshot() map[string]interface{} {
	mc.mu.RLock()
	defer mc.mu.RUnlock()
	
	snapshot := make(map[string]interface{})
	
	// Copy request counts
	requests := make(map[string]int64)
	for k, v := range mc.requests {
		requests[k] = v
	}
	snapshot["requests"] = requests
	
	// Copy token counts
	tokens := make(map[string]int64)
	for k, v := range mc.tokens {
		tokens[k] = v
	}
	snapshot["tokens"] = tokens
	
	// Copy costs
	costs := make(map[string]float64)
	for k, v := range mc.costs {
		costs[k] = v
	}
	snapshot["costs"] = costs
	
	// Calculate average latencies
	avgLatencies := make(map[string]float64)
	for k, latencies := range mc.latencies {
		if len(latencies) > 0 {
			var total time.Duration
			for _, l := range latencies {
				total += l
			}
			avgLatencies[k] = float64(total.Milliseconds()) / float64(len(latencies))
		}
	}
	snapshot["avg_latency_ms"] = avgLatencies
	
	// Copy error counts
	errors := make(map[string]int64)
	for k, v := range mc.errors {
		errors[k] = v
	}
	snapshot["errors"] = errors
	
	return snapshot
}

// Reset resets all metrics
func (mc *MetricsCollector) Reset() {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	
	mc.requests = make(map[string]int64)
	mc.tokens = make(map[string]int64)
	mc.costs = make(map[string]float64)
	mc.latencies = make(map[string][]time.Duration)
	mc.errors = make(map[string]int64)
}

// Flush flushes metrics (for persistent storage)
func (mc *MetricsCollector) Flush() {
	// This would typically write to a database or metrics service
	snapshot := mc.GetSnapshot()
	// For now, just log
	fmt.Printf("[Metrics] Flushing metrics: %+v\n", snapshot)
}