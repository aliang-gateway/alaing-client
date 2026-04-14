package aliang

import (
	"context"
	"crypto/tls"
	"fmt"
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

type ProbeTimings struct {
	TCPConnect   time.Duration
	TLSHandshake time.Duration
	Total        time.Duration
}

func (pt ProbeTimings) DisplayLatency() time.Duration {
	if pt.TCPConnect > 0 {
		return pt.TCPConnect
	}
	if pt.Total > 0 {
		return pt.Total
	}
	return 0
}

// NewAliangServerConnector creates a new cursor server connector
func NewAliangServerConnector(config *AliangConfig) *AliangServerConnector {
	return &AliangServerConnector{
		config: config,
	}
}

// Dial establishes a mTLS connection to the aliang server.
// appProto controls whether the tunnel should advertise h2 ALPN.
func (csc *AliangServerConnector) Dial(ctx context.Context, network, address string, appProto string) (net.Conn, error) {
	conn, _, err := csc.DialWithTiming(ctx, network, address, appProto)
	return conn, err
}

func (csc *AliangServerConnector) DialWithTiming(ctx context.Context, network, address string, appProto string) (net.Conn, ProbeTimings, error) {
	logger.Info(fmt.Sprintf("[AliangGate] Connecting to server %s (app_proto=%s)", address, appProto))
	serverName := normalizeServerName(address)
	enableHTTP2ALPN := appProto == "http2"
	tlsConfig, err := clientcert.GetMTLSClientTLSConfig(enableHTTP2ALPN, serverName)
	if err != nil {
		logger.Error(fmt.Sprintf("[AliangGate] Failed to build outbound TLS config for %s: %v", address, err))
		return nil, ProbeTimings{}, NewErrorWithCause(ErrTLSHandshakeFailed, "failed to load outbound TLS config", err)
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

	startedAt := time.Now()
	conn, err := dialer.DialContext(ctx, network, address)
	if err != nil {
		logger.Warn(fmt.Sprintf("[AliangGate] TCP connect failed to %s: %v", address, err))
		return nil, ProbeTimings{}, NewErrorWithCause(ErrTLSHandshakeFailed, "failed to dial cursor server", err)
	}
	tcpConnectedAt := time.Now()
	logger.Debug(fmt.Sprintf("[AliangGate] TCP connected to %s in %v", address, tcpConnectedAt.Sub(startedAt)))

	// Upgrade connection to TLS using hardcoded certs
	tlsConn := tls.Client(conn, tlsConfig)
	handshakeStartedAt := time.Now()
	if err := tlsConn.HandshakeContext(ctx); err != nil {
		conn.Close()
		logger.Warn(fmt.Sprintf("[AliangGate] mTLS handshake failed with %s: %v", address, err))
		return nil, ProbeTimings{}, NewErrorWithCause(ErrTLSHandshakeFailed, "mTLS handshake failed", err)
	}
	handshakeCompletedAt := time.Now()

	logger.Info(fmt.Sprintf("[AliangGate] mTLS tunnel ready: server=%s app_proto=%s alpn=%s tcp=%v tls=%v",
		address,
		appProto,
		tlsConn.ConnectionState().NegotiatedProtocol,
		tcpConnectedAt.Sub(startedAt).Round(time.Millisecond),
		handshakeCompletedAt.Sub(handshakeStartedAt).Round(time.Millisecond)))
	return tlsConn, ProbeTimings{
		TCPConnect:   tcpConnectedAt.Sub(startedAt),
		TLSHandshake: handshakeCompletedAt.Sub(handshakeStartedAt),
		Total:        handshakeCompletedAt.Sub(startedAt),
	}, nil
}

func normalizeServerName(address string) string {
	host, _, err := net.SplitHostPort(address)
	if err == nil && host != "" {
		return host
	}
	return address
}
