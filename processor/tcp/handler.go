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
	"nursor.org/nursorgate/processor/rules"
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
		metadata.Route = "RouteDirect" // HTTP defaults to direct connection
		remoteConn, err = h.dialDirect(ctx, metadata)
	default:
		metadata.Route = "RouteDirect" // Other protocols default to direct connection
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
	err = h.relayManager.Relay(ctx, newOriginConn, trackedRemote, metadata)

	// Store DNS binding to cache after successful relay
	// This persists domain-IP relationships for future cache hits
	if err == nil && metadata.DNSInfo != nil && metadata.DNSInfo.ShouldCache {
		engine := rules.GetEngine()
		if engine != nil {
			engine.StoreBinding(metadata)
		}
	}

	return err
}

// handleTLS processes TLS connections (port 443)
// Implements three-step routing decision process:
// 1. Attempt IP→Domain reverse lookup from cache
// 2. If cache miss, extract SNI from TLS ClientHello
// 3. Determine route based on rules engine with available domain
// 4. GeoIP routing only used as last fallback if no domain info
func (h *TCPConnectionHandler) handleTLS(
	ctx context.Context,
	originConn net.Conn,
	metadata *M.Metadata,
) (remoteConn net.Conn, newOriginConn net.Conn, err error) {
	var sni string
	var sniBuf []byte
	var wrapped *WrappedConn
	var cacheHit bool

	// STEP 1: Attempt cache reverse lookup by destination IP
	// This is the hot path - check if we've seen this IP-domain pair before
	cache := rules.GetCache()
	if cache != nil && !metadata.DstIP.IsUnspecified() {
		cacheEntries := cache.GetByIP(metadata.DstIP)
		if len(cacheEntries) > 0 {
			// Use the first entry's domain for routing
			cachedEntry := cacheEntries[0]
			metadata.HostName = cachedEntry.Domain
			cacheHit = true
			logger.Debug(fmt.Sprintf("TLS: Found domain in cache for IP %s: %s (hit count: %d)",
				metadata.DstIP, metadata.HostName, cachedEntry.HitCount))
		}
	}

	// STEP 2: Only extract SNI if we didn't find domain in cache
	if !cacheHit || metadata.HostName == "" {
		logger.Debug("TLS: Cache miss or empty hostname, extracting SNI from TLS ClientHello")
		sni, sniBuf, err = h.tlsHandler.ExtractSNI(ctx, originConn)

		if err != nil {
			logger.Debug(fmt.Sprintf("SNI extraction error: %v", err))
			sni = ""
		} else if sni != "" {
			// Successfully extracted SNI, update metadata and record binding
			metadata.HostName = sni

			// Check if this is a DoH (DNS over HTTPS) provider
			// DoH traffic should be routed directly without proxy interception
			if IsDoHProvider(sni) {
				logger.Info(fmt.Sprintf("[DoH] Detected DoH provider: %s, routing directly", sni))
				// Return nil to allow direct connection (bypass proxy)
				// This means the connection will be handled directly without MITM or proxy routing
				return nil, nil, nil
			}

			// Record SNI binding information for DNS caching
			// This allows future reverse lookups to find this domain-IP relationship
			if metadata.DNSInfo == nil {
				metadata.DNSInfo = &M.DNSInfo{
					BindingSource: M.BindingSourceSNI,
					BindingTime:   time.Now(),
					CacheTTL:      5 * time.Minute,
					ShouldCache:   true,
				}
			}
			logger.Debug(fmt.Sprintf("TLS: Extracted SNI: %s", sni))
		}
	} else if metadata.HostName != "" {
		// We have domain from cache, check if it's a DoH provider
		if IsDoHProvider(metadata.HostName) {
			logger.Info(fmt.Sprintf("[DoH] Detected DoH provider from cache: %s, routing directly", metadata.HostName))
			// Return nil to allow direct connection (bypass proxy)
			return nil, nil, nil
		}
	}

	// Wrap the connection with buffered SNI data for protocols that need it
	wrapped = &WrappedConn{
		Conn: originConn,
		Buf:  sniBuf,
	}

	// STEP 3: Determine route based on rules engine with domain info available
	// Now that we have domain info (from cache or SNI extraction), the rules engine
	// can make a more informed routing decision
	route, _ := h.tlsHandler.DetermineRouteWithContext(metadata)

	// Log routing decision for debugging
	var routeSource string
	if cacheHit && metadata.HostName != "" {
		routeSource = "cache"
	} else if sni != "" {
		routeSource = "SNI"
	} else {
		routeSource = "IP/GeoIP"
	}
	logger.Debug(fmt.Sprintf("TLS: Route decision for %s (%s via %s): %v", metadata.HostName, metadata.DstIP, routeSource, route))

	// STEP 4: Route based on final decision
	switch route {
	case RouteToCursor:
		// Store route decision in metadata for caching
		metadata.Route = "RouteToCursor"

		// MITM proxy route (Cursor/Nonelane)
		// Extract SNI for MITM if we have it
		mitmedSNI := sni
		if cacheHit && sni == "" {
			// If we got domain from cache but not SNI, use cached domain as SNI
			mitmedSNI = metadata.HostName
		}

		mitmed, err := h.tlsHandler.PerformMITM(ctx, originConn, mitmedSNI)
		if err != nil {
			return nil, nil, err
		}

		// Connect through Nonelane proxy
		nonelaneProxy, err := registry.GetRegistry().GetNonelane()
		if err != nil {
			return nil, nil, err
		}

		remote, err := nonelaneProxy.DialContext(ctx, metadata)
		if err != nil {
			return nil, nil, err
		}

		return remote, mitmed, nil

	case RouteToDoor:
		// Store route decision in metadata for caching
		metadata.Route = "RouteToDoor"

		// Route through door proxy (VLESS/Shadowsocks)
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
		// Store route decision in metadata for caching
		metadata.Route = "RouteDirect"

		// RouteDirect: Direct connection without proxy
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
