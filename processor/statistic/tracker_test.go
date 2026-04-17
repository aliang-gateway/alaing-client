package statistic

import (
	"net"
	"testing"
	"time"

	M "aliang.one/nursorgate/inbound/tun/metadata"
)

type closeCountingConn struct {
	closeCount int
}

func (c *closeCountingConn) Read(_ []byte) (int, error)       { return 0, nil }
func (c *closeCountingConn) Write(p []byte) (int, error)      { return len(p), nil }
func (c *closeCountingConn) Close() error                     { c.closeCount++; return nil }
func (c *closeCountingConn) LocalAddr() net.Addr              { return &net.TCPAddr{} }
func (c *closeCountingConn) RemoteAddr() net.Addr             { return &net.TCPAddr{} }
func (c *closeCountingConn) SetDeadline(time.Time) error      { return nil }
func (c *closeCountingConn) SetReadDeadline(time.Time) error  { return nil }
func (c *closeCountingConn) SetWriteDeadline(time.Time) error { return nil }

func TestTCPTrackerClose_IsIdempotent(t *testing.T) {
	manager := &Manager{}
	conn := &closeCountingConn{}
	tracked := NewTCPTracker(conn, &M.Metadata{}, manager)

	if err := tracked.Close(); err != nil {
		t.Fatalf("first Close() error = %v", err)
	}
	if err := tracked.Close(); err != nil {
		t.Fatalf("second Close() error = %v", err)
	}
	if conn.closeCount != 1 {
		t.Fatalf("underlying close count = %d, want 1", conn.closeCount)
	}
}
