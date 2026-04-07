package tls

import (
	"bytes"
	"context"
	"io"
	"net"
	"strings"
	"sync"
	"testing"
	"time"

	user "aliang.one/nursorgate/processor/auth"
)

type scriptedCaptureConn struct {
	mu         sync.Mutex
	chunks     [][]byte
	index      int
	offset     int
	writes     bytes.Buffer
	closeRead  bool
	closeWrite bool
}

type relayDummyAddr string

func (a relayDummyAddr) Network() string { return "tcp" }
func (a relayDummyAddr) String() string  { return string(a) }

func (c *scriptedCaptureConn) Read(p []byte) (int, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.index >= len(c.chunks) {
		return 0, io.EOF
	}
	chunk := c.chunks[c.index]
	n := copy(p, chunk[c.offset:])
	c.offset += n
	if c.offset >= len(chunk) {
		c.index++
		c.offset = 0
	}
	return n, nil
}

func (c *scriptedCaptureConn) Write(p []byte) (int, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.writes.Write(p)
}

func (c *scriptedCaptureConn) Close() error                     { return nil }
func (c *scriptedCaptureConn) LocalAddr() net.Addr              { return relayDummyAddr("local") }
func (c *scriptedCaptureConn) RemoteAddr() net.Addr             { return relayDummyAddr("remote") }
func (c *scriptedCaptureConn) SetDeadline(time.Time) error      { return nil }
func (c *scriptedCaptureConn) SetReadDeadline(time.Time) error  { return nil }
func (c *scriptedCaptureConn) SetWriteDeadline(time.Time) error { return nil }
func (c *scriptedCaptureConn) CloseRead() error {
	c.closeRead = true
	return nil
}
func (c *scriptedCaptureConn) CloseWrite() error {
	c.closeWrite = true
	return nil
}

func (c *scriptedCaptureConn) WrittenString() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.writes.String()
}

func TestRelayHTTP1_InjectsAuthorizationHeaderForMultipleRequestsOnSameConnection(t *testing.T) {
	previous := user.GetCurrentUserInfo()
	user.SetCurrentUserInfo(&user.UserInfo{
		AccessToken: "relay-token",
		TokenType:   "Bearer",
	})
	t.Cleanup(func() {
		user.SetCurrentUserInfo(previous)
	})

	clientStream := strings.Join([]string{
		"POST /one HTTP/1.1\r\nHost: example.com\r\nContent-Length: 5\r\n\r\nhello",
		"POST /two HTTP/1.1\r\nHost: example.com\r\nContent-Length: 5\r\n\r\nworld",
	}, "")
	serverResponses := strings.Join([]string{
		"HTTP/1.1 200 OK\r\nContent-Length: 2\r\n\r\nok",
		"HTTP/1.1 200 OK\r\nContent-Length: 2\r\n\r\nok",
	}, "")

	clientConn := &scriptedCaptureConn{
		chunks: [][]byte{
			[]byte(clientStream[:30]),
			[]byte(clientStream[30:70]),
			[]byte(clientStream[70:]),
		},
	}
	remoteConn := &scriptedCaptureConn{
		chunks: [][]byte{
			[]byte(serverResponses[:25]),
			[]byte(serverResponses[25:]),
		},
	}

	stats, err := RelayHTTP1(context.Background(), clientConn, remoteConn)
	if err != nil {
		t.Fatalf("RelayHTTP1() error = %v", err)
	}
	if stats.ClientToServerByte == 0 || stats.ServerToClientByte == 0 {
		t.Fatalf("expected non-zero relay stats, got %#v", stats)
	}

	requests := remoteConn.WrittenString()
	if got := strings.Count(strings.ToLower(requests), "authorization-inner: bearer relay-token\r\n"); got != 2 {
		t.Fatalf("authorization-inner count = %d, want 2\nrequests:\n%s", got, requests)
	}
	if !strings.Contains(clientConn.WrittenString(), "HTTP/1.1 200 OK\r\nContent-Length: 2\r\n\r\nok") {
		t.Fatalf("client did not receive proxied responses: %q", clientConn.WrittenString())
	}
}
