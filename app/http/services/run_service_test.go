package services

import (
	"fmt"
	"testing"
	"time"

	"nursor.org/nursorgate/app/http/models"
	"nursor.org/nursorgate/processor/config"
	"nursor.org/nursorgate/processor/routing"
	"nursor.org/nursorgate/processor/runtime"
)

func seedActiveIngressSnapshot(t *testing.T, mode string) {
	t.Helper()
	config.ResetRoutingApplyStoreForTest()
	raw := []byte(fmt.Sprintf(`{
"version": 1,
"ingress": {"mode": %q},
"egress": {
  "direct": {"enabled": true},
  "toAliang": {"enabled": true},
  "toSocks": {"enabled": true, "upstream": {"type": "socks"}}
},
"routing": {"rules": []}
}`, mode))
	if _, err := config.GetRoutingApplyStore().Apply(raw, func(canonical *config.CanonicalRoutingSchema) (any, error) {
		return routing.CompileRuntimeSnapshot(canonical)
	}); err != nil {
		t.Fatalf("seed routing snapshot failed: %v", err)
	}
}

func resetRunServiceHooksForTest() {
	activeIngressModeResolver = activeIngressModeFromSnapshot
	applyIngressModeUpdater = applyIngressModeToSnapshot
	tunStartRunner = defaultStartTUN
	httpStartRunner = func() {}
	httpStopRunner = func() {}
	tunStopRunner = func() {}
}

func waitForEventCount(events *[]string, expected int, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if len(*events) >= expected {
			return true
		}
		time.Sleep(5 * time.Millisecond)
	}
	return len(*events) >= expected
}

func TestRunServiceStartServiceAlreadyRunning(t *testing.T) {
	defer resetRunServiceHooksForTest()
	seedActiveIngressSnapshot(t, string(models.ModeHTTP))
	runtime.ResetGlobalStartupStateForTest()
	runtime.GetStartupState().SetStatus(runtime.READY)

	runService := NewRunService()
	runService.SetRunning(true)

	result := runService.StartService()

	if status, ok := result["status"].(string); !ok || status != "already_running" {
		t.Fatalf("expected status=already_running, got %#v", result["status"])
	}

	if message, ok := result["message"].(string); !ok || message == "" {
		t.Fatalf("expected non-empty message, got %#v", result["message"])
	}
}

func TestRunServiceCharacterization_StartServiceActivationGuard(t *testing.T) {
	defer resetRunServiceHooksForTest()
	seedActiveIngressSnapshot(t, string(models.ModeHTTP))
	runtime.ResetGlobalStartupStateForTest()
	runtime.GetStartupState().SetStatus(runtime.UNCONFIGURED)

	runService := NewRunService()
	result := runService.StartService()

	if status, ok := result["status"].(string); !ok || status != "failed" {
		t.Fatalf("expected status=failed, got %#v", result["status"])
	}
	if errCode, ok := result["error"].(string); !ok || errCode != "activation_required" {
		t.Fatalf("expected error=activation_required, got %#v", result["error"])
	}
	if runService.IsRunning() {
		t.Fatalf("expected service to remain not running when activation guard rejects start")
	}
}

func TestRunServiceCharacterization_StopServiceWhenNotRunning(t *testing.T) {
	defer resetRunServiceHooksForTest()
	seedActiveIngressSnapshot(t, string(models.ModeHTTP))
	runService := NewRunService()
	runService.SetCurrentMode(string(models.ModeHTTP))
	runService.SetRunning(false)

	result := runService.StopService()

	if status, ok := result["status"].(string); !ok || status != "failed" {
		t.Fatalf("expected status=failed, got %#v", result["status"])
	}
	if errCode, ok := result["error"].(string); !ok || errCode != "not_running" {
		t.Fatalf("expected error=not_running, got %#v", result["error"])
	}
}

func TestRunServiceCharacterization_GetStatusDescriptions(t *testing.T) {
	tests := []struct {
		name            string
		mode            string
		running         bool
		wantStatus      string
		wantDescription string
	}{
		{
			name:            "http running",
			mode:            string(models.ModeHTTP),
			running:         true,
			wantStatus:      "HTTP proxy server is running",
			wantDescription: "HTTP CONNECT proxy mode on port 56432",
		},
		{
			name:            "http idle",
			mode:            string(models.ModeHTTP),
			running:         false,
			wantStatus:      "HTTP mode selected, service not running",
			wantDescription: "HTTP mode is ready, call start to activate",
		},
		{
			name:            "tun running",
			mode:            string(models.ModeTUN),
			running:         true,
			wantStatus:      "TUN service is running",
			wantDescription: "Transparent proxy mode via TUN interface",
		},
		{
			name:            "tun idle",
			mode:            string(models.ModeTUN),
			running:         false,
			wantStatus:      "TUN mode selected, service not running",
			wantDescription: "TUN mode is ready, call start to activate",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer resetRunServiceHooksForTest()
			seedActiveIngressSnapshot(t, tt.mode)
			runService := NewRunService()
			runService.SetCurrentMode(tt.mode)
			runService.SetRunning(tt.running)

			status := runService.GetStatus()

			if got, ok := status["status"].(string); !ok || got != tt.wantStatus {
				t.Fatalf("status text mismatch: got=%#v want=%q", status["status"], tt.wantStatus)
			}
			if got, ok := status["description"].(string); !ok || got != tt.wantDescription {
				t.Fatalf("description mismatch: got=%#v want=%q", status["description"], tt.wantDescription)
			}
		})
	}
}

func TestRunServiceCharacterization_SwitchModeGuards(t *testing.T) {
	t.Run("invalid mode rejected", func(t *testing.T) {
		defer resetRunServiceHooksForTest()
		seedActiveIngressSnapshot(t, string(models.ModeHTTP))
		runService := NewRunService()
		result := runService.SwitchMode("udp")

		if status, ok := result["status"].(string); !ok || status != "failed" {
			t.Fatalf("expected status=failed, got %#v", result["status"])
		}
		if errCode, ok := result["error"].(string); !ok || errCode != "invalid_mode" {
			t.Fatalf("expected error=invalid_mode, got %#v", result["error"])
		}
	})

	t.Run("same mode while running returns already_running", func(t *testing.T) {
		defer resetRunServiceHooksForTest()
		seedActiveIngressSnapshot(t, string(models.ModeTUN))
		runService := NewRunService()
		runService.SetCurrentMode(string(models.ModeTUN))
		runService.SetRunning(true)

		result := runService.SwitchMode(string(models.ModeTUN))

		if status, ok := result["status"].(string); !ok || status != "already_running" {
			t.Fatalf("expected status=already_running, got %#v", result["status"])
		}
		if currentMode, ok := result["current_mode"].(string); !ok || currentMode != string(models.ModeTUN) {
			t.Fatalf("expected current_mode=tun, got %#v", result["current_mode"])
		}
	})

	t.Run("switch to tun while idle does not auto-start", func(t *testing.T) {
		defer resetRunServiceHooksForTest()
		seedActiveIngressSnapshot(t, string(models.ModeHTTP))
		runService := NewRunService()
		runService.SetCurrentMode(string(models.ModeHTTP))
		runService.SetRunning(false)

		result := runService.SwitchMode(string(models.ModeTUN))

		if status, ok := result["status"].(string); !ok || status != "switched" {
			t.Fatalf("expected status=switched, got %#v", result["status"])
		}
		if runService.GetStatus()["is_running"].(bool) {
			t.Fatalf("expected switch-to-tun while idle to keep service stopped")
		}
	})
}

func TestRunServiceHotSwitchHTTPToTUN(t *testing.T) {
	defer resetRunServiceHooksForTest()
	seedActiveIngressSnapshot(t, string(models.ModeHTTP))

	runtime.ResetGlobalStartupStateForTest()
	runtime.GetStartupState().SetStatus(runtime.READY)

	events := make([]string, 0, 8)
	httpStopRunner = func() {
		events = append(events, "http:stop")
	}
	tunStartRunner = func() map[string]string {
		events = append(events, "tun:start")
		return map[string]string{"status": "success", "message": "ok"}
	}

	runService := NewRunService()
	runService.SetCurrentMode(string(models.ModeHTTP))
	runService.SetRunning(true)

	result := runService.SwitchMode(string(models.ModeTUN))
	if status, _ := result["status"].(string); status != "switched" {
		t.Fatalf("expected switched status, got %#v", result)
	}
	if got := runService.GetCurrentMode(); got != string(models.ModeTUN) {
		t.Fatalf("current mode mismatch: got=%q want=%q", got, models.ModeTUN)
	}
	status := runService.GetStatus()
	if current, _ := status["current_mode"].(string); current != string(models.ModeTUN) {
		t.Fatalf("status current_mode mismatch: got=%q want=%q", current, models.ModeTUN)
	}
	if running, _ := status["is_running"].(bool); !running {
		t.Fatalf("expected running=true after successful hot switch, got %#v", status["is_running"])
	}

	if len(events) != 2 {
		t.Fatalf("unexpected event count: got=%d events=%v", len(events), events)
	}
	if events[0] != "http:stop" || events[1] != "tun:start" {
		t.Fatalf("mutual exclusion sequencing violated, events=%v", events)
	}
}

func TestRunServiceHotSwitchFailureRollback(t *testing.T) {
	defer resetRunServiceHooksForTest()
	seedActiveIngressSnapshot(t, string(models.ModeHTTP))

	runtime.ResetGlobalStartupStateForTest()
	runtime.GetStartupState().SetStatus(runtime.READY)

	events := make([]string, 0, 12)
	httpStopRunner = func() {
		events = append(events, "http:stop")
	}
	tunStartRunner = func() map[string]string {
		events = append(events, "tun:start")
		return map[string]string{"status": "failed", "message": "tun failed"}
	}
	httpStartRunner = func() {
		events = append(events, "http:start")
	}

	runService := NewRunService()
	runService.SetCurrentMode(string(models.ModeHTTP))
	runService.SetRunning(true)

	result := runService.SwitchMode(string(models.ModeTUN))
	if status, _ := result["status"].(string); status != "failed" {
		t.Fatalf("expected failed status on activation failure rollback, got %#v", result)
	}
	if errCode, _ := result["error"].(string); errCode != "switch_failed" {
		t.Fatalf("expected error=switch_failed, got %#v", result["error"])
	}

	if got := runService.GetCurrentMode(); got != string(models.ModeHTTP) {
		t.Fatalf("rollback current mode mismatch: got=%q want=%q", got, models.ModeHTTP)
	}
	status := runService.GetStatus()
	if current, _ := status["current_mode"].(string); current != string(models.ModeHTTP) {
		t.Fatalf("rollback status current_mode mismatch: got=%q want=%q", current, models.ModeHTTP)
	}
	if running, _ := status["is_running"].(bool); !running {
		t.Fatalf("expected running=true after rollback, got %#v", status["is_running"])
	}

	if !waitForEventCount(&events, 3, 300*time.Millisecond) {
		t.Fatalf("unexpected event count: got=%d events=%v", len(events), events)
	}
	if len(events) != 3 {
		t.Fatalf("unexpected event count: got=%d events=%v", len(events), events)
	}
	if events[0] != "http:stop" || events[1] != "tun:start" || events[2] != "http:start" {
		t.Fatalf("rollback sequencing mismatch, events=%v", events)
	}
}
