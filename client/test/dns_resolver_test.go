package test

import (
	"context"
	"testing"
	"time"

	"nursor.org/nursorgate/client/server/tun/proxy"
	"nursor.org/nursorgate/client/server/tun/tunnel"
)

// TestDNSResolver_Google 使用直连 dialer 通过 8.8.8.8 查询 www.google.com
func TestDNSResolver_Google(t *testing.T) {
	d := proxy.NewDirect()
	r := tunnel.NewDNSResolver("8.8.8.8:53", d, 5*time.Second, 5*time.Minute)

	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()

	ips, err := r.LookupA(ctx, "www.google.com")
	if err != nil {
		t.Fatalf("dns lookup failed: %v", err)
	}
	if len(ips) == 0 {
		t.Fatalf("no A records returned for www.google.com")
	}
	t.Logf("got %d A records for www.google.com: %+v", len(ips), ips)
}
