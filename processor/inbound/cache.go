package inbound

import (
	"sync"
	"time"
)

// InboundsCache manages in-memory cache for inbound configurations
type InboundsCache struct {
	mu        sync.RWMutex
	inbounds  []InboundInfo
	timestamp int64 // Last update time (Unix seconds)
}

var cache = &InboundsCache{
	inbounds:  []InboundInfo{},
	timestamp: 0,
}

// GetCachedInbounds retrieves cached inbound configurations
func GetCachedInbounds() ([]InboundInfo, int64) {
	cache.mu.RLock()
	defer cache.mu.RUnlock()

	// Return copies to prevent external modification
	inboundsCopy := make([]InboundInfo, len(cache.inbounds))
	copy(inboundsCopy, cache.inbounds)

	return inboundsCopy, cache.timestamp
}

// SetCachedInbounds updates the in-memory cache with new inbound configurations
func SetCachedInbounds(inbounds []InboundInfo) {
	cache.mu.Lock()
	defer cache.mu.Unlock()

	// Store copy to prevent external modification
	inboundsCopy := make([]InboundInfo, len(inbounds))
	copy(inboundsCopy, inbounds)

	cache.inbounds = inboundsCopy
	cache.timestamp = time.Now().Unix()
}

// ClearCachedInbounds clears the in-memory cache
func ClearCachedInbounds() {
	cache.mu.Lock()
	defer cache.mu.Unlock()

	cache.inbounds = []InboundInfo{}
	cache.timestamp = 0
}

// HasCachedInbounds checks if cache has any inbound configurations
func HasCachedInbounds() bool {
	cache.mu.RLock()
	defer cache.mu.RUnlock()

	return len(cache.inbounds) > 0
}
