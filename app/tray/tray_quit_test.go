package tray

import "testing"

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
