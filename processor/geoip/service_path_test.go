package geoip

import (
	"path/filepath"
	"testing"

	"aliang.one/nursorgate/common/cache"
)

func TestDefaultDatabasePath_UsesGeoIPSubdirUnderRuntimeStateDir(t *testing.T) {
	cache.ResetCacheDirForTest()
	t.Cleanup(cache.ResetCacheDirForTest)

	runtimeDir := t.TempDir()
	t.Setenv("HOME", "")
	t.Setenv("ALIANG_DATA_DIR", runtimeDir)
	t.Setenv("ALIANG_CACHE_DIR", "")

	dbPath, err := DefaultDatabasePath()
	if err != nil {
		t.Fatalf("DefaultDatabasePath() error = %v", err)
	}

	wantPath := filepath.Join(runtimeDir, "geoip", DefaultGeoIPDatabaseFile)
	if dbPath != wantPath {
		t.Fatalf("db path = %q, want %q", dbPath, wantPath)
	}
}
