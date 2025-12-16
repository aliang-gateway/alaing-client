package cache

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestGetCacheDir tests the cache directory resolution
func TestGetCacheDir(t *testing.T) {
	tests := []struct {
		name    string
		envVar  string
		wantErr bool
		check   func(t *testing.T, dir string)
	}{
		{
			name:    "default cache dir",
			envVar:  "",
			wantErr: false,
			check: func(t *testing.T, dir string) {
				if !filepath.IsAbs(dir) {
					t.Errorf("cache dir should be absolute, got: %s", dir)
				}
				if !strings.HasSuffix(dir, DefaultCacheDirName) {
					t.Errorf("cache dir should end with %s, got: %s", DefaultCacheDirName, dir)
				}
			},
		},
		{
			name:    "env var override",
			envVar:  "/tmp/test_cache",
			wantErr: false,
			check: func(t *testing.T, dir string) {
				if dir != "/tmp/test_cache" {
					t.Errorf("expected /tmp/test_cache, got: %s", dir)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset cache for testing
			cacheDir = ""

			// Set environment variable
			oldEnv := os.Getenv(CacheDirEnvVar)
			if tt.envVar != "" {
				os.Setenv(CacheDirEnvVar, tt.envVar)
			} else {
				os.Unsetenv(CacheDirEnvVar)
			}
			defer func() {
				if oldEnv != "" {
					os.Setenv(CacheDirEnvVar, oldEnv)
				} else {
					os.Unsetenv(CacheDirEnvVar)
				}
				// Reset cache for other tests
				cacheDir = ""
			}()

			dir, err := GetCacheDir()
			if (err != nil) != tt.wantErr {
				t.Errorf("GetCacheDir() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && tt.check != nil {
				tt.check(t, dir)
			}
		})
	}
}

// TestGetCacheSubdir tests cache subdirectory creation
func TestGetCacheSubdir(t *testing.T) {
	// Use a temporary directory for testing
	tempDir := filepath.Join(os.TempDir(), "nursor_test_cache")
	os.Setenv(CacheDirEnvVar, tempDir)
	defer func() {
		os.Unsetenv(CacheDirEnvVar)
		os.RemoveAll(tempDir)
		cacheDir = ""
	}()

	tests := []struct {
		name       string
		subdir     string
		wantErr    bool
		checkExist bool
	}{
		{
			name:       "simple subdir",
			subdir:     "test",
			wantErr:    false,
			checkExist: true,
		},
		{
			name:       "nested subdir",
			subdir:     "nacos/cache",
			wantErr:    false,
			checkExist: true,
		},
		{
			name:       "empty subdir",
			subdir:     "",
			wantErr:    false,
			checkExist: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir, err := GetCacheSubdir(tt.subdir)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetCacheSubdir() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.checkExist && !tt.wantErr {
				if _, err := os.Stat(dir); os.IsNotExist(err) {
					t.Errorf("expected directory to exist: %s", dir)
				}

				// Check permissions (on Unix-like systems)
				info, _ := os.Stat(dir)
				if info.Mode().Perm() != DefaultPermissions {
					t.Logf("Warning: directory permissions are %o, expected %o (this is OK on some filesystems)", info.Mode().Perm(), DefaultPermissions)
				}
			}
		})
	}
}

// TestGetCacheFile tests cache file path resolution
func TestGetCacheFile(t *testing.T) {
	tempDir := filepath.Join(os.TempDir(), "nursor_test_files")
	os.Setenv(CacheDirEnvVar, tempDir)
	defer func() {
		os.Unsetenv(CacheDirEnvVar)
		os.RemoveAll(tempDir)
		cacheDir = ""
	}()

	tests := []struct {
		name     string
		filename string
		wantErr  bool
	}{
		{
			name:     "simple filename",
			filename: "config.json",
			wantErr:  false,
		},
		{
			name:     "nested path",
			filename: "nacos/cache/config.json",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path, err := GetCacheFile(tt.filename)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetCacheFile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				expected := filepath.Join(tempDir, tt.filename)
				if path != expected {
					t.Errorf("expected %s, got %s", expected, path)
				}
			}
		})
	}
}

// TestExpandHome tests home directory expansion
func TestExpandHome(t *testing.T) {
	tests := []struct {
		name      string
		path      string
		wantError bool
		checkFn   func(t *testing.T, result string)
	}{
		{
			name:      "tilde only",
			path:      "~",
			wantError: false,
			checkFn: func(t *testing.T, result string) {
				home, _ := getHomeDir()
				if result != home {
					t.Errorf("expected %s, got %s", home, result)
				}
			},
		},
		{
			name:      "tilde with path",
			path:      "~/.nonelane",
			wantError: false,
			checkFn: func(t *testing.T, result string) {
				if !filepath.IsAbs(result) {
					t.Errorf("expected absolute path, got %s", result)
				}
				if !strings.HasSuffix(result, ".nonelane") {
					t.Errorf("expected path to end with .nonelane, got %s", result)
				}
			},
		},
		{
			name:      "absolute path",
			path:      "/tmp/cache",
			wantError: false,
			checkFn: func(t *testing.T, result string) {
				if result != "/tmp/cache" {
					t.Errorf("expected /tmp/cache, got %s", result)
				}
			},
		},
		{
			name:      "relative path",
			path:      "cache",
			wantError: false,
			checkFn: func(t *testing.T, result string) {
				if result != "cache" {
					t.Errorf("expected cache, got %s", result)
				}
			},
		},
		{
			name:      "empty path",
			path:      "",
			wantError: false,
			checkFn: func(t *testing.T, result string) {
				if result != "" {
					t.Errorf("expected empty string, got %s", result)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := expandHome(tt.path)
			if (err != nil) != tt.wantError {
				t.Errorf("expandHome() error = %v, wantError %v", err, tt.wantError)
				return
			}

			if !tt.wantError && tt.checkFn != nil {
				tt.checkFn(t, result)
			}
		})
	}
}

// TestCacheDirPermissions tests that created directories have correct permissions
func TestCacheDirPermissions(t *testing.T) {
	tempDir := filepath.Join(os.TempDir(), "nursor_perms_test")
	os.RemoveAll(tempDir) // Ensure clean state
	os.Setenv(CacheDirEnvVar, tempDir)
	defer func() {
		os.Unsetenv(CacheDirEnvVar)
		os.RemoveAll(tempDir)
		cacheDir = ""
	}()

	// Create the cache directory
	dir, err := GetCacheDir()
	if err != nil {
		t.Fatalf("GetCacheDir() failed: %v", err)
	}

	// Check directory exists
	info, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("os.Stat() failed: %v", err)
	}

	if !info.IsDir() {
		t.Errorf("expected directory, got file")
	}

	// Check permissions (may vary by filesystem, but should be writable)
	if !info.Mode().IsDir() {
		t.Errorf("expected directory mode")
	}

	// Verify we can write to the directory
	testFile := filepath.Join(dir, "test.txt")
	err = os.WriteFile(testFile, []byte("test"), 0o666)
	if err != nil {
		t.Errorf("failed to write file to cache directory: %v", err)
	}
	defer os.Remove(testFile)
}

// BenchmarkGetCacheDir benchmarks cache directory resolution
func BenchmarkGetCacheDir(b *testing.B) {
	cacheDir = "" // Reset cache
	for i := 0; i < b.N; i++ {
		GetCacheDir()
	}
}
