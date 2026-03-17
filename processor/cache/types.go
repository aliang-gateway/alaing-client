package cache

import (
	"net/netip"
	"time"

	M "nursor.org/nursorgate/inbound/tun/metadata"
)

// RouteDecision represents the routing decision for a connection
type RouteDecision string

const (
	// RouteToCursor routes traffic through Cursor MITM proxy (Nonelane)
	RouteToCursor RouteDecision = "cursor"

	// RouteToSocks routes traffic through SOCKS proxy
	RouteToSocks RouteDecision = "socks"

	// RouteDirect routes traffic directly without proxy
	RouteDirect RouteDecision = "direct"
)

// CacheEntry represents a cached routing decision for an IP-domain pair
type CacheEntry struct {
	Domain         string            // Domain name (may be empty for pure IP traffic)
	IP             netip.Addr        // Destination IP address
	Route          RouteDecision     // Routing decision
	BindingSources []M.BindingSource // Sources where this binding came from (SNI, HTTP, DNS, etc.)
	ExpiresAt      time.Time         // Expiration time (TTL-based)
	CreatedAt      time.Time         // Creation time for statistics
	HitCount       uint64            // Number of times this entry was accessed
}

// IsExpired checks if the cache entry has expired
func (e *CacheEntry) IsExpired() bool {
	return time.Now().After(e.ExpiresAt)
}

// TimeToLive returns the remaining time until expiration
func (e *CacheEntry) TimeToLive() time.Duration {
	return time.Until(e.ExpiresAt)
}
