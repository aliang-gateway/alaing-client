package tcp

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/netip"
	"net/url"
	"strings"
	"time"

	"nursor.org/nursorgate/common/logger"
	"nursor.org/nursorgate/inbound/tun/dialer"
	M "nursor.org/nursorgate/inbound/tun/metadata"
	"nursor.org/nursorgate/outbound"
	outboundproxy "nursor.org/nursorgate/outbound/proxy"
	"nursor.org/nursorgate/processor/config"
	"nursor.org/nursorgate/processor/rules"
	"nursor.org/nursorgate/processor/statistic"
	watcher "nursor.org/nursorgate/processor/watcher"
)

var reverseLookupAddr = func(ctx context.Context, addr string) ([]string, error) {
	return net.DefaultResolver.LookupAddr(ctx, addr)
}

const (
	applicationPrefetchInitialTimeout = 200 * time.Millisecond
	applicationPrefetchRetryTimeout   = 50 * time.Millisecond
	applicationPrefetchMaxBytes       = 8192
)

const (
	DenyReasonToAliangDisabled     = "toAliang_disabled"
	DenyReasonToAliangUnavailable  = "toAliang_unavailable"
	DenyReasonToSocksDisabled      = "toSocks_disabled"
	DenyReasonToSocksMisconfigured = "toSocks_misconfigured"
	DenyReasonToSocksUnavailable   = "toSocks_unavailable"
	DenyReasonToSocksUnsupported   = "toSocks_unsupported_upstream_type"
	toSocksUpstreamTypeSocks       = "socks"
	toSocksUpstreamTypeHTTP        = "http"
	toSocksBranchName              = "toSocks"
	toAliangBranchName             = "toAliang"
)

type BranchDenyError struct {
	Branch string
	Reason string
	Cause  error
}

func (e *BranchDenyError) Error() string {
	if e == nil {
		return "route denied"
	}
	if e.Cause == nil {
		return fmt.Sprintf("route denied: branch=%s reason=%s", e.Branch, e.Reason)
	}
	return fmt.Sprintf("route denied: branch=%s reason=%s: %v", e.Branch, e.Reason, e.Cause)
}

func (e *BranchDenyError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Cause
}

func newBranchDenyError(branch, reason string, cause error) error {
	return &BranchDenyError{Branch: branch, Reason: reason, Cause: cause}
}

func IsBranchDenyError(err error) bool {
	var denyErr *BranchDenyError
	return errors.As(err, &denyErr)
}

func BranchDenyReason(err error) string {
	var denyErr *BranchDenyError
	if errors.As(err, &denyErr) {
		return denyErr.Reason
	}
	return ""
}

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

	// Create timeout context (使用父 context 而非 Background)
	timeout := time.Duration(DefaultTCPConnectTimeout) * time.Second
	if deadline, ok := ctx.Deadline(); ok {
		timeout = time.Until(deadline)
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
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
		remoteConn, newOriginConn, err = h.handleNonTLS(ctx, originConn, metadata)
	default:
		remoteConn, newOriginConn, err = h.handleNonTLS(ctx, originConn, metadata)
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
	relayStats, err := h.relayManager.Relay(ctx, newOriginConn, trackedRemote, metadata)
	if err == nil {
		statistic.GetDefaultHTTPStatsCollector().RecordConnection(
			metadata,
			relayStats.RequestPayload,
			relayStats.ResponsePayload,
			relayStats.ClientToServerByte,
			relayStats.ServerToClientByte,
			relayStats.StartedAt,
			relayStats.FirstResponseAt,
			relayStats.CompletedAt,
		)
	}

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

func (h *TCPConnectionHandler) handleNonTLS(
	ctx context.Context,
	originConn net.Conn,
	metadata *M.Metadata,
) (remoteConn net.Conn, newOriginConn net.Conn, err error) {
	if metadata != nil && metadata.DstIP.IsValid() && !metadata.DstIP.IsUnspecified() &&
		(IsLoopbackIP(metadata.DstIP) || IsPrivateIP(metadata.DstIP)) {
		metadata.Route = "RouteDirect"
		remote, dialErr := h.dialDirect(ctx, metadata)
		return remote, originConn, dialErr
	}

	bufferedData, sniffErr := prefetchApplicationData(ctx, originConn, applicationPrefetchMaxBytes)
	if sniffErr != nil {
		logger.Debug(fmt.Sprintf("non-TLS prefetch failed: %v", sniffErr))
	}

	newOriginConn = wrapBufferedConn(originConn, bufferedData)

	host, bindingSource, isHTTP := extractHTTPRoutingHost(bufferedData)
	if isHTTP {
		if host != "" {
			ttl := M.DefaultHTTPTTL
			if bindingSource == M.BindingSourceCONNECT {
				ttl = M.DefaultCONNECTTTL
			}
			metadata.SetHostName(host, bindingSource, ttl)
		}

		route, _ := h.tlsHandler.DetermineRouteWithContext(metadata)
		logger.Debug(fmt.Sprintf("HTTP: Route decision for host=%s ip=%s: %v", metadata.HostName, metadata.DstIP, route))
		remote, dialErr := h.dialByRoute(ctx, metadata, route)
		if dialErr != nil {
			return nil, newOriginConn, dialErr
		}
		if route == RouteToALiang {
			return remote, h.wrapAliangHTTPConn(newOriginConn), nil
		}
		return remote, newOriginConn, nil
	}

	h.enrichMetadataFromReverseLookup(ctx, metadata)
	route, _ := h.tlsHandler.DetermineRouteWithContext(metadata)
	logger.Debug(fmt.Sprintf("TCP: Route decision for host=%s ip=%s (non-HTTP payload): %v", metadata.HostName, metadata.DstIP, route))
	remote, dialErr := h.dialByRoute(ctx, metadata, route)
	return remote, newOriginConn, dialErr
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
	if cache == nil {
		// 诊断日志：如果 cache 为 nil，说明 Rule Engine 未初始化
		logger.Debug("TLS: DNS cache not initialized - Rule engine may not be configured. Will extract SNI directly.")
		cacheHit = false
	} else if !metadata.DstIP.IsUnspecified() && metadata.HostName == "" {
		cacheEntries := cache.GetByIP(metadata.DstIP)
		if len(cacheEntries) > 0 {
			// Use the first entry's domain for routing
			cachedEntry := cacheEntries[0]
			// Extract binding source from cache (use first source if multiple exist)
			bindingSource := ""
			if len(cachedEntry.BindingSources) > 0 {
				bindingSource = string(cachedEntry.BindingSources[0])
			}
			metadata.SetHostNameFromCacheEntry(
				cachedEntry.Domain,
				M.BindingSource(bindingSource),
				cachedEntry.CreatedAt,
				cachedEntry.TimeToLive(),
			)
			cacheHit = true
			logger.Debug(fmt.Sprintf("TLS: Found domain in cache for IP %s: %s (hit count: %d)",
				metadata.DstIP, metadata.HostName, cachedEntry.HitCount))
		}
	} else {
		cacheHit = true
		sni = metadata.HostName
	}

	// STEP 2: Only extract SNI if we didn't find domain in cache
	if !cacheHit || metadata.HostName == "" {
		logger.Debug("TLS: Cache miss or empty hostname, extracting SNI from TLS ClientHello")
		sni, sniBuf, err = h.tlsHandler.ExtractSNI(ctx, originConn)

		if err != nil {
			logger.Debug(fmt.Sprintf("SNI extraction error: %v", err))
			sni = ""
		} else if sni != "" {
			// Set hostname with SNI binding source
			metadata.SetHostName(sni, M.BindingSourceSNI, 5*time.Minute)

			// Check if this is a DoH (DNS over HTTPS) provider
			// DoH traffic should be routed directly without proxy interception
			if IsDoHProvider(sni) {
				logger.Info(fmt.Sprintf("[DoH] Detected DoH provider: %s, routing directly", sni))
				// Return nil to allow direct connection (bypass proxy)
				// This means the connection will be handled directly without MITM or proxy routing
				return nil, nil, nil
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

	mitmedSNI := sni
	if cacheHit && sni == "" {
		// If we got domain from cache but not SNI, use cached domain as SNI
		mitmedSNI = metadata.HostName
	}

	return h.resolveTLSRoute(ctx, originConn, metadata, route, wrapped, mitmedSNI)
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

func (h *TCPConnectionHandler) dialByRoute(ctx context.Context, metadata *M.Metadata, route ProxyRoute) (net.Conn, error) {
	switch route {
	case RouteToALiang:
		metadata.Route = "RouteToCursor"
		aliangProxy, err := h.getAliangProxyForExecution()
		if err != nil {
			return nil, err
		}
		return aliangProxy.DialContext(ctx, metadata)
	case RouteToLocalProxy:
		return h.dialViaSocksOrDirect(ctx, metadata)
	default:
		metadata.Route = "RouteDirect"
		return h.dialDirect(ctx, metadata)
	}
}

func (h *TCPConnectionHandler) resolveTLSRoute(
	ctx context.Context,
	originConn net.Conn,
	metadata *M.Metadata,
	route ProxyRoute,
	wrapped net.Conn,
	mitmedSNI string,
) (net.Conn, net.Conn, error) {
	if route == RouteToALiang {
		mitmed, err := h.tlsHandler.PerformMITM(ctx, originConn, mitmedSNI)
		if err != nil {
			return nil, nil, err
		}

		remote, err := h.dialByRoute(ctx, metadata, route)
		if err != nil {
			return nil, nil, err
		}

		return remote, h.wrapAliangHTTPConn(mitmed), nil
	}

	remote, err := h.dialByRoute(ctx, metadata, route)
	if err != nil {
		return nil, nil, err
	}
	return remote, wrapped, nil
}

func (h *TCPConnectionHandler) wrapAliangHTTPConn(conn net.Conn) net.Conn {
	if conn == nil {
		return nil
	}
	return watcher.NewWatcherWrapConn(conn)
}

func (h *TCPConnectionHandler) dialViaSocksOrDirect(ctx context.Context, metadata *M.Metadata) (net.Conn, error) {
	if !isCustomerProxyEnabled() {
		metadata.Route = "RouteDirect"
		return h.dialDirect(ctx, metadata)
	}

	socksProxy, upstreamType, err := h.getToSocksProxyForExecution()
	if err != nil {
		return nil, err
	}
	logger.Debug(fmt.Sprintf("toSocks execution using upstream type: %s", upstreamType))

	metadata.Route = "RouteToSocks"
	return socksProxy.DialContext(ctx, metadata)
}

func isCustomerProxyEnabled() bool {
	cfg := config.GetGlobalConfig()
	if cfg == nil || cfg.Customer == nil || cfg.Customer.Proxy == nil {
		return true
	}
	return cfg.Customer.Proxy.IsEnabled()
}

func (h *TCPConnectionHandler) enrichMetadataFromReverseLookup(ctx context.Context, metadata *M.Metadata) {
	if metadata == nil || metadata.HostName != "" || !metadata.DstIP.IsValid() || metadata.DstIP.IsUnspecified() {
		return
	}
	if IsLoopbackIP(metadata.DstIP) || IsPrivateIP(metadata.DstIP) {
		return
	}

	cache := rules.GetCache()
	if cache != nil {
		cacheEntries := cache.GetByIP(metadata.DstIP)
		if len(cacheEntries) > 0 {
			cachedEntry := cacheEntries[0]
			bindingSource := M.BindingSourceDNS
			if len(cachedEntry.BindingSources) > 0 {
				bindingSource = cachedEntry.BindingSources[0]
			}
			metadata.SetHostNameFromCacheEntry(
				cachedEntry.Domain,
				bindingSource,
				cachedEntry.CreatedAt,
				cachedEntry.TimeToLive(),
			)
			return
		}
	}

	names, err := reverseLookupAddr(ctx, metadata.DstIP.String())
	if err != nil {
		logger.Debug(fmt.Sprintf("reverse lookup failed for %s: %v", metadata.DstIP, err))
		return
	}

	for _, candidate := range names {
		host := normalizeRoutingHost(candidate)
		if host == "" {
			continue
		}
		metadata.SetHostName(host, M.BindingSourceDNS, M.DefaultDNSTTL)
		logger.Debug(fmt.Sprintf("reverse lookup matched %s -> %s", metadata.DstIP, host))
		return
	}
}

func prefetchApplicationData(ctx context.Context, conn net.Conn, maxBytes int) ([]byte, error) {
	if conn == nil || maxBytes <= 0 {
		return nil, nil
	}

	initialDeadline := time.Now().Add(applicationPrefetchInitialTimeout)
	if deadline, ok := ctx.Deadline(); ok && deadline.Before(initialDeadline) {
		initialDeadline = deadline
	}
	if err := conn.SetReadDeadline(initialDeadline); err != nil {
		return nil, err
	}
	defer conn.SetReadDeadline(time.Time{})

	buf := make([]byte, 0, 1024)
	tmp := make([]byte, 1024)

	for len(buf) < maxBytes {
		readSize := len(tmp)
		if remaining := maxBytes - len(buf); remaining < readSize {
			readSize = remaining
		}

		n, err := conn.Read(tmp[:readSize])
		if n > 0 {
			buf = append(buf, tmp[:n]...)
		}
		if err != nil {
			if errors.Is(err, io.EOF) {
				return buf, nil
			}
			var netErr net.Error
			if errors.As(err, &netErr) && netErr.Timeout() {
				return buf, nil
			}
			return buf, err
		}

		if len(buf) == 0 {
			continue
		}
		if hasCompleteHTTPHeaders(buf) || !couldStillBeHTTP(buf) {
			return buf, nil
		}

		nextDeadline := time.Now().Add(applicationPrefetchRetryTimeout)
		if deadline, ok := ctx.Deadline(); ok && deadline.Before(nextDeadline) {
			nextDeadline = deadline
		}
		if err := conn.SetReadDeadline(nextDeadline); err != nil {
			return buf, err
		}
	}

	return buf, nil
}

func wrapBufferedConn(originConn net.Conn, buf []byte) net.Conn {
	if len(buf) == 0 {
		return originConn
	}
	return &WrappedConn{
		Conn: originConn,
		Buf:  buf,
	}
}

func hasCompleteHTTPHeaders(buf []byte) bool {
	return bytes.Contains(buf, []byte("\r\n\r\n")) || bytes.Contains(buf, []byte("\n\n"))
}

func couldStillBeHTTP(buf []byte) bool {
	if len(buf) == 0 {
		return true
	}

	methods := []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS", "CONNECT", "TRACE"}
	upperPrefix := strings.ToUpper(string(buf))
	for _, method := range methods {
		if strings.HasPrefix(method, upperPrefix) || strings.HasPrefix(upperPrefix, method+" ") {
			return true
		}
	}
	return false
}

func extractHTTPRoutingHost(buf []byte) (host string, source M.BindingSource, isHTTP bool) {
	if len(buf) == 0 {
		return "", "", false
	}

	headerSlice := buf
	if idx := bytes.Index(buf, []byte("\r\n\r\n")); idx >= 0 {
		headerSlice = buf[:idx]
	} else if idx := bytes.Index(buf, []byte("\n\n")); idx >= 0 {
		headerSlice = buf[:idx]
	}

	headerText := strings.ReplaceAll(string(headerSlice), "\r\n", "\n")
	lines := strings.Split(headerText, "\n")
	if len(lines) == 0 {
		return "", "", false
	}

	requestLine := strings.TrimSpace(lines[0])
	parts := strings.Fields(requestLine)
	if len(parts) < 2 {
		return "", "", false
	}

	method := strings.ToUpper(strings.TrimSpace(parts[0]))
	if !isSupportedHTTPMethod(method) {
		return "", "", false
	}

	isHTTP = true
	if method == "CONNECT" {
		return normalizeRoutingHost(parts[1]), M.BindingSourceCONNECT, true
	}

	for _, line := range lines[1:] {
		line = strings.TrimSpace(line)
		if line == "" {
			break
		}
		key, value, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		if strings.EqualFold(strings.TrimSpace(key), "host") {
			return normalizeRoutingHost(value), M.BindingSourceHTTP, true
		}
	}

	target := strings.TrimSpace(parts[1])
	if parsedURL, err := url.Parse(target); err == nil && parsedURL.Host != "" {
		return normalizeRoutingHost(parsedURL.Host), M.BindingSourceHTTP, true
	}

	return "", M.BindingSourceHTTP, true
}

func isSupportedHTTPMethod(method string) bool {
	switch method {
	case "GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS", "CONNECT", "TRACE":
		return true
	default:
		return false
	}
}

func normalizeRoutingHost(raw string) string {
	value := strings.ToLower(strings.TrimSpace(raw))
	value = strings.TrimSuffix(value, ".")
	if value == "" {
		return ""
	}

	if parsedURL, err := url.Parse(value); err == nil && parsedURL.Host != "" {
		value = parsedURL.Host
	}

	if host, port, err := net.SplitHostPort(value); err == nil {
		_ = port
		value = host
	} else if strings.HasPrefix(value, "[") && strings.Contains(value, "]") {
		value = strings.TrimPrefix(strings.SplitN(value, "]", 2)[0], "[")
	} else if host, port, ok := strings.Cut(value, ":"); ok && port != "" && isAllDigits(port) {
		value = host
	}

	value = strings.Trim(value, "[]")
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}

	if ip, err := netip.ParseAddr(value); err == nil && ip.IsValid() {
		return ""
	}

	if !IsValidHostname(value) {
		return ""
	}

	return value
}

func isAllDigits(value string) bool {
	if value == "" {
		return false
	}
	for _, r := range value {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func (h *TCPConnectionHandler) getAliangProxyForExecution() (outboundproxy.Proxy, error) {
	canonical := config.GetRoutingApplyStore().ActiveCanonicalSchema()
	if canonical != nil && !canonical.Egress.ToAliang.Enabled {
		return nil, newBranchDenyError(toAliangBranchName, DenyReasonToAliangDisabled, nil)
	}

	aliangProxy, err := outbound.GetRegistry().GetAliang()
	if err != nil {
		return nil, newBranchDenyError(toAliangBranchName, DenyReasonToAliangUnavailable, err)
	}

	return aliangProxy, nil
}

func (h *TCPConnectionHandler) getToSocksProxyForExecution() (outboundproxy.Proxy, string, error) {
	canonical := config.GetRoutingApplyStore().ActiveCanonicalSchema()
	upstreamType := toSocksUpstreamTypeSocks

	if canonical != nil {
		if !canonical.Egress.ToSocks.Enabled {
			return nil, "", newBranchDenyError(toSocksBranchName, DenyReasonToSocksDisabled, nil)
		}

		upstreamType = canonical.Egress.ToSocks.Upstream.Type
		if upstreamType == "" {
			return nil, "", newBranchDenyError(toSocksBranchName, DenyReasonToSocksMisconfigured, nil)
		}
	}

	proxyName := ""
	switch upstreamType {
	case toSocksUpstreamTypeSocks:
		proxyName = "socks"
	case toSocksUpstreamTypeHTTP:
		proxyName = "http"
	default:
		return nil, "", newBranchDenyError(
			toSocksBranchName,
			DenyReasonToSocksUnsupported,
			fmt.Errorf("upstream.type=%s", upstreamType),
		)
	}

	socksProxy, err := outbound.GetRegistry().Get(proxyName)
	if err != nil {
		return nil, "", newBranchDenyError(toSocksBranchName, DenyReasonToSocksUnavailable, err)
	}

	return socksProxy, upstreamType, nil
}
