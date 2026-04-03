package setup

import (
	"path/filepath"
	"testing"
)

func TestResolveDefaultConfigPathForMode_DaemonUsesRuntimeConfigPath(t *testing.T) {
	runtimeDir := t.TempDir()
	t.Setenv("ALIANG_DATA_DIR", runtimeDir)

	resolvedPath, err := ResolveDefaultConfigPathForMode(RuntimeModeDaemon, "~/.aliang/config.json")
	if err != nil {
		t.Fatalf("ResolveDefaultConfigPathForMode() error = %v", err)
	}

	wantPath := filepath.Join(runtimeDir, "config.json")
	if resolvedPath != wantPath {
		t.Fatalf("resolved path = %q, want %q", resolvedPath, wantPath)
	}
}

func TestRuntimeExecutablePath_UsesCoreDataDir(t *testing.T) {
	runtimeDir := t.TempDir()
	t.Setenv("ALIANG_DATA_DIR", runtimeDir)

	wantPath := filepath.Join(runtimeDir, BinaryFilename())
	if got := RuntimeExecutablePath(); got != wantPath {
		t.Fatalf("RuntimeExecutablePath() = %q, want %q", got, wantPath)
	}
}
