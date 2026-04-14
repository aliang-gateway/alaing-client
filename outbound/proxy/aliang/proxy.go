package aliang

import (
	"context"
	"fmt"
	"net"
	"sync"

	"aliang.one/nursorgate/common/logger"
	"aliang.one/nursorgate/inbound/tun/metadata"
	"aliang.one/nursorgate/outbound/proxy"
	"aliang.one/nursorgate/outbound/proxy/proto"
)

// Aliang implements the Proxy interface for cursor H2 proxy
// Core responsibility: mTLS connection establishment and pooling
type Aliang struct {
	*proxy.Base
	config    *AliangConfig
	connector *AliangServerConnector
	connPool  *ConnectionPool
	status    *linkStatusTracker
	mu        sync.RWMutex
	closed    bool
}

// New creates a new CursorH2 proxy instance
func NewAliang(config *AliangConfig) (*Aliang, error) {
	if config == nil {
		return nil, NewErrorf(ErrInvalidConfig, "config is required")
	}

	if err := config.Validate(); err != nil {
		return nil, err
	}

	return &Aliang{
		Base: &proxy.Base{
			Address:  config.Addr,
			Protocol: proto.Aliang,
		},
		config:    config,
		connector: NewAliangServerConnector(config),
		connPool:  NewConnectionPool(config.ConnectionPool),
		status:    newLinkStatusTracker(config.Addr, 0),
		closed:    false,
	}, nil
}

// DialContext implements the Proxy interface
func (c *Aliang) DialContext(ctx context.Context, metadata *metadata.Metadata) (net.Conn, error) {
	c.mu.RLock()
	if c.closed {
		c.mu.RUnlock()
		return nil, NewErrorf(ErrInvalidConfig, "proxy is closed")
	}
	c.mu.RUnlock()

	// Establish a dedicated mTLS connection for the current tunneled TCP session.
	// Raw byte relaying is not safe to reuse across independent sessions, even if
	// they target the same destination and protocol.
	c.status.markConnecting()
	if metadata != nil && metadata.ConnID != "" {
		ctx = context.WithValue(ctx, aliangContextConnIDKey{}, metadata.ConnID)
	}
	conn, timing, err := c.connector.DialWithTiming(ctx, "tcp", c.config.Addr, metadata.AppProto)
	if err != nil {
		c.status.markFailure(describeProbeFailure(c.config.Addr, err))
		return nil, err
	}
	c.status.markSuccess(timing)

	if metadata != nil {
		appProto := metadata.AppProto
		if appProto == "" {
			appProto = "unknown"
		}
		logger.Debug(fmt.Sprintf("[AliangGate] conn_id=%s established dedicated mtls session app_proto=%s target=%s via=%s", metadata.ConnID, appProto, metadata.DestinationAddress(), c.config.Addr))
	}

	return conn, nil
}

// DialUDP implements the Proxy interface
// UDP is not supported for cursor_h2 proxy
func (c *Aliang) DialUDP(metadata *metadata.Metadata) (net.PacketConn, error) {
	return nil, NewErrorf(ErrInvalidConfig, "cursor_h2 does not support UDP")
}

// Close closes the proxy and releases resources
func (c *Aliang) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return nil
	}

	c.closed = true

	// Close connection pool
	if c.connPool != nil {
		c.connPool.Close()
	}

	return nil
}

// GetStats returns statistics about the proxy
func (c *Aliang) GetStats() map[string]interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return map[string]interface{}{
		"addr":            c.config.Addr,
		"proto":           "cursor_h2",
		"closed":          c.closed,
		"connection_pool": c.connPool.Stats(),
		"link_status":     c.status.snapshotMap(),
	}
}

// LinkStatusSnapshot returns the latest observed mTLS link status without forcing a new probe.
func (c *Aliang) LinkStatusSnapshot() map[string]interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.closed {
		return unavailableLinkStatus(c.config.Addr, NewErrorf(ErrInvalidConfig, "proxy is closed"))
	}
	return c.status.snapshotMap()
}

// ProbeLink actively performs a new mTLS dial to measure reachability and latency.
func (c *Aliang) ProbeLink(ctx context.Context) map[string]interface{} {
	c.mu.RLock()
	if c.closed {
		c.mu.RUnlock()
		return unavailableLinkStatus(c.config.Addr, NewErrorf(ErrInvalidConfig, "proxy is closed"))
	}
	serverAddr := c.config.Addr
	c.mu.RUnlock()

	c.status.markConnecting()

	conn, timing, err := c.connector.DialWithTiming(ctx, "tcp", serverAddr, "unknown")
	if err != nil {
		c.status.markFailure(describeProbeFailure(serverAddr, err))
		return c.status.snapshotMap()
	}
	_ = conn.Close()

	c.status.markSuccess(timing)
	return c.status.snapshotMap()
}
