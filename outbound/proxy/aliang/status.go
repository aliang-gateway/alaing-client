package aliang

import (
	"fmt"
	"sync"
	"time"
)

const (
	LinkStateUnknown      = "unknown"
	LinkStateConnecting   = "connecting"
	LinkStateConnected    = "connected"
	LinkStateDegraded     = "degraded"
	LinkStateDisconnected = "disconnected"

	defaultLatencyWarningThreshold = 800 * time.Millisecond
)

// LinkStatusSnapshot is the serialized view consumed by the HTTP API and UI.
type LinkStatusSnapshot struct {
	ServerAddr         string `json:"server_addr"`
	State              string `json:"state"`
	LatencyMS          int64  `json:"latency_ms"`
	LastError          string `json:"last_error"`
	LastCheckedAt      int64  `json:"last_checked_at"`
	LastConnectedAt    int64  `json:"last_connected_at"`
	ConsecutiveFailure int    `json:"consecutive_failures"`
}

type linkStatusTracker struct {
	mu               sync.RWMutex
	serverAddr       string
	latencyThreshold time.Duration
	snapshot         LinkStatusSnapshot
}

func newLinkStatusTracker(serverAddr string, latencyThreshold time.Duration) *linkStatusTracker {
	if latencyThreshold <= 0 {
		latencyThreshold = defaultLatencyWarningThreshold
	}

	return &linkStatusTracker{
		serverAddr:       serverAddr,
		latencyThreshold: latencyThreshold,
		snapshot: LinkStatusSnapshot{
			ServerAddr: serverAddr,
			State:      LinkStateUnknown,
		},
	}
}

func (t *linkStatusTracker) markConnecting() {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.snapshot.ServerAddr = t.serverAddr
	t.snapshot.State = LinkStateConnecting
	t.snapshot.LastCheckedAt = time.Now().UnixMilli()
	t.snapshot.LastError = ""
}

func (t *linkStatusTracker) markSuccess(latency time.Duration) {
	t.mu.Lock()
	defer t.mu.Unlock()

	state := LinkStateConnected
	if latency >= t.latencyThreshold {
		state = LinkStateDegraded
	}

	now := time.Now().UnixMilli()
	t.snapshot.ServerAddr = t.serverAddr
	t.snapshot.State = state
	t.snapshot.LatencyMS = latency.Milliseconds()
	t.snapshot.LastError = ""
	t.snapshot.LastCheckedAt = now
	t.snapshot.LastConnectedAt = now
	t.snapshot.ConsecutiveFailure = 0
}

func (t *linkStatusTracker) markFailure(err error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.snapshot.ServerAddr = t.serverAddr
	t.snapshot.State = LinkStateDisconnected
	t.snapshot.LatencyMS = 0
	t.snapshot.LastCheckedAt = time.Now().UnixMilli()
	t.snapshot.ConsecutiveFailure++
	if err != nil {
		t.snapshot.LastError = err.Error()
	} else {
		t.snapshot.LastError = "link probe failed"
	}
}

func (t *linkStatusTracker) markReused() {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.snapshot.State == LinkStateUnknown || t.snapshot.State == LinkStateDisconnected {
		t.snapshot.State = LinkStateConnected
	}
	t.snapshot.ServerAddr = t.serverAddr
	t.snapshot.LastCheckedAt = time.Now().UnixMilli()
	t.snapshot.LastError = ""
}

func (t *linkStatusTracker) snapshotMap() map[string]interface{} {
	t.mu.RLock()
	defer t.mu.RUnlock()

	snapshot := t.snapshot
	if snapshot.ServerAddr == "" {
		snapshot.ServerAddr = t.serverAddr
	}

	return map[string]interface{}{
		"server_addr":            snapshot.ServerAddr,
		"state":                  snapshot.State,
		"latency_ms":             snapshot.LatencyMS,
		"last_error":             snapshot.LastError,
		"last_checked_at":        snapshot.LastCheckedAt,
		"last_connected_at":      snapshot.LastConnectedAt,
		"consecutive_failures":   snapshot.ConsecutiveFailure,
		"high_latency_threshold": t.latencyThreshold.Milliseconds(),
	}
}

func unavailableLinkStatus(serverAddr string, err error) map[string]interface{} {
	message := "aliang outbound is unavailable"
	if err != nil {
		message = err.Error()
	}

	return map[string]interface{}{
		"server_addr":            serverAddr,
		"state":                  LinkStateDisconnected,
		"latency_ms":             int64(0),
		"last_error":             message,
		"last_checked_at":        time.Now().UnixMilli(),
		"last_connected_at":      int64(0),
		"consecutive_failures":   0,
		"high_latency_threshold": defaultLatencyWarningThreshold.Milliseconds(),
	}
}

func describeProbeFailure(serverAddr string, err error) error {
	if err == nil {
		return fmt.Errorf("mTLS link probe to %s failed", serverAddr)
	}
	return fmt.Errorf("mTLS link probe to %s failed: %w", serverAddr, err)
}
