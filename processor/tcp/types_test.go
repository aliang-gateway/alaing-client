package tcp

import "testing"

func TestWrappedConn_ForwardsHalfClose(t *testing.T) {
	conn := newRecordingRelayConn(nil)
	wrapped := &WrappedConn{Conn: conn}

	if err := wrapped.CloseRead(); err != nil {
		t.Fatalf("CloseRead() error = %v", err)
	}
	if err := wrapped.CloseWrite(); err != nil {
		t.Fatalf("CloseWrite() error = %v", err)
	}
	if conn.closeReadCount != 1 {
		t.Fatalf("underlying CloseRead count = %d, want 1", conn.closeReadCount)
	}
	if conn.closeWriteCount != 1 {
		t.Fatalf("underlying CloseWrite count = %d, want 1", conn.closeWriteCount)
	}
}
