// Package routing provides routing decision engine interfaces
package routing

import (
	"nursor.org/nursorgate/inbound/tun/engine"
	"nursor.org/nursorgate/outbound/proxy"
)

// Engine interface for routing decisions
type Engine interface {
	// Start starts the routing engine
	Start() error

	// Stop stops the routing engine
	Stop() error

	// InsertKey loads configuration key
	InsertKey(k *engine.Key)

	// GetDefaultProxy returns the default proxy
	GetDefaultProxy() proxy.Proxy
}
