package tls

import (
	"bytes"
	"io"
	"net"
	"testing"
	"time"

	user "aliang.one/nursorgate/processor/auth"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/hpack"
)

type scriptedConn struct {
	chunks [][]byte
	index  int
	offset int
}

func (c *scriptedConn) Read(p []byte) (int, error) {
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

func (c *scriptedConn) Write(p []byte) (int, error)      { return len(p), nil }
func (c *scriptedConn) Close() error                     { return nil }
func (c *scriptedConn) LocalAddr() net.Addr              { return dummyAddr("local") }
func (c *scriptedConn) RemoteAddr() net.Addr             { return dummyAddr("remote") }
func (c *scriptedConn) SetDeadline(time.Time) error      { return nil }
func (c *scriptedConn) SetReadDeadline(time.Time) error  { return nil }
func (c *scriptedConn) SetWriteDeadline(time.Time) error { return nil }

type dummyAddr string

func (a dummyAddr) Network() string { return "tcp" }
func (a dummyAddr) String() string  { return string(a) }

func TestWatcherWrapConn_HTTP1SplitRequestPreservesBytesAndInjectsHeader(t *testing.T) {
	previous := user.GetCurrentUserInfo()
	user.SetCurrentUserInfo(&user.UserInfo{
		AccessToken: "watcher-token",
		TokenType:   "Bearer",
	})
	t.Cleanup(func() {
		user.SetCurrentUserInfo(previous)
	})
	t.Logf("auth header: %q", user.GetCurrentAuthorizationHeader())

	conn := &scriptedConn{
		chunks: [][]byte{
			[]byte("GE"),
			[]byte("T /demo HTTP/1.1\r\nHos"),
			[]byte("t: example.com\r\n\r\nhello"),
		},
	}

	wrapped := NewWatcherWrapConn(conn)
	got, err := io.ReadAll(wrapped)
	if err != nil {
		t.Fatalf("io.ReadAll() error = %v", err)
	}

	text := string(got)
	if !bytes.HasPrefix(got, []byte("GET /demo HTTP/1.1\r\n")) {
		t.Fatalf("unexpected request line in %q", text)
	}
	if !stringsContains(text, "Host: example.com\r\n") {
		t.Fatalf("missing host header in %q", text)
	}
	if !stringsContains(text, "Authorization-Inner: Bearer watcher-token\r\n") {
		t.Fatalf("missing injected authorization header in %q", text)
	}
	if !bytes.HasSuffix(got, []byte("\r\n\r\nhello")) {
		t.Fatalf("missing body in %q", text)
	}
}

func TestWatcherWrapConn_HTTP2PrefaceAndSettingsSplitNoLossOrDuplication(t *testing.T) {
	var input bytes.Buffer
	input.WriteString(http2.ClientPreface)

	framer := http2.NewFramer(&input, nil)
	if err := framer.WriteSettings(); err != nil {
		t.Fatalf("WriteSettings() error = %v", err)
	}

	raw := input.Bytes()
	conn := &scriptedConn{
		chunks: [][]byte{
			raw[:2],
			raw[2:11],
			raw[11:24],
			raw[24:28],
			raw[28:],
		},
	}

	wrapped := NewWatcherWrapConn(conn)
	got, err := io.ReadAll(wrapped)
	if err != nil {
		t.Fatalf("io.ReadAll() error = %v", err)
	}

	if !bytes.Equal(got, raw) {
		t.Fatalf("HTTP/2 bytes changed\n got: %x\nwant: %x", got, raw)
	}
}

func TestWatcherWrapConn_HTTP2InjectsAuthorizationHeader(t *testing.T) {
	previous := user.GetCurrentUserInfo()
	user.SetCurrentUserInfo(&user.UserInfo{
		AccessToken: "h2-token",
		TokenType:   "Bearer",
	})
	t.Cleanup(func() {
		user.SetCurrentUserInfo(previous)
	})
	if got := user.GetCurrentAuthorizationHeader(); got != "Bearer h2-token" {
		t.Fatalf("GetCurrentAuthorizationHeader() = %q", got)
	}

	var input bytes.Buffer
	input.WriteString(http2.ClientPreface)

	framer := http2.NewFramer(&input, nil)
	if err := framer.WriteSettings(); err != nil {
		t.Fatalf("WriteSettings() error = %v", err)
	}

	var encodedHeaderBlock bytes.Buffer
	encoder := hpack.NewEncoder(&encodedHeaderBlock)
	fields := []hpack.HeaderField{
		{Name: ":method", Value: "GET"},
		{Name: ":path", Value: "/demo"},
		{Name: ":scheme", Value: "https"},
		{Name: ":authority", Value: "example.com"},
	}
	for _, field := range fields {
		if err := encoder.WriteField(field); err != nil {
			t.Fatalf("WriteField() error = %v", err)
		}
	}

	if err := framer.WriteHeaders(http2.HeadersFrameParam{
		StreamID:      1,
		BlockFragment: encodedHeaderBlock.Bytes(),
		EndHeaders:    true,
		EndStream:     true,
	}); err != nil {
		t.Fatalf("WriteHeaders() error = %v", err)
	}

	raw := input.Bytes()
	conn := &scriptedConn{
		chunks: [][]byte{
			raw[:5],
			raw[5:17],
			raw[17:31],
			raw[31:],
		},
	}

	wrapped := NewWatcherWrapConn(conn)
	got, err := io.ReadAll(wrapped)
	if err != nil {
		t.Fatalf("io.ReadAll() error = %v", err)
	}

	reader := bytes.NewReader(got[len(http2.ClientPreface):])
	decodeFramer := http2.NewFramer(nil, reader)

	var headerBlock bytes.Buffer
	var found bool
	for {
		frame, err := decodeFramer.ReadFrame()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("ReadFrame() error = %v", err)
		}

		switch f := frame.(type) {
		case *http2.HeadersFrame:
			headerBlock.Write(f.HeaderBlockFragment())
			if f.HeadersEnded() {
				decoder := hpack.NewDecoder(4096, func(field hpack.HeaderField) {
					if field.Name == "authorization-inner" && field.Value == "Bearer h2-token" {
						found = true
					}
				})
				if _, err := decoder.Write(headerBlock.Bytes()); err != nil {
					t.Fatalf("decoder.Write() error = %v", err)
				}
				headerBlock.Reset()
			}
		case *http2.ContinuationFrame:
			headerBlock.Write(f.HeaderBlockFragment())
			if f.HeadersEnded() {
				decoder := hpack.NewDecoder(4096, func(field hpack.HeaderField) {
					if field.Name == "authorization-inner" && field.Value == "Bearer h2-token" {
						found = true
					}
				})
				if _, err := decoder.Write(headerBlock.Bytes()); err != nil {
					t.Fatalf("decoder.Write() error = %v", err)
				}
				headerBlock.Reset()
			}
		}
	}

	if !found {
		t.Fatalf("authorization-inner header not found in rewritten HTTP/2 frames")
	}
}

func stringsContains(haystack, needle string) bool {
	return bytes.Contains([]byte(haystack), []byte(needle))
}
