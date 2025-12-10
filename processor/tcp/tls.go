package tcp

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"strings"

	"nursor.org/nursorgate/common/logger"
	"nursor.org/nursorgate/common/model"
	M "nursor.org/nursorgate/inbound/tun/metadata"
	"nursor.org/nursorgate/processor/cache"
	cert_client "nursor.org/nursorgate/processor/cert/client"
	"nursor.org/nursorgate/processor/rules"
	tls_helper "nursor.org/nursorgate/processor/tls"
	watcher "nursor.org/nursorgate/processor/watcher"
)

// DefaultTLSHandler implements the TLSHandler interface.
// It handles SNI extraction, MITM certificate generation, and domain routing.
type DefaultTLSHandler struct {
	// Whether to enable cursor proxy (for request interception)
	cursorProxyEnabled bool
}

// NewDefaultTLSHandler creates a new TLS handler
func NewDefaultTLSHandler() *DefaultTLSHandler {
	return &DefaultTLSHandler{
		cursorProxyEnabled: watcher.IsCursorProxyEnabled,
	}
}

// ExtractSNI extracts the Server Name Indication from a TLS ClientHello.
// It delegates to processor/tls/tls_sni_helper.go which has the low-level parsing.
func (h *DefaultTLSHandler) ExtractSNI(ctx context.Context, conn net.Conn) (string, []byte, error) {
	// Apply context timeout to the read
	if deadline, ok := ctx.Deadline(); ok {
		conn.SetReadDeadline(deadline)
	}

	// Use the existing SNI extraction from processor/tls
	serverName, buffer, err := tls_helper.ExtractSNI(conn)
	if err != nil {
		logger.Debug("SNI extraction failed: " + err.Error())
		return "", buffer, err
	}

	logger.Debug("Extracted SNI: " + serverName)
	return serverName, buffer, nil
}

// PerformMITM creates and performs a TLS handshake with intercepted certificate.
// Steps:
// 1. Generate/retrieve certificate for serverName
// 2. Create TLS config with certificate
// 3. Perform TLS handshake as server
// 4. Return the TLS connection
func (h *DefaultTLSHandler) PerformMITM(ctx context.Context, originConn net.Conn, serverName string) (net.Conn, error) {
	// Generate TLS config for MITM
	tlsConfig := cert_client.CreateTlsConfigForHost(serverName)
	if tlsConfig == nil {
		return nil, fmt.Errorf("failed to create TLS config for host: %s", serverName)
	}

	// Create TLS server connection
	tlsConn := tls.Server(originConn, tlsConfig)

	// Apply context timeout to handshake
	if deadline, ok := ctx.Deadline(); ok {
		originConn.SetReadDeadline(deadline)
	}

	// Perform TLS handshake
	if err := tlsConn.Handshake(); err != nil {
		logger.Error(fmt.Sprintf("TLS handshake failed for %s: %v", serverName, err))
		return nil, err
	}

	// Log successful handshake
	state := tlsConn.ConnectionState()
	logger.Debug(fmt.Sprintf("TLS handshake successful for %s. Protocol: %s, Version: 0x%04x",
		serverName, state.NegotiatedProtocol, state.Version))

	return tlsConn, nil
}

// DetermineRoute checks if a domain should be routed to cursor, door, or direct.
// It uses the domain allowlist from model.AllowProxyDomain.
func (h *DefaultTLSHandler) DetermineRoute(serverName string) ProxyRoute {
	if serverName == "" {
		return RouteDirect
	}

	router := model.NewAllowProxyDomain()

	// Check if allowed to cursor proxy (MITM interception)
	if router.IsAllowToCursor(serverName) {
		return RouteToCursor
	}

	// Check if allowed to door proxy (gateway)
	if router.IsAllowToAnyDoor(serverName) {
		return RouteToDoor
	}

	// Default to direct connection
	return RouteDirect
}

// IsDoHProvider checks if a domain is a DNS-over-HTTPS provider.
// DoH providers typically offer DNS resolution over HTTPS which should not be intercepted.
func IsDoHProvider(domain string) bool {
	if domain == "" {
		return false
	}

	domain = strings.ToLower(domain)

	// List of known DoH providers
	dohProviders := []string{
		DoHProviderGoogle,
		DoHProviderCloudflare,
		DoHProviderOpenDNS,
		DoHProviderQuad9,
		DoHProviderCleanBrowse,
		DoHProviderGoogle8,
		DoHProviderGoogle9,
		DoHProviderCloudflare1,
		DoHProviderCloudflare2,
		DoHProviderQuad9ip,
	}

	for _, provider := range dohProviders {
		if strings.Contains(domain, provider) {
			return true
		}
	}

	return false
}

// DetectDoH detects if an established TLS connection is being used for DNS-over-HTTPS.
// This is useful because some applications establish HTTPS connections that are actually
// used for DNS queries. The HTTP/2 PRI frame or HTTP request will contain "/dns-query" or "/resolve".
func DetectDoH(tlsConn net.Conn) bool {
	// Try to peek at the first few bytes to detect HTTP request
	// This is a simple heuristic check
	if tc, ok := tlsConn.(interface{ Peek(int) ([]byte, error) }); ok {
		data, err := tc.Peek(256)
		if err == nil {
			dataStr := string(data)
			if strings.Contains(dataStr, "/dns-query") || strings.Contains(dataStr, "/resolve") {
				return true
			}
		}
	}

	return false
}

// TLSConnectionInfo contains information about an established TLS connection
type TLSConnectionInfo struct {
	ServerName         string
	Protocol           string
	Version            uint16
	CipherSuite        uint16
	IsResumed          bool
	VerificationError  error
}

// GetTLSConnectionInfo extracts information from an established TLS connection
func GetTLSConnectionInfo(tlsConn *tls.Conn) *TLSConnectionInfo {
	state := tlsConn.ConnectionState()
	return &TLSConnectionInfo{
		ServerName:  state.ServerName,
		Protocol:    state.NegotiatedProtocol,
		Version:     state.Version,
		CipherSuite: state.CipherSuite,
		IsResumed:   state.DidResume,
	}
}

// WrappedConnWithTLS combines an origin connection with TLS metadata
// for easier handling of SNI-extracted connections
type WrappedConnWithTLS struct {
	net.Conn
	SNI    string // Extracted Server Name Indication
	Buffer []byte // Preserved TLS ClientHello
}

// DetermineRouteWithContext uses the rule engine to make intelligent routing decisions.
// This method leverages:
// 1. Bypass rules (user-configured direct routes)
// 2. IP-Domain cache (avoid repeated SNI extraction)
// 3. Nacos rules (Cursor MITM and Door acceleration)
// 4. GeoIP routing (country-based decisions)
//
// Returns both the routing decision and whether SNI extraction is required.
func (h *DefaultTLSHandler) DetermineRouteWithContext(metadata *M.Metadata) (ProxyRoute, bool) {
	// CONNECT tunnel requests must be routed directly without going through proxies.
	// CONNECT tunnels require raw TCP passthrough, which is incompatible with application-layer
	// proxies like Door (VLESS/Shadowsocks) that expect protocol-specific handshakes.
	if metadata.IsFromCONNECT {
		logger.Debug("CONNECT tunnel detected: routing directly without proxy")
		return RouteDirect, false
	}

	engine := rules.GetEngine()

	// If rule engine is disabled or not initialized, fallback to old logic
	if engine == nil || !engine.IsEnabled() {
		return h.DetermineRoute(metadata.HostName), true
	}

	// Build evaluation context
	ctx := &rules.EvaluationContext{
		DstIP:    metadata.DstIP,
		DstPort:  metadata.DstPort,
		SrcIP:    metadata.SrcIP,
		Domain:   metadata.HostName,
		Protocol: "tcp",
	}

	// Evaluate routing rules
	result, err := engine.EvaluateRoute(ctx)
	if err != nil {
		logger.Warn(fmt.Sprintf("Rule engine error: %v, fallback to old logic", err))
		return h.DetermineRoute(metadata.HostName), true
	}

	// Log decision for debugging
	logger.Debug(fmt.Sprintf("Route decision: %s (rule: %s, reason: %s, requiresSNI: %v)",
		result.Route, result.MatchedRule, result.Reason, result.RequiresSNI))

	// Convert cache.RouteDecision to ProxyRoute
	var proxyRoute ProxyRoute
	switch result.Route {
	case cache.RouteToCursor:
		proxyRoute = RouteToCursor
	case cache.RouteToDoor:
		proxyRoute = RouteToDoor
	default:
		proxyRoute = RouteDirect
	}

	return proxyRoute, result.RequiresSNI
}
