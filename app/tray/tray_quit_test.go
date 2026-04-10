package tray

import "testing"

func TestTrayModeDisplayName(t *testing.T) {
	testCases := []struct {
		mode string
		want string
	}{
		{mode: "http", want: "Regular Mode"},
		{mode: "tun", want: "Deep Mode"},
		{mode: "unknown", want: "Unknown Mode"},
		{mode: "", want: "Unknown Mode"},
	}

	for _, tc := range testCases {
		if got := trayModeDisplayName(tc.mode); got != tc.want {
			t.Fatalf("trayModeDisplayName(%q) = %q, want %q", tc.mode, got, tc.want)
		}
	}
}

func TestIsAcceptableQuitProxyStopResult(t *testing.T) {
	testCases := []struct {
		name   string
		result map[string]interface{}
		want   bool
	}{
		{
			name: "success",
			result: map[string]interface{}{
				"status": "success",
			},
			want: true,
		},
		{
			name: "already stopped",
			result: map[string]interface{}{
				"status": "failed",
				"error":  "not_running",
			},
			want: true,
		},
		{
			name: "other failure",
			result: map[string]interface{}{
				"status": "failed",
				"error":  "stop_failed",
			},
			want: false,
		},
		{
			name:   "nil result",
			result: nil,
			want:   false,
		},
	}

	for _, tc := range testCases {
		if got := isAcceptableQuitProxyStopResult(tc.result); got != tc.want {
			t.Fatalf("%s: isAcceptableQuitProxyStopResult() = %v, want %v", tc.name, got, tc.want)
		}
	}
}
