package tcp

import (
	"context"
	"net"
	"net/netip"
	"testing"
	"time"

	M "aliang.one/nursorgate/inbound/tun/metadata"
	"aliang.one/nursorgate/outbound"
	httpproxy "aliang.one/nursorgate/outbound/proxy/http"
	"aliang.one/nursorgate/processor/config"
	"aliang.one/nursorgate/processor/routing"
)

func TestTCPHandler_ToSocksHTTPType_UsesHTTPProxy(t *testing.T) {
	config.ResetRoutingApplyStoreForTest()
	t.Cleanup(config.ResetRoutingApplyStoreForTest)

	registry := outbound.GetRegistry()
	registry.Clear()
	t.Cleanup(registry.Clear)

	httpProxy, err := httpproxy.NewHTTP("127.0.0.1:8080", "", "")
	if err != nil {
		t.Fatalf("NewHTTP() error = %v", err)
	}
	if err := registry.Register("http", httpProxy); err != nil {
		t.Fatalf("registry.Register(http) error = %v", err)
	}

	raw := []byte(`{
		"version": 1,
		"ingress": {"mode": "tun"},
		"egress": {
			"direct": {"enabled": true},
			"toAliang": {"enabled": true},
			"toSocks": {"enabled": true, "upstream": {"type": "http"}}
		},
		"routing": {"rules": [], "default_egress": "direct"}
	}`)

	if _, err := config.GetRoutingApplyStore().Apply(raw, func(canonical *config.CanonicalRoutingSchema) (any, error) {
		return canonical, nil
	}); err != nil {
		t.Fatalf("Apply() error = %v", err)
	}

	h := &TCPConnectionHandler{}
	gotProxy, gotType, err := h.getToSocksProxyForExecution()
	if err != nil {
		t.Fatalf("getToSocksProxyForExecution() error = %v", err)
	}
	if gotType != "http" {
		t.Fatalf("upstream type = %q, want %q", gotType, "http")
	}
	if gotProxy.Addr() != httpProxy.Addr() {
		t.Fatalf("proxy addr = %q, want %q", gotProxy.Addr(), httpProxy.Addr())
	}
}

func TestTCPHandler_ToAliangDisabledDeny_ReturnsStableReason(t *testing.T) {
	config.ResetRoutingApplyStoreForTest()
	t.Cleanup(config.ResetRoutingApplyStoreForTest)

	registry := outbound.GetRegistry()
	registry.Clear()
	t.Cleanup(registry.Clear)

	raw := []byte(`{
		"version": 1,
		"ingress": {"mode": "tun"},
		"egress": {
			"direct": {"enabled": true},
			"toAliang": {"enabled": false},
			"toSocks": {"enabled": true, "upstream": {"type": "socks"}}
		},
		"routing": {"rules": [], "default_egress": "direct"}
	}`)

	if _, err := config.GetRoutingApplyStore().Apply(raw, func(canonical *config.CanonicalRoutingSchema) (any, error) {
		return canonical, nil
	}); err != nil {
		t.Fatalf("Apply() error = %v", err)
	}

	h := &TCPConnectionHandler{}
	_, err := h.getAliangProxyForExecution()
	if err == nil {
		t.Fatal("expected deny error, got nil")
	}
	if !IsBranchDenyError(err) {
		t.Fatalf("error should be BranchDenyError, got %T", err)
	}
	if got := BranchDenyReason(err); got != DenyReasonToAliangDisabled {
		t.Fatalf("deny reason = %q, want %q", got, DenyReasonToAliangDisabled)
	}
}

func TestExtractHTTPRoutingHost_ParsesHostHeader(t *testing.T) {
	buf := []byte("GET /chat HTTP/1.1\r\nHost: api.openai.com\r\nUser-Agent: test\r\n\r\n")

	host, source, isHTTP := extractHTTPRoutingHost(buf)
	if !isHTTP {
		t.Fatal("expected HTTP payload to be detected")
	}
	if host != "api.openai.com" {
		t.Fatalf("host = %q, want %q", host, "api.openai.com")
	}
	if source != M.BindingSourceHTTP {
		t.Fatalf("binding source = %q, want %q", source, M.BindingSourceHTTP)
	}
}

func TestExtractHTTPRoutingHost_ParsesConnectAuthority(t *testing.T) {
	buf := []byte("CONNECT claude.ai:443 HTTP/1.1\r\nHost: claude.ai:443\r\n\r\n")

	host, source, isHTTP := extractHTTPRoutingHost(buf)
	if !isHTTP {
		t.Fatal("expected CONNECT payload to be detected as HTTP")
	}
	if host != "claude.ai" {
		t.Fatalf("host = %q, want %q", host, "claude.ai")
	}
	if source != M.BindingSourceCONNECT {
		t.Fatalf("binding source = %q, want %q", source, M.BindingSourceCONNECT)
	}
}

func TestExtractHTTPRoutingHost_NonHTTPPayloadReturnsFalse(t *testing.T) {
	buf := []byte{0x16, 0x03, 0x01, 0x01, 0x7f, 0x01, 0x02}

	host, source, isHTTP := extractHTTPRoutingHost(buf)
	if isHTTP {
		t.Fatalf("unexpected HTTP detection: host=%q source=%q", host, source)
	}
}

func TestTCPHandler_ReverseLookupEnrichesMetadata(t *testing.T) {
	previousLookup := reverseLookupAddr
	reverseLookupAddr = func(ctx context.Context, addr string) ([]string, error) {
		if addr != "8.8.8.8" {
			t.Fatalf("unexpected reverse lookup addr: %s", addr)
		}
		return []string{"dns.google."}, nil
	}
	defer func() {
		reverseLookupAddr = previousLookup
	}()

	metadata := &M.Metadata{
		DstIP: parseTestAddr(t, "8.8.8.8"),
	}

	h := &TCPConnectionHandler{}
	h.enrichMetadataFromReverseLookup(context.Background(), metadata)

	if metadata.HostName != "dns.google" {
		t.Fatalf("hostname = %q, want %q", metadata.HostName, "dns.google")
	}
	if metadata.DNSInfo == nil {
		t.Fatal("expected DNSInfo to be populated")
	}
	if metadata.DNSInfo.BindingSource != M.BindingSourceDNS {
		t.Fatalf("binding source = %q, want %q", metadata.DNSInfo.BindingSource, M.BindingSourceDNS)
	}
	if metadata.DNSInfo.CacheTTL != M.DefaultDNSTTL {
		t.Fatalf("cache ttl = %v, want %v", metadata.DNSInfo.CacheTTL, M.DefaultDNSTTL)
	}
}

func TestPrefetchApplicationData_ReadsHTTPHeaders(t *testing.T) {
	serverConn, clientConn := netPipeForTest(t)
	defer serverConn.Close()
	defer clientConn.Close()

	go func() {
		_, _ = clientConn.Write([]byte("GET / HTTP/1.1\r\nHost: example.com\r\n\r\n"))
	}()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	buf, err := prefetchApplicationData(ctx, serverConn, 4096)
	if err != nil {
		t.Fatalf("prefetchApplicationData() error = %v", err)
	}
	if string(buf) != "GET / HTTP/1.1\r\nHost: example.com\r\n\r\n" {
		t.Fatalf("unexpected prefetched buffer: %q", string(buf))
	}
}

func TestIsCustomerProxyEnabled_DefaultsToTrueWhenUnset(t *testing.T) {
	previousCfg := config.GetGlobalConfig()
	config.SetGlobalConfig(&config.Config{
		Customer: &config.CustomerConfig{
			Proxy: &config.CustomerProxyConfig{Type: "socks5"},
		},
	})
	defer config.SetGlobalConfig(previousCfg)

	if !isCustomerProxyEnabled() {
		t.Fatal("expected customer proxy to default to enabled when flag is unset")
	}
}

func TestIsCustomerProxyEnabled_FalseWhenDisabled(t *testing.T) {
	previousCfg := config.GetGlobalConfig()
	config.SetGlobalConfig(&config.Config{
		Customer: &config.CustomerConfig{
			Proxy: &config.CustomerProxyConfig{Enable: boolPtrMetadata(false), Type: "socks5"},
		},
	})
	defer config.SetGlobalConfig(previousCfg)

	if isCustomerProxyEnabled() {
		t.Fatal("expected customer proxy to be disabled")
	}
}

func TestDefaultTLSHandler_DetermineRouteWithContext_UsesProxyRulesWhenEnabled(t *testing.T) {
	previousCfg := config.GetGlobalConfig()
	config.SetGlobalConfig(&config.Config{
		Customer: &config.CustomerConfig{
			Proxy:      &config.CustomerProxyConfig{Enable: boolPtrMetadata(true), Type: "socks5", Server: "127.0.0.1:1080"},
			ProxyRules: []string{"domain,cursor.com,proxy"},
		},
	})
	defer config.SetGlobalConfig(previousCfg)

	switchManager := routing.GetSwitchManager()
	switchManager.ResetToDefaults()
	switchManager.SetSocksEnabled(true)

	h := NewDefaultTLSHandler()
	route, requiresSNI := h.DetermineRouteWithContext(&M.Metadata{
		HostName: "cursor.com",
		DstPort:  443,
		DstIP:    parseTestAddr(t, "8.8.8.8"),
	})

	if requiresSNI {
		t.Fatal("expected no SNI requirement when hostname is already known")
	}
	if route != RouteToLocalProxy {
		t.Fatalf("route = %v, want %v", route, RouteToLocalProxy)
	}
}

func TestDefaultTLSHandler_DetermineRouteWithContext_ProxyRulesFallbackDirectWhenDisabled(t *testing.T) {
	previousCfg := config.GetGlobalConfig()
	config.SetGlobalConfig(&config.Config{
		Customer: &config.CustomerConfig{
			Proxy:      &config.CustomerProxyConfig{Enable: boolPtrMetadata(false), Type: "socks5", Server: "127.0.0.1:1080"},
			ProxyRules: []string{"domain,cursor.com,proxy"},
		},
	})
	defer config.SetGlobalConfig(previousCfg)

	switchManager := routing.GetSwitchManager()
	switchManager.ResetToDefaults()
	switchManager.SetSocksEnabled(true)

	h := NewDefaultTLSHandler()
	route, _ := h.DetermineRouteWithContext(&M.Metadata{
		HostName: "cursor.com",
		DstPort:  443,
		DstIP:    parseTestAddr(t, "8.8.8.8"),
	})

	if route != RouteDirect {
		t.Fatalf("route = %v, want %v", route, RouteDirect)
	}
}

func parseTestAddr(t *testing.T, raw string) netip.Addr {
	t.Helper()
	addr, err := netip.ParseAddr(raw)
	if err != nil {
		t.Fatalf("ParseAddr(%q) error = %v", raw, err)
	}
	return addr
}

func netPipeForTest(t *testing.T) (net.Conn, net.Conn) {
	t.Helper()
	serverConn, clientConn := net.Pipe()
	return serverConn, clientConn
}

func boolPtrMetadata(v bool) *bool {
	return &v
}
