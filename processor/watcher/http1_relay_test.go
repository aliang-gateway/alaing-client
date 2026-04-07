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
	mu           sync.Mutex
	chunks       [][]byte
	index        int
	offset       int
	writes       bytes.Buffer
	closeRead    bool
	closeWrite   bool
	blockOnEmpty bool
	closedCh     chan struct{}
}

type relayDummyAddr string

func (a relayDummyAddr) Network() string { return "tcp" }
func (a relayDummyAddr) String() string  { return string(a) }

func (c *scriptedCaptureConn) Read(p []byte) (int, error) {
	c.mu.Lock()
	if c.index >= len(c.chunks) {
		blockOnEmpty := c.blockOnEmpty
		closedCh := c.closedCh
		c.mu.Unlock()
		if blockOnEmpty {
			if closedCh == nil {
				closedCh = make(chan struct{})
			}
			<-closedCh
			return 0, io.EOF
		}
		return 0, io.EOF
	}
	chunk := c.chunks[c.index]
	n := copy(p, chunk[c.offset:])
	c.offset += n
	if c.offset >= len(chunk) {
		c.index++
		c.offset = 0
	}
	c.mu.Unlock()
	return n, nil
}

func (c *scriptedCaptureConn) Write(p []byte) (int, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.writes.Write(p)
}

func (c *scriptedCaptureConn) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closedCh != nil {
		select {
		case <-c.closedCh:
		default:
			close(c.closedCh)
		}
	}
	return nil
}
func (c *scriptedCaptureConn) LocalAddr() net.Addr              { return relayDummyAddr("local") }
func (c *scriptedCaptureConn) RemoteAddr() net.Addr             { return relayDummyAddr("remote") }
func (c *scriptedCaptureConn) SetDeadline(time.Time) error      { return nil }
func (c *scriptedCaptureConn) SetReadDeadline(time.Time) error  { return nil }
func (c *scriptedCaptureConn) SetWriteDeadline(time.Time) error { return nil }
func (c *scriptedCaptureConn) CloseRead() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.closeRead = true
	if c.closedCh != nil {
		select {
		case <-c.closedCh:
		default:
			close(c.closedCh)
		}
	}
	return nil
}
func (c *scriptedCaptureConn) CloseWrite() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.closeWrite = true
	if c.closedCh != nil {
		select {
		case <-c.closedCh:
		default:
			close(c.closedCh)
		}
	}
	return nil
}

func (c *scriptedCaptureConn) WrittenString() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.writes.String()
}

type scriptedRemoteHTTPConn struct {
	mu             sync.Mutex
	cond           *sync.Cond
	responses      [][]byte
	responseIndex  int
	readBuf        bytes.Buffer
	writes         bytes.Buffer
	closeRead      bool
	closeWrite     bool
	closed         bool
	closeAfterLast bool
}

func newScriptedRemoteHTTPConn(responses [][]byte, closeAfterLast bool) *scriptedRemoteHTTPConn {
	conn := &scriptedRemoteHTTPConn{
		responses:      responses,
		closeAfterLast: closeAfterLast,
	}
	conn.cond = sync.NewCond(&conn.mu)
	return conn
}

func (c *scriptedRemoteHTTPConn) Read(p []byte) (int, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for c.readBuf.Len() == 0 && !c.closed {
		c.cond.Wait()
	}
	if c.readBuf.Len() == 0 && c.closed {
		return 0, io.EOF
	}
	return c.readBuf.Read(p)
}

func (c *scriptedRemoteHTTPConn) Write(p []byte) (int, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return 0, io.EOF
	}
	n, err := c.writes.Write(p)
	if err != nil {
		return n, err
	}
	if c.responseIndex < len(c.responses) {
		_, _ = c.readBuf.Write(c.responses[c.responseIndex])
		c.responseIndex++
		if c.responseIndex >= len(c.responses) && c.closeAfterLast {
			c.closed = true
		}
		c.cond.Broadcast()
	}
	return n, nil
}

func (c *scriptedRemoteHTTPConn) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.closed = true
	c.cond.Broadcast()
	return nil
}

func (c *scriptedRemoteHTTPConn) LocalAddr() net.Addr              { return relayDummyAddr("remote-local") }
func (c *scriptedRemoteHTTPConn) RemoteAddr() net.Addr             { return relayDummyAddr("remote-remote") }
func (c *scriptedRemoteHTTPConn) SetDeadline(time.Time) error      { return nil }
func (c *scriptedRemoteHTTPConn) SetReadDeadline(time.Time) error  { return nil }
func (c *scriptedRemoteHTTPConn) SetWriteDeadline(time.Time) error { return nil }
func (c *scriptedRemoteHTTPConn) CloseRead() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.closeRead = true
	c.closed = true
	c.cond.Broadcast()
	return nil
}
func (c *scriptedRemoteHTTPConn) CloseWrite() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.closeWrite = true
	return nil
}
func (c *scriptedRemoteHTTPConn) WrittenString() string {
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
	clientConn := &scriptedCaptureConn{
		chunks: [][]byte{
			[]byte(clientStream[:30]),
			[]byte(clientStream[30:70]),
			[]byte(clientStream[70:]),
		},
	}
	remoteConn := newScriptedRemoteHTTPConn([][]byte{
		[]byte("HTTP/1.1 200 OK\r\nContent-Length: 2\r\n\r\nok"),
		[]byte("HTTP/1.1 200 OK\r\nContent-Length: 2\r\n\r\nok"),
	}, false)

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

func TestRelayHTTP1_ReturnsWhenRemoteClosesDuringIdleKeepAlive(t *testing.T) {
	previous := user.GetCurrentUserInfo()
	user.SetCurrentUserInfo(&user.UserInfo{
		AccessToken: "relay-token",
		TokenType:   "Bearer",
	})
	t.Cleanup(func() {
		user.SetCurrentUserInfo(previous)
	})

	clientConn := &scriptedCaptureConn{
		chunks:       [][]byte{[]byte("POST /one HTTP/1.1\r\nHost: example.com\r\nContent-Length: 5\r\n\r\nhello")},
		blockOnEmpty: true,
		closedCh:     make(chan struct{}),
	}
	remoteConn := newScriptedRemoteHTTPConn([][]byte{
		[]byte("HTTP/1.1 200 OK\r\nContent-Length: 2\r\n\r\nok"),
	}, true)

	done := make(chan error, 1)
	go func() {
		_, err := RelayHTTP1(context.Background(), clientConn, remoteConn)
		done <- err
	}()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("RelayHTTP1() error = %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("RelayHTTP1() did not return after remote idle close")
	}
}
