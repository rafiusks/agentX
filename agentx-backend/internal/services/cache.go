package services

import (
	"sync"
	"time"
)

// CacheService provides caching capabilities
// Initially uses in-memory cache, will be enhanced with Redis support
type CacheService struct {
	mu    sync.RWMutex
	items map[string]*cacheItem
	
	// For future Redis integration
	// redis *redis.Client
}

type cacheItem struct {
	value      interface{}
	expiration time.Time
}

// NewCacheService creates a new cache service
func NewCacheService() *CacheService {
	cs := &CacheService{
		items: make(map[string]*cacheItem),
	}
	
	// Start cleanup goroutine
	go cs.cleanupExpired()
	
	return cs
}

// Get retrieves a value from cache
func (cs *CacheService) Get(key string) interface{} {
	cs.mu.RLock()
	defer cs.mu.RUnlock()
	
	item, exists := cs.items[key]
	if !exists {
		return nil
	}
	
	// Check if expired
	if time.Now().After(item.expiration) {
		return nil
	}
	
	return item.value
}

// Set stores a value in cache with TTL
func (cs *CacheService) Set(key string, value interface{}, ttl time.Duration) {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	
	cs.items[key] = &cacheItem{
		value:      value,
		expiration: time.Now().Add(ttl),
	}
}

// Delete removes a value from cache
func (cs *CacheService) Delete(key string) {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	
	delete(cs.items, key)
}

// Clear removes all items from cache
func (cs *CacheService) Clear() {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	
	cs.items = make(map[string]*cacheItem)
}

// cleanupExpired periodically removes expired items
func (cs *CacheService) cleanupExpired() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()
	
	for range ticker.C {
		cs.mu.Lock()
		now := time.Now()
		for key, item := range cs.items {
			if now.After(item.expiration) {
				delete(cs.items, key)
			}
		}
		cs.mu.Unlock()
	}
}

// TODO: Add Redis integration
// func (cs *CacheService) initRedis(addr string, password string, db int) error {
//     cs.redis = redis.NewClient(&redis.Options{
//         Addr:     addr,
//         Password: password,
//         DB:       db,
//     })
//     
//     return cs.redis.Ping(context.Background()).Err()
// }