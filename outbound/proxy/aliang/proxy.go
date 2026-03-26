package aliang

import (
	"context"
	"net"
	"sync"

	"nursor.org/nursorgate/inbound/tun/metadata"
	"nursor.org/nursorgate/outbound/proxy"
	"nursor.org/nursorgate/outbound/proxy/proto"
)

// Aliang implements the Proxy interface for cursor H2 proxy
// Core responsibility: mTLS connection establishment and pooling
type Aliang struct {
	*proxy.Base
	config    *AliangConfig
	connector *AliangServerConnector
	connPool  *ConnectionPool
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
			Protocol: proto.Aliang, // 使用 HY2 作为协议类型，或者可以添加新的类型
		},
		config:    config,
		connector: NewAliangServerConnector(config),
		connPool:  NewConnectionPool(config.ConnectionPool),
		closed:    false,
	}, nil
}

// DialContext implements the Proxy interface
// Establishes a connection to the target address through the cursor H2 proxy
func (c *Aliang) DialContext(ctx context.Context, metadata *metadata.Metadata) (net.Conn, error) {
	c.mu.RLock()
	if c.closed {
		c.mu.RUnlock()
		return nil, NewErrorf(ErrInvalidConfig, "proxy is closed")
	}
	c.mu.RUnlock()

	// Get destination address from metadata
	address := metadata.DestinationAddress()

	// Try to get connection from pool
	pooledConn := c.connPool.Get(address)
	if pooledConn != nil && pooledConn.Conn != nil {
		return pooledConn.Conn, nil
	}

	// Establish new mTLS connection to cursor server
	conn, err := c.connector.Dial(ctx, "tcp", c.config.Addr)
	if err != nil {
		return nil, err
	}

	// Store connection in pool for reuse
	pooledConn = &PooledConn{
		Conn: conn,
	}
	if err := c.connPool.Put(address, pooledConn); err != nil {
		conn.Close()
		return nil, err
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
	}
}
