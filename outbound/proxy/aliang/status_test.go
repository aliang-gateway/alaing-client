package aliang

import (
	"errors"
	"testing"
	"time"
)

func TestLinkStatusTrackerTransitions(t *testing.T) {
	tracker := newLinkStatusTracker("ai-gateway.aliang.one:443", 200*time.Millisecond)

	initial := tracker.snapshotMap()
	if got := initial["state"]; got != LinkStateUnknown {
		t.Fatalf("expected initial state=%s, got %#v", LinkStateUnknown, got)
	}

	tracker.markConnecting()
	connecting := tracker.snapshotMap()
	if got := connecting["state"]; got != LinkStateConnecting {
		t.Fatalf("expected connecting state, got %#v", got)
	}

	tracker.markSuccess(120 * time.Millisecond)
	connected := tracker.snapshotMap()
	if got := connected["state"]; got != LinkStateConnected {
		t.Fatalf("expected connected state, got %#v", got)
	}
	if got := connected["latency_ms"]; got != int64(120) {
		t.Fatalf("expected latency 120ms, got %#v", got)
	}

	tracker.markSuccess(350 * time.Millisecond)
	degraded := tracker.snapshotMap()
	if got := degraded["state"]; got != LinkStateDegraded {
		t.Fatalf("expected degraded state, got %#v", got)
	}

	tracker.markFailure(errors.New("handshake timeout"))
	disconnected := tracker.snapshotMap()
	if got := disconnected["state"]; got != LinkStateDisconnected {
		t.Fatalf("expected disconnected state, got %#v", got)
	}
	if got := disconnected["consecutive_failures"]; got != 1 {
		t.Fatalf("expected consecutive_failures=1, got %#v", got)
	}
	if got := disconnected["last_error"]; got != "handshake timeout" {
		t.Fatalf("expected last_error to be recorded, got %#v", got)
	}
}
