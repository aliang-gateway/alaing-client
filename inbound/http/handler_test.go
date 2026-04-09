package http

import (
	"bufio"
	"context"
	"io"
	"net"
	stdhttp "net/http"
	"testing"

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
			"POST http://example.com/upload HTTP/1.1\r\n"+
				"Host: example.com\r\n"+
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
	if got, want := handler.req.RequestURI, "http://example.com/upload"; got != want {
		t.Fatalf("unexpected request uri: got %q want %q", got, want)
	}
	if got, want := handler.req.Host, "example.com"; got != want {
		t.Fatalf("unexpected host: got %q want %q", got, want)
	}
	if got, want := string(handler.body), "hello"; got != want {
		t.Fatalf("unexpected body: got %q want %q", got, want)
	}
	if handler.metadata == nil {
		t.Fatal("expected metadata to be forwarded")
	}
	if got, want := handler.metadata.HostName, "example.com"; got != want {
		t.Fatalf("unexpected metadata hostname: got %q want %q", got, want)
	}
	if got, want := handler.metadata.DstPort, uint16(80); got != want {
		t.Fatalf("unexpected metadata dst port: got %d want %d", got, want)
	}
}
