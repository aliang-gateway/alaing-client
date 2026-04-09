package aliang

import (
	"testing"
	"time"
)

func TestLinkStatusTrackerMarkSuccessUsesTCPConnectAsDisplayLatency(t *testing.T) {
	tracker := newLinkStatusTracker("example.com:443", 200*time.Millisecond)

	tracker.markSuccess(ProbeTimings{
		TCPConnect:   40 * time.Millisecond,
		TLSHandshake: 120 * time.Millisecond,
		Total:        160 * time.Millisecond,
	})

	snapshot := tracker.snapshotMap()

	if got := snapshot["latency_ms"]; got != int64(40) {
		t.Fatalf("latency_ms = %#v, want 40", got)
	}
	if got := snapshot["tcp_connect_ms"]; got != int64(40) {
		t.Fatalf("tcp_connect_ms = %#v, want 40", got)
	}
	if got := snapshot["tls_handshake_ms"]; got != int64(120) {
		t.Fatalf("tls_handshake_ms = %#v, want 120", got)
	}
	if got := snapshot["probe_total_ms"]; got != int64(160) {
		t.Fatalf("probe_total_ms = %#v, want 160", got)
	}
	if got := snapshot["state"]; got != LinkStateConnected {
		t.Fatalf("state = %#v, want %q", got, LinkStateConnected)
	}
}

func TestLinkStatusTrackerMarkSuccessUsesTotalLatencyForDegradedState(t *testing.T) {
	tracker := newLinkStatusTracker("example.com:443", 150*time.Millisecond)

	tracker.markSuccess(ProbeTimings{
		TCPConnect:   35 * time.Millisecond,
		TLSHandshake: 170 * time.Millisecond,
		Total:        205 * time.Millisecond,
	})

	snapshot := tracker.snapshotMap()

	if got := snapshot["latency_ms"]; got != int64(35) {
		t.Fatalf("latency_ms = %#v, want 35", got)
	}
	if got := snapshot["state"]; got != LinkStateDegraded {
		t.Fatalf("state = %#v, want %q", got, LinkStateDegraded)
	}
}
