package user

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"aliang.one/nursorgate/processor/config"
)

func TestRefreshSession_InvalidRefreshTokenClearsPersistedSession(t *testing.T) {
	baseDir := t.TempDir()
	t.Setenv("HOME", filepath.Join(baseDir, "home"))
	t.Setenv("ALIANG_CACHE_DIR", filepath.Join(baseDir, "cache"))

	ResetAuthPersistenceForTest()
	config.ResetGlobalConfigForTest()
	tokenRefresher = nil
	t.Cleanup(func() {
		StopTokenRefresh()
		tokenRefresher = nil
		ResetAuthPersistenceForTest()
		config.ResetGlobalConfigForTest()
	})

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("unexpected method %s", r.Method)
		}
		if r.URL.Path != "/api/v1/auth/refresh" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"code":401,"message":"invalid refresh token","reason":"REFRESH_TOKEN_INVALID"}`))
	}))
	defer server.Close()

	config.SetGlobalConfig(&config.Config{Core: &config.CoreConfig{APIServer: server.URL}})
	config.SetHasLocalUserInfo(true)

	if err := SaveUserInfo(&UserInfo{
		AccessToken:  "access-token",
		RefreshToken: "stale-refresh-token",
		TokenType:    "Bearer",
		Username:     "tester",
		UpdatedAt:    time.Now(),
	}); err != nil {
		t.Fatalf("SaveUserInfo() error = %v", err)
	}

	tokenRefresher = NewTokenRefresher()
	if err := tokenRefresher.Start(); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	_, err := RefreshSession("stale-refresh-token")
	if !errors.Is(err, ErrRefreshTokenInvalid) {
		t.Fatalf("RefreshSession() error = %v, want ErrRefreshTokenInvalid", err)
	}

	if persisted, persistedErr := HasPersistedUserInfo(); persistedErr != nil {
		t.Fatalf("HasPersistedUserInfo() error = %v", persistedErr)
	} else if persisted {
		t.Fatal("expected persisted user info to be cleared")
	}

	if got := GetCurrentUserInfo(); got != nil {
		t.Fatalf("GetCurrentUserInfo() = %#v, want nil", got)
	}

	if config.HasLocalUserInfo() {
		t.Fatal("expected local user info flag to be cleared")
	}

	if refresher := GetTokenRefresher(); refresher != nil && refresher.IsRunning() {
		t.Fatal("expected token refresher to stop after invalid refresh token")
	}
}

func TestRestoreSession_InvalidRefreshTokenDoesNotFallbackToStaleLocalUserInfo(t *testing.T) {
	baseDir := t.TempDir()
	t.Setenv("HOME", filepath.Join(baseDir, "home"))
	t.Setenv("ALIANG_CACHE_DIR", filepath.Join(baseDir, "cache"))

	ResetAuthPersistenceForTest()
	config.ResetGlobalConfigForTest()
	tokenRefresher = nil
	t.Cleanup(func() {
		StopTokenRefresh()
		tokenRefresher = nil
		ResetAuthPersistenceForTest()
		config.ResetGlobalConfigForTest()
	})

	var profileCalls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/auth/refresh":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"code":401,"message":"invalid refresh token","reason":"REFRESH_TOKEN_INVALID"}`))
		case "/api/v1/user/profile":
			profileCalls.Add(1)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"data":{"id":1,"username":"should-not-be-used"}}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	config.SetGlobalConfig(&config.Config{Core: &config.CoreConfig{APIServer: server.URL}})
	config.SetHasLocalUserInfo(true)

	if err := SaveUserInfo(&UserInfo{
		AccessToken:  "stale-access-token",
		RefreshToken: "stale-refresh-token",
		TokenType:    "Bearer",
		Username:     "stale-user",
		UpdatedAt:    time.Now(),
	}); err != nil {
		t.Fatalf("SaveUserInfo() error = %v", err)
	}

	info, err := RestoreSession()
	if info != nil {
		t.Fatalf("RestoreSession() info = %#v, want nil", info)
	}
	if !errors.Is(err, ErrRefreshTokenInvalid) {
		t.Fatalf("RestoreSession() error = %v, want ErrRefreshTokenInvalid", err)
	}

	if got := profileCalls.Load(); got != 0 {
		t.Fatalf("expected no fallback profile fetch after invalid refresh token, got %d calls", got)
	}

	if persisted, persistedErr := HasPersistedUserInfo(); persistedErr != nil {
		t.Fatalf("HasPersistedUserInfo() error = %v", persistedErr)
	} else if persisted {
		t.Fatal("expected persisted user info to be cleared")
	}
}
