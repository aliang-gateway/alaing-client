package tcp

import (
	"testing"

	"nursor.org/nursorgate/outbound"
	httpproxy "nursor.org/nursorgate/outbound/proxy/http"
	"nursor.org/nursorgate/processor/config"
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
