package http

import (
	"bufio"
	"context"
	"io"
	"net"
	stdhttp "net/http"
	"net/netip"
	"net/url"
	"testing"
	"time"

	M "aliang.one/nursorgate/inbound/tun/metadata"
	"aliang.one/nursorgate/processor/tcp"
)

type captureTCPHandler struct {
	req      *stdhttp.Request
	body     []byte
	metadata *M.Metadata
}

func (h *captureTCPHandler) Handle(_ context.Context, conn net.Conn, metadata *M.Metadata) error {
	defer conn.Close()

	reader := bufio.NewReader(conn)
	req, err := stdhttp.ReadRequest(reader)
	if err != nil {
		return err
	}
	defer req.Body.Close()

	body, err := io.ReadAll(req.Body)
	if err != nil {
		return err
	}

	h.req = req
	h.body = body
	h.metadata = metadata
	return nil
}

func TestHandleHttpConnectionReplaysNonCONNECTRequest(t *testing.T) {
	tcp.ResetHandler()
	defer tcp.ResetHandler()

	handler := &captureTCPHandler{}
	tcp.SetHandler(handler)

	serverConn, clientConn := net.Pipe()
	defer clientConn.Close()

	writeErrCh := make(chan error, 1)
	go func() {
		_, err := io.WriteString(clientConn,
			"POST /upload HTTP/1.1\r\n"+
				"Host: 127.0.0.1:56432\r\n"+
				"User-Agent: replay-test\r\n"+
				"Content-Length: 5\r\n"+
				"\r\n"+
				"hello",
		)
		writeErrCh <- err
	}()

	reader := bufio.NewReader(serverConn)
	req, err := stdhttp.ReadRequest(reader)
	if err != nil {
		t.Fatalf("read initial request failed: %v", err)
	}

	HandleHttpConnection(serverConn, reader, req)

	if err := <-writeErrCh; err != nil {
		t.Fatalf("write request failed: %v", err)
	}

	if handler.req == nil {
		t.Fatal("expected downstream handler request to be captured")
	}
	if handler.req.Method != stdhttp.MethodPost {
		t.Fatalf("unexpected method: got %q want %q", handler.req.Method, stdhttp.MethodPost)
	}
	if got, want := handler.req.RequestURI, "/upload"; got != want {
		t.Fatalf("unexpected request uri: got %q want %q", got, want)
	}
	if got, want := handler.req.Host, "127.0.0.1:56432"; got != want {
		t.Fatalf("unexpected host: got %q want %q", got, want)
	}
	if got, want := string(handler.body), "hello"; got != want {
		t.Fatalf("unexpected body: got %q want %q", got, want)
	}
	if handler.metadata == nil {
		t.Fatal("expected metadata to be forwarded")
	}
	if got, want := handler.metadata.HostName, "127.0.0.1"; got != want {
		t.Fatalf("unexpected metadata hostname: got %q want %q", got, want)
	}
	if got, want := handler.metadata.DstPort, uint16(56432); got != want {
		t.Fatalf("unexpected metadata dst port: got %d want %d", got, want)
	}
}

type tcpAddrConn struct {
	net.Conn
	local  *net.TCPAddr
	remote *net.TCPAddr
}

func (c *tcpAddrConn) LocalAddr() net.Addr {
	return c.local
}

func (c *tcpAddrConn) RemoteAddr() net.Addr {
	return c.remote
}

func (c *tcpAddrConn) SetDeadline(time.Time) error {
	return nil
}

func (c *tcpAddrConn) SetReadDeadline(time.Time) error {
	return nil
}

func (c *tcpAddrConn) SetWriteDeadline(time.Time) error {
	return nil
}

func TestExtractMetadataFromHTTP_AllowsMissingHostByFallingBackToLocalAddr(t *testing.T) {
	req := &stdhttp.Request{
		Method: stdhttp.MethodGet,
		URL:    mustParseURL(t, "/health"),
		Header: make(stdhttp.Header),
	}
	conn := &tcpAddrConn{
		Conn:   nil,
		local:  &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 56432},
		remote: &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 41000},
	}

	metadata, err := extractMetadataFromHTTP(req, conn)
	if err != nil {
		t.Fatalf("extract metadata failed: %v", err)
	}
	if got, want := metadata.DstIP, netip.MustParseAddr("127.0.0.1"); got != want {
		t.Fatalf("unexpected dst ip: got %s want %s", got, want)
	}
	if got, want := metadata.DstPort, uint16(56432); got != want {
		t.Fatalf("unexpected dst port: got %d want %d", got, want)
	}
}

func TestExtractMetadataFromHTTP_RejectsExplicitNonLoopbackHost(t *testing.T) {
	req := &stdhttp.Request{
		Method: stdhttp.MethodGet,
		URL:    mustParseURL(t, "http://example.com/path"),
		Header: make(stdhttp.Header),
		Host:   "example.com",
	}
	conn := &tcpAddrConn{
		Conn:   nil,
		local:  &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 56432},
		remote: &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 41000},
	}

	_, err := extractMetadataFromHTTP(req, conn)
	if err == nil {
		t.Fatal("expected explicit non-loopback host to be rejected")
	}
}

func mustParseURL(t *testing.T, raw string) *url.URL {
	t.Helper()
	parsed, err := url.Parse(raw)
	if err != nil {
		t.Fatalf("parse url failed: %v", err)
	}
	return parsed
}
