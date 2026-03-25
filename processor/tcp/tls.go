package tcp

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"strings"
	"time"

	"nursor.org/nursorgate/common/logger"
	"nursor.org/nursorgate/common/model"
	M "nursor.org/nursorgate/inbound/tun/metadata"
	cert_client "nursor.org/nursorgate/processor/cert/client"
	"nursor.org/nursorgate/processor/config"
	"nursor.org/nursorgate/processor/routing"
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

func (h *DefaultTLSHandler) DetermineRoute(serverName string) ProxyRoute {
	metadata := &M.Metadata{HostName: serverName, DstPort: 443}
	route, _ := h.DetermineRouteWithContext(metadata)
	return route
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
	ServerName        string
	Protocol          string
	Version           uint16
	CipherSuite       uint16
	IsResumed         bool
	VerificationError error
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

func (h *DefaultTLSHandler) DetermineRouteWithContext(metadata *M.Metadata) (ProxyRoute, bool) {
	if metadata == nil {
		return h.defaultFallbackRoute(), false
	}

	route := h.decideRouteWithRoutingEngine(metadata)
	requiresSNI := metadata.DstPort == 443 && metadata.HostName == ""

	logger.Debug(fmt.Sprintf("Route decision: %v (requiresSNI: %v)", route, requiresSNI))

	return route, requiresSNI
}

func (h *DefaultTLSHandler) decideRouteWithRoutingEngine(metadata *M.Metadata) ProxyRoute {
	if metadata == nil {
		return h.defaultFallbackRoute()
	}

	routingCfg := h.buildRoutingRulesConfig()
	routeCtx := &routing.MatchContext{
		Domain: strings.ToLower(strings.TrimSpace(metadata.HostName)),
	}
	if metadata.DstIP.IsValid() && !metadata.DstIP.IsUnspecified() {
		routeCtx.IP = metadata.DstIP.String()
	}

	decision, err := routing.DecideRoute(routingCfg, routeCtx)
	if err != nil {
		logger.Warn(fmt.Sprintf("routing.DecideRoute failed, fallback route used: %v", err))
		return h.defaultFallbackRoute()
	}

	switch decision {
	case routing.RouteToAliang:
		return RouteToALiang
	case routing.RouteToSocks:
		return RouteToLocalProxy
	default:
		return h.defaultFallbackRoute()
	}
}

func (h *DefaultTLSHandler) buildRoutingRulesConfig() *model.RoutingRulesConfig {
	now := time.Now()
	rc := model.NewRoutingRulesConfig()

	switchStatus := routing.GetSwitchManager().GetStatus()
	rc.Settings.AliangEnabled = switchStatus.AliangEnabled
	rc.Settings.SocksEnabled = switchStatus.SocksEnabled
	rc.Settings.GeoIPEnabled = switchStatus.GeoIPEnabled
	rc.Settings.UpdatedAt = now
	rc.UpdatedAt = now

	cfg := config.GetGlobalConfig()
	if cfg == nil {
		return rc
	}

	allowlist := cfg.EffectiveAIAllowlist()
	rules := make([]model.RoutingRule, 0, len(allowlist))
	for i, domain := range allowlist {
		normalizedDomain := strings.ToLower(strings.TrimSpace(domain))
		if normalizedDomain == "" {
			continue
		}

		rules = append(rules, model.RoutingRule{
			ID:        fmt.Sprintf("aliang_allowlist_%d", i),
			Type:      model.RuleTypeDomain,
			Condition: normalizedDomain,
			Enabled:   true,
			CreatedAt: now,
			UpdatedAt: now,
		})
	}

	rc.Aliang.Rules = rules
	rc.Aliang.Count = len(rules)
	rc.Aliang.UpdatedAt = now

	return rc
}

func (h *DefaultTLSHandler) defaultFallbackRoute() ProxyRoute {
	cfg := config.GetGlobalConfig()
	switchStatus := routing.GetSwitchManager().GetStatus()
	if cfg != nil && cfg.EffectiveDefaultProxy() == "socks" && switchStatus.SocksEnabled {
		return RouteToLocalProxy
	}
	return RouteDirect
}
