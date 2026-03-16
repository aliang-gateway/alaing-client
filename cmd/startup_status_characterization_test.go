package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"nursor.org/nursorgate/processor/runtime"
)

func TestDetermineInitialStartupStatus_WhenTokenProvided_ReturnsConfiguring(t *testing.T) {
	tempHome := t.TempDir()
	t.Setenv("HOME", tempHome)

	previousDefault := IsUsingDefaultConfig()
	setUseDefaultConfig(false)
	t.Cleanup(func() {
		setUseDefaultConfig(previousDefault)
	})

	status := DetermineInitialStartupStatusForTest("token-123")
	if status != runtime.CONFIGURING {
		t.Fatalf("expected %s, got %s", runtime.CONFIGURING, status)
	}
}

func TestDetermineInitialStartupStatus_WhenNoTokenNoLocalUser_ReturnsUnconfigured(t *testing.T) {
	tempHome := t.TempDir()
	t.Setenv("HOME", tempHome)

	previousDefault := IsUsingDefaultConfig()
	setUseDefaultConfig(false)
	t.Cleanup(func() {
		setUseDefaultConfig(previousDefault)
	})

	status := DetermineInitialStartupStatusForTest("")
	if status != runtime.UNCONFIGURED {
		t.Fatalf("expected %s, got %s", runtime.UNCONFIGURED, status)
	}
}

func TestDetermineInitialStartupStatus_WhenNoTokenWithLocalUserAndDefaultConfig_ReturnsReady(t *testing.T) {
	tempHome := t.TempDir()
	t.Setenv("HOME", tempHome)

	userInfoPath := filepath.Join(tempHome, ".nonelane", "userinfo.json")
	if err := os.MkdirAll(filepath.Dir(userInfoPath), 0o755); err != nil {
		t.Fatalf("failed to create user info dir: %v", err)
	}
	if err := os.WriteFile(userInfoPath, []byte(`{"username":"tester"}`), 0o600); err != nil {
		t.Fatalf("failed to create user info file: %v", err)
	}

	previousDefault := IsUsingDefaultConfig()
	setUseDefaultConfig(true)
	t.Cleanup(func() {
		setUseDefaultConfig(previousDefault)
	})

	status := DetermineInitialStartupStatusForTest("")
	if status != runtime.READY {
		t.Fatalf("expected %s, got %s", runtime.READY, status)
	}
}

func TestDetermineInitialStartupStatus_WhenNoTokenWithLocalUserAndCustomConfig_ReturnsConfigured(t *testing.T) {
	tempHome := t.TempDir()
	t.Setenv("HOME", tempHome)

	userInfoPath := filepath.Join(tempHome, ".nonelane", "userinfo.json")
	if err := os.MkdirAll(filepath.Dir(userInfoPath), 0o755); err != nil {
		t.Fatalf("failed to create user info dir: %v", err)
	}
	if err := os.WriteFile(userInfoPath, []byte(`{"username":"tester"}`), 0o600); err != nil {
		t.Fatalf("failed to create user info file: %v", err)
	}

	previousDefault := IsUsingDefaultConfig()
	setUseDefaultConfig(false)
	t.Cleanup(func() {
		setUseDefaultConfig(previousDefault)
	})

	status := DetermineInitialStartupStatusForTest("")
	if status != runtime.CONFIGURED {
		t.Fatalf("expected %s, got %s", runtime.CONFIGURED, status)
	}
}
