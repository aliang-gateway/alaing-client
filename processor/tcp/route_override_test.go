package tcp

import (
	"bytes"
	"context"
	"io"
	"net"
	"net/netip"
	"testing"
	"time"

	M "aliang.one/nursorgate/inbound/tun/metadata"
	"aliang.one/nursorgate/outbound"
	outboundproxy "aliang.one/nursorgate/outbound/proxy"
	"aliang.one/nursorgate/outbound/proxy/proto"
	"aliang.one/nursorgate/processor/config"
)

type fakeConn struct {
	reader *bytes.Reader
}

func newFakeConn(payload []byte) *fakeConn {
	return &fakeConn{reader: bytes.NewReader(payload)}
}

func (c *fakeConn) Read(p []byte) (int, error)  { return c.reader.Read(p) }
func (c *fakeConn) Write(p []byte) (int, error) { return len(p), nil }
func (c *fakeConn) Close() error                { return nil }
func (c *fakeConn) LocalAddr() net.Addr {
	return &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 56432}
}
func (c *fakeConn) RemoteAddr() net.Addr {
	return &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 41000}
}
func (c *fakeConn) SetDeadline(time.Time) error      { return nil }
func (c *fakeConn) SetReadDeadline(time.Time) error  { return nil }
func (c *fakeConn) SetWriteDeadline(time.Time) error { return nil }
func (c *fakeConn) CloseRead() error                 { return nil }
func (c *fakeConn) CloseWrite() error                { return nil }

type fakeAliangProxy struct{}

func (p *fakeAliangProxy) DialContext(context.Context, *M.Metadata) (net.Conn, error) {
	return newFakeConn(nil), nil
}

func (p *fakeAliangProxy) DialUDP(*M.Metadata) (net.PacketConn, error) {
	return nil, io.EOF
}

func (p *fakeAliangProxy) Addr() string {
	return "fake-aliang"
}

func (p *fakeAliangProxy) Proto() proto.Proto {
	return proto.Aliang
}

var _ outboundproxy.Proxy = (*fakeAliangProxy)(nil)

func TestDetermineRouteWithContext_ForcesAliangForLocalHTTPProxyPort(t *testing.T) {
	handler := NewDefaultTLSHandler()
	metadata := &M.Metadata{
		Network: M.TCP,
		DstIP:   netip.MustParseAddr("127.0.0.1"),
		DstPort: aliangLocalHTTPProxyPort,
	}

	route, requiresSNI := handler.DetermineRouteWithContext(metadata)
	if route != RouteToALiang {
		t.Fatalf("unexpected route: got %v want %v", route, RouteToALiang)
	}
	if requiresSNI {
		t.Fatal("expected local proxy override to not require SNI")
	}
}

func TestHandleNonTLS_DoesNotShortCircuitLocalHTTPProxyPortToDirect(t *testing.T) {
	config.ResetRoutingApplyStoreForTest()

	registry := outbound.GetRegistry()
	registry.Clear()
	defer registry.Clear()
	if err := registry.Register("aliang", &fakeAliangProxy{}); err != nil {
		t.Fatalf("register fake aliang proxy failed: %v", err)
	}

	handler := NewTCPConnectionHandler(NewDefaultProtocolDetector(), NewDefaultTLSHandler(), nil, nil)
	metadata := &M.Metadata{
		Network: M.TCP,
		DstIP:   netip.MustParseAddr("127.0.0.1"),
		DstPort: aliangLocalHTTPProxyPort,
	}

	remote, _, err := handler.handleNonTLS(context.Background(), newFakeConn(nil), metadata)
	if err != nil {
		t.Fatalf("handleNonTLS failed: %v", err)
	}
	if remote == nil {
		t.Fatal("expected remote conn to be created via aliang override")
	}
	if got, want := metadata.Route, "RouteToALiang"; got != want {
		t.Fatalf("unexpected metadata route: got %q want %q", got, want)
	}
}
