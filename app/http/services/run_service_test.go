package services

import (
	"testing"

	"nursor.org/nursorgate/processor/runtime"
)

func TestRunServiceStartServiceAlreadyRunning(t *testing.T) {
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
