package nonelane

import (
	"fmt"
	"time"
)

// NoneLaneConfig contains configuration for the cursor_h2 proxy
type NoneLaneConfig struct {
	// Addr is the cursor server address (host:port)
	Addr string

	// DialTimeout is the timeout for establishing connections
	DialTimeout time.Duration

	// ReadTimeout is the timeout for read operations
	ReadTimeout time.Duration

	// WriteTimeout is the timeout for write operations
	WriteTimeout time.Duration

	// MaxConcurrentStreams is the maximum number of concurrent HTTP/2 streams
	MaxConcurrentStreams uint32

	// ConnectionPoolConfig contains connection pool settings
	ConnectionPool *ConnectionPoolConfig

	// TLSConfig contains TLS/mTLS settings
	TLSConfig *TLSConfigOptions
}

// ConnectionPoolConfig contains settings for the connection pool
type ConnectionPoolConfig struct {
	// MaxConnPerHost is the maximum number of connections per host
	MaxConnPerHost int

	// MaxIdleTime is the maximum idle time for connections
	MaxIdleTime time.Duration

	// CleanupInterval is the interval for cleanup of idle connections
	CleanupInterval time.Duration
}

// TLSConfigOptions contains TLS/mTLS settings
type TLSConfigOptions struct {
	// InsecureSkipVerify disables certificate verification
	InsecureSkipVerify bool

	// ServerName is the server name for SNI
	ServerName string

	// CAFile is the path to CA certificate file
	CAFile string

	// CertFile is the path to client certificate file
	CertFile string

	// KeyFile is the path to client key file
	KeyFile string
}

// DefaultConfig creates a default configuration for the given server address
func DefaultConfig(addr string) *NoneLaneConfig {
	return &NoneLaneConfig{
		Addr:                 addr,
		DialTimeout:          10 * time.Second,
		ReadTimeout:          30 * time.Second,
		WriteTimeout:         30 * time.Second,
		MaxConcurrentStreams: 250,
		ConnectionPool: &ConnectionPoolConfig{
			MaxConnPerHost:  4,
			MaxIdleTime:     5 * time.Minute,
			CleanupInterval: 1 * time.Minute,
		},
		TLSConfig: &TLSConfigOptions{
			InsecureSkipVerify: false,
			ServerName:         "",
		},
	}
}

// Validate validates the configuration
func (c *NoneLaneConfig) Validate() error {
	if c == nil {
		return NewErrorf(ErrInvalidConfig, "config is nil")
	}

	if c.Addr == "" {
		return NewErrorf(ErrInvalidConfig, "addr is required")
	}

	if c.DialTimeout == 0 {
		return NewErrorf(ErrInvalidConfig, "dial timeout must be > 0")
	}

	if c.ReadTimeout == 0 {
		return NewErrorf(ErrInvalidConfig, "read timeout must be > 0")
	}

	if c.WriteTimeout == 0 {
		return NewErrorf(ErrInvalidConfig, "write timeout must be > 0")
	}

	if c.MaxConcurrentStreams == 0 {
		c.MaxConcurrentStreams = 250 // Set default
	}

	if c.ConnectionPool == nil {
		c.ConnectionPool = &ConnectionPoolConfig{
			MaxConnPerHost:  4,
			MaxIdleTime:     5 * time.Minute,
			CleanupInterval: 1 * time.Minute,
		}
	}

	if c.ConnectionPool.MaxConnPerHost <= 0 {
		return NewErrorf(ErrInvalidConfig, "connection pool max conn per host must be > 0")
	}

	if c.ConnectionPool.MaxIdleTime <= 0 {
		return NewErrorf(ErrInvalidConfig, "connection pool max idle time must be > 0")
	}

	if c.TLSConfig == nil {
		c.TLSConfig = &TLSConfigOptions{
			InsecureSkipVerify: false,
		}
	}

	return nil
}

// String returns a string representation of the configuration
func (c *NoneLaneConfig) String() string {
	return fmt.Sprintf(
		"CursorH2Config{Addr: %s, DialTimeout: %v, MaxConcurrentStreams: %d}",
		c.Addr,
		c.DialTimeout,
		c.MaxConcurrentStreams,
	)
}
