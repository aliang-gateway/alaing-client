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
	logger.Debug("[cursor_h2] Starting mTLS handshake with", address, " app_proto=", appProto)
	serverName := normalizeServerName(address)
	enableHTTP2ALPN := appProto == "http2"
	tlsConfig, err := clientcert.GetMTLSClientTLSConfig(enableHTTP2ALPN, serverName)
	if err != nil {
		logger.Error("[cursor_h2] Failed to build outbound TLS config for", address, err)
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
		logger.Error("[cursor_h2] Failed to dial cursor server", address, err)
		return nil, ProbeTimings{}, NewErrorWithCause(ErrTLSHandshakeFailed, "failed to dial cursor server", err)
	}
	tcpConnectedAt := time.Now()

	// Upgrade connection to TLS using hardcoded certs
	tlsConn := tls.Client(conn, tlsConfig)
	handshakeStartedAt := time.Now()
	if err := tlsConn.HandshakeContext(ctx); err != nil {
		conn.Close()
		logger.Error("[cursor_h2] mTLS handshake failed with", address, err)
		return nil, ProbeTimings{}, NewErrorWithCause(ErrTLSHandshakeFailed, "mTLS handshake failed", err)
	}
	handshakeCompletedAt := time.Now()

	logger.Info(fmt.Sprintf(
		"[AliangGate] mtls tunnel ready server=%s app_proto=%s negotiated_alpn=%s",
		address,
		appProto,
		tlsConn.ConnectionState().NegotiatedProtocol,
	))
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
