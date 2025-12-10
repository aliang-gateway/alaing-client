package tcp

import (
	"context"
	"fmt"
	"net"
	"net/netip"
	"time"

	"nursor.org/nursorgate/common/logger"
	"nursor.org/nursorgate/inbound/tun/dialer"
	M "nursor.org/nursorgate/inbound/tun/metadata"
	registry "nursor.org/nursorgate/processor/registry"
	"nursor.org/nursorgate/processor/statistic"
)

// parseNetIPAddr converts a string IP to netip.Addr
func parseNetIPAddr(ipStr string) (netip.Addr, error) {
	return netip.ParseAddr(ipStr)
}

// TCPConnectionHandler implements the TCPConnHandler interface.
// It orchestrates TCP connection handling from protocol detection
// through routing and bidirectional data relay.
type TCPConnectionHandler struct {
	protocolDetector ProtocolDetector
	tlsHandler       TLSHandler
	relayManager     RelayManager
	statsManager     *statistic.Manager
}

// NewTCPConnectionHandler creates a new TCP handler
func NewTCPConnectionHandler(
	protocolDetector ProtocolDetector,
	tlsHandler TLSHandler,
	relayManager RelayManager,
	statsManager *statistic.Manager,
) *TCPConnectionHandler {
	return &TCPConnectionHandler{
		protocolDetector: protocolDetector,
		tlsHandler:       tlsHandler,
		relayManager:     relayManager,
		statsManager:     statsManager,
	}
}

// Handle processes a single TCP connection.
// It is the main orchestration entry point.
func (h *TCPConnectionHandler) Handle(ctx context.Context, originConn net.Conn, metadata *M.Metadata) error {
	// Ensure we close the origin connection when done
	defer originConn.Close()

	// Create timeout context
	timeout := time.Duration(DefaultTCPConnectTimeout) * time.Second
	if deadline, ok := ctx.Deadline(); ok {
		timeout = time.Until(deadline)
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// Detect protocol
	protocol := h.protocolDetector.Detect(metadata.DstPort)
	logger.Debug(fmt.Sprintf("TCP: handling %v connection to %s:%d",
		protocol, metadata.DstIP, metadata.DstPort))

	var remoteConn net.Conn
	var newOriginConn net.Conn = originConn
	var err error

	// Route based on protocol
	switch protocol {
	case ProtocolTLS:
		remoteConn, newOriginConn, err = h.handleTLS(ctx, originConn, metadata)
	case ProtocolHTTP:
		remoteConn, err = h.dialDirect(ctx, metadata)
	default:
		remoteConn, err = h.dialDirect(ctx, metadata)
	}

	if err != nil {
		logger.Error(fmt.Sprintf("[TCP HANDLER] ❌ 处理失败 - 域名:%s, 端口:%d, 错误:%v", metadata.HostName, metadata.DstPort, err))
		return err
	}

	if remoteConn == nil {
		// Special handling (e.g., DoH)
		return nil
	}

	defer remoteConn.Close()

	// Update metadata MidIP/MidPort
	if localAddr, ok := remoteConn.LocalAddr().(*net.TCPAddr); ok {
		if ip, err := parseNetIPAddr(localAddr.IP.String()); err == nil {
			metadata.MidIP = ip
			metadata.MidPort = uint16(localAddr.Port)
		}
	}

	// Track statistics
	trackedRemote := statistic.NewTCPTracker(remoteConn, metadata, h.statsManager)
	defer trackedRemote.Close()

	// Relay data bidirectionally
	return h.relayManager.Relay(ctx, newOriginConn, trackedRemote, metadata)
}

// handleTLS processes TLS connections (port 443)
func (h *TCPConnectionHandler) handleTLS(
	ctx context.Context,
	originConn net.Conn,
	metadata *M.Metadata,
) (remoteConn net.Conn, newOriginConn net.Conn, err error) {
	// Step 1: Pre-evaluate routing WITHOUT SNI (using IP, cache, bypass rules, GeoIP)
	route, requiresSNI := h.tlsHandler.DetermineRouteWithContext(metadata)

	var sni string
	var sniBuf []byte
	var wrapped *WrappedConn

	// Step 2: Only extract SNI if the rule engine says it's necessary
	logger.Debug("Rule engine requires SNI extraction")
	sni, sniBuf, err = h.tlsHandler.ExtractSNI(ctx, originConn)
	metadata.HostName = sni

	wrapped = &WrappedConn{
		Conn: originConn,
		Buf:  sniBuf,
	}

	if err != nil {
		logger.Debug(fmt.Sprintf("SNI extraction error: %v", err))
		// Check for DoH
		if IsDoHProvider(sni) {
			logger.Info(fmt.Sprintf("[DoH] Detected DoH for %s", sni))
			return nil, nil, nil
		}
		sni = ""
	}

	if requiresSNI {
		// Re-evaluate routing with SNI now available
		route, _ = h.tlsHandler.DetermineRouteWithContext(metadata)
	}

	switch route {
	case RouteToCursor:
		// MITM
		mitmed, err := h.tlsHandler.PerformMITM(ctx, originConn, sni)
		if err != nil {
			return nil, nil, err
		}

		// Connect through door proxy
		doorProxy, err := registry.GetRegistry().GetNonelane()
		if err != nil {
			return nil, nil, err
		}

		remote, err := doorProxy.DialContext(ctx, metadata)
		if err != nil {
			return nil, nil, err
		}

		return remote, mitmed, nil

	case RouteToDoor:
		// Route through door proxy
		doorProxy, err := registry.GetRegistry().GetDoor()
		if err != nil {
			return nil, nil, err
		}

		remote, err := doorProxy.DialContext(ctx, metadata)
		if err != nil {
			return nil, nil, err
		}

		return remote, wrapped, nil

	default:
		// Direct connection
		remote, err := h.dialDirect(ctx, metadata)
		if err != nil {
			return nil, nil, err
		}

		return remote, wrapped, nil
	}
}

// dialDirect dials a direct connection to the target using tun dialer
func (h *TCPConnectionHandler) dialDirect(ctx context.Context, metadata *M.Metadata) (net.Conn, error) {
	addr := net.JoinHostPort(metadata.DstIP.String(), fmt.Sprintf("%d", metadata.DstPort))
	conn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		logger.Debug(fmt.Sprintf("Dial failed: %v", err))
		return nil, err
	}

	return conn, nil
}
