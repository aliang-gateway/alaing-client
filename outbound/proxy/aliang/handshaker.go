package aliang

import (
	"context"
	"crypto/tls"
	"net"
	"time"

	"aliang.one/nursorgate/common/logger"
	clientcert "aliang.one/nursorgate/processor/cert/client"
)

// AliangServerConnector establishes mTLS connections to the cursor server
// Uses hardcoded client certificates embedded in processor/cert/server
type AliangServerConnector struct {
	config *AliangConfig
}

// NewAliangServerConnector creates a new cursor server connector
func NewAliangServerConnector(config *AliangConfig) *AliangServerConnector {
	return &AliangServerConnector{
		config: config,
	}
}

// Dial establishes a mTLS connection to the cursor server
// Uses hardcoded client certificates for authentication
func (csc *AliangServerConnector) Dial(ctx context.Context, network, address string) (net.Conn, error) {
	logger.Debug("[cursor_h2] Starting mTLS handshake with", address)
	serverName := normalizeServerName(address)
	tlsConfig, err := clientcert.GetMTLSClientTLSConfig(true, serverName)
	if err != nil {
		logger.Error("[cursor_h2] Failed to build outbound TLS config for", address, err)
		return nil, NewErrorWithCause(ErrTLSHandshakeFailed, "failed to load outbound TLS config", err)
	}

	// Use config timeout or default
	dialTimeout := csc.config.DialTimeout
	if dialTimeout == 0 {
		dialTimeout = 10 * time.Second
	}

	// Create context with timeout if not already present
	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, dialTimeout)
		defer cancel()
	}

	// Perform TLS handshake
	dialer := &net.Dialer{
		Timeout: dialTimeout,
	}

	conn, err := dialer.DialContext(ctx, network, address)
	if err != nil {
		logger.Error("[cursor_h2] Failed to dial cursor server", address, err)
		return nil, NewErrorWithCause(ErrTLSHandshakeFailed, "failed to dial cursor server", err)
	}

	// Upgrade connection to TLS using hardcoded certs
	tlsConn := tls.Client(conn, tlsConfig)
	if err := tlsConn.HandshakeContext(ctx); err != nil {
		conn.Close()
		logger.Error("[cursor_h2] mTLS handshake failed with", address, err)
		return nil, NewErrorWithCause(ErrTLSHandshakeFailed, "mTLS handshake failed", err)
	}

	logger.Debug("[cursor_h2] mTLS handshake successful with", address)
	return tlsConn, nil
}

func normalizeServerName(address string) string {
	host, _, err := net.SplitHostPort(address)
	if err == nil && host != "" {
		return host
	}
	return address
}
