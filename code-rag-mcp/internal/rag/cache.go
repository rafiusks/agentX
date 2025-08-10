package rag

import (
	"crypto/sha256"
	"encoding/hex"
	"sync"
	"time"
)

// CacheEntry represents a cached search result
type CacheEntry struct {
	Results   []SearchResult
	Timestamp time.Time
	HitCount  int
}

// SearchCache implements an LRU cache for search results
type SearchCache struct {
	mu         sync.RWMutex
	entries    map[string]*CacheEntry
	maxEntries int
	ttl        time.Duration
	// Track access order for LRU eviction
	accessOrder []string
}

// NewSearchCache creates a new search cache
func NewSearchCache(maxEntries int, ttl time.Duration) *SearchCache {
	if maxEntries <= 0 {
		maxEntries = 1000 // Default to 1000 entries
	}
	if ttl <= 0 {
		ttl = 15 * time.Minute // Default to 15 minutes
	}
	
	return &SearchCache{
		entries:     make(map[string]*CacheEntry),
		maxEntries:  maxEntries,
		ttl:         ttl,
		accessOrder: make([]string, 0, maxEntries),
	}
}

// generateCacheKey creates a unique key for a search query
func (sc *SearchCache) generateCacheKey(query string, collection string, limit int) string {
	data := query + "|" + collection + "|" + string(rune(limit))
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}

// Get retrieves cached results if available and not expired
func (sc *SearchCache) Get(query string, collection string, limit int) ([]SearchResult, bool) {
	sc.mu.RLock()
	defer sc.mu.RUnlock()
	
	key := sc.generateCacheKey(query, collection, limit)
	entry, exists := sc.entries[key]
	
	if !exists {
		return nil, false
	}
	
	// Check if entry has expired
	if time.Since(entry.Timestamp) > sc.ttl {
		// Don't return expired entries
		return nil, false
	}
	
	// Update hit count
	entry.HitCount++
	
	// Move to end of access order (most recently used)
	sc.updateAccessOrder(key)
	
	// Return a copy to prevent external modifications
	resultsCopy := make([]SearchResult, len(entry.Results))
	copy(resultsCopy, entry.Results)
	
	return resultsCopy, true
}

// Set stores search results in the cache
func (sc *SearchCache) Set(query string, collection string, limit int, results []SearchResult) {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	
	key := sc.generateCacheKey(query, collection, limit)
	
	// Check if we need to evict entries
	if len(sc.entries) >= sc.maxEntries {
		sc.evictLRU()
	}
	
	// Store a copy to prevent external modifications
	resultsCopy := make([]SearchResult, len(results))
	copy(resultsCopy, results)
	
	sc.entries[key] = &CacheEntry{
		Results:   resultsCopy,
		Timestamp: time.Now(),
		HitCount:  0,
	}
	
	// Add to access order
	sc.accessOrder = append(sc.accessOrder, key)
}

// updateAccessOrder moves a key to the end (most recently used)
func (sc *SearchCache) updateAccessOrder(key string) {
	// Find and remove the key
	for i, k := range sc.accessOrder {
		if k == key {
			sc.accessOrder = append(sc.accessOrder[:i], sc.accessOrder[i+1:]...)
			break
		}
	}
	// Add to end
	sc.accessOrder = append(sc.accessOrder, key)
}

// evictLRU removes the least recently used entry
func (sc *SearchCache) evictLRU() {
	if len(sc.accessOrder) == 0 {
		return
	}
	
	// Remove the first entry (least recently used)
	oldestKey := sc.accessOrder[0]
	delete(sc.entries, oldestKey)
	sc.accessOrder = sc.accessOrder[1:]
}

// Clear removes all entries from the cache
func (sc *SearchCache) Clear() {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	
	sc.entries = make(map[string]*CacheEntry)
	sc.accessOrder = make([]string, 0, sc.maxEntries)
}

// Stats returns cache statistics
func (sc *SearchCache) Stats() map[string]interface{} {
	sc.mu.RLock()
	defer sc.mu.RUnlock()
	
	totalHits := 0
	for _, entry := range sc.entries {
		totalHits += entry.HitCount
	}
	
	return map[string]interface{}{
		"entries":    len(sc.entries),
		"max_entries": sc.maxEntries,
		"total_hits": totalHits,
		"ttl_seconds": sc.ttl.Seconds(),
	}
}

// EmbeddingCache caches embeddings for code chunks
type EmbeddingCache struct {
	mu         sync.RWMutex
	embeddings map[string][]float32
	maxSize    int
	accessTime map[string]time.Time
}

// NewEmbeddingCache creates a new embedding cache
func NewEmbeddingCache(maxSize int) *EmbeddingCache {
	if maxSize <= 0 {
		maxSize = 10000 // Default to 10k embeddings
	}
	
	return &EmbeddingCache{
		embeddings: make(map[string][]float32),
		maxSize:    maxSize,
		accessTime: make(map[string]time.Time),
	}
}

// Get retrieves a cached embedding
func (ec *EmbeddingCache) Get(text string) ([]float32, bool) {
	ec.mu.RLock()
	defer ec.mu.RUnlock()
	
	key := ec.generateKey(text)
	embedding, exists := ec.embeddings[key]
	
	if exists {
		ec.accessTime[key] = time.Now()
	}
	
	return embedding, exists
}

// Set stores an embedding in the cache
func (ec *EmbeddingCache) Set(text string, embedding []float32) {
	ec.mu.Lock()
	defer ec.mu.Unlock()
	
	key := ec.generateKey(text)
	
	// Check if we need to evict
	if len(ec.embeddings) >= ec.maxSize {
		ec.evictOldest()
	}
	
	// Store a copy
	embeddingCopy := make([]float32, len(embedding))
	copy(embeddingCopy, embedding)
	
	ec.embeddings[key] = embeddingCopy
	ec.accessTime[key] = time.Now()
}

// generateKey creates a cache key for text
func (ec *EmbeddingCache) generateKey(text string) string {
	// Use first 100 chars + hash for very long text
	if len(text) > 100 {
		hash := sha256.Sum256([]byte(text))
		return text[:100] + hex.EncodeToString(hash[:8])
	}
	return text
}

// evictOldest removes the least recently accessed embedding
func (ec *EmbeddingCache) evictOldest() {
	var oldestKey string
	var oldestTime time.Time
	
	for key, accessTime := range ec.accessTime {
		if oldestKey == "" || accessTime.Before(oldestTime) {
			oldestKey = key
			oldestTime = accessTime
		}
	}
	
	if oldestKey != "" {
		delete(ec.embeddings, oldestKey)
		delete(ec.accessTime, oldestKey)
	}
}

// Clear removes all cached embeddings
func (ec *EmbeddingCache) Clear() {
	ec.mu.Lock()
	defer ec.mu.Unlock()
	
	ec.embeddings = make(map[string][]float32)
	ec.accessTime = make(map[string]time.Time)
}

// FileChunkCache caches parsed chunks for files
type FileChunkCache struct {
	mu      sync.RWMutex
	chunks  map[string][]Chunk
	modTime map[string]time.Time
	maxSize int
}

// NewFileChunkCache creates a new file chunk cache
func NewFileChunkCache(maxSize int) *FileChunkCache {
	if maxSize <= 0 {
		maxSize = 5000 // Default to 5k files
	}
	
	return &FileChunkCache{
		chunks:  make(map[string][]Chunk),
		modTime: make(map[string]time.Time),
		maxSize: maxSize,
	}
}

// Get retrieves cached chunks for a file if not modified
func (fc *FileChunkCache) Get(filePath string, currentModTime time.Time) ([]Chunk, bool) {
	fc.mu.RLock()
	defer fc.mu.RUnlock()
	
	chunks, exists := fc.chunks[filePath]
	if !exists {
		return nil, false
	}
	
	// Check if file has been modified
	cachedModTime, hasModTime := fc.modTime[filePath]
	if !hasModTime || !cachedModTime.Equal(currentModTime) {
		return nil, false
	}
	
	return chunks, true
}

// Set stores chunks for a file
func (fc *FileChunkCache) Set(filePath string, chunks []Chunk, modTime time.Time) {
	fc.mu.Lock()
	defer fc.mu.Unlock()
	
	// Check if we need to evict
	if len(fc.chunks) >= fc.maxSize {
		// Simple eviction: remove first entry
		for path := range fc.chunks {
			delete(fc.chunks, path)
			delete(fc.modTime, path)
			break
		}
	}
	
	fc.chunks[filePath] = chunks
	fc.modTime[filePath] = modTime
}

// Clear removes all cached chunks
func (fc *FileChunkCache) Clear() {
	fc.mu.Lock()
	defer fc.mu.Unlock()
	
	fc.chunks = make(map[string][]Chunk)
	fc.modTime = make(map[string]time.Time)
}