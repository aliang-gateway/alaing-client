package services

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	auth "aliang.one/nursorgate/processor/auth"
	"aliang.one/nursorgate/processor/config"
	"aliang.one/nursorgate/processor/runtime"
)

func TestAuthServiceRestoreSession_ReturnsSessionExpiredWhenRefreshTokenInvalid(t *testing.T) {
	baseDir := t.TempDir()
	t.Setenv("HOME", filepath.Join(baseDir, "home"))
	t.Setenv("ALIANG_CACHE_DIR", filepath.Join(baseDir, "cache"))

	auth.ResetAuthPersistenceForTest()
	auth.StopTokenRefresh()
	config.ResetGlobalConfigForTest()
	runtime.ResetGlobalStartupStateForTest()
	t.Cleanup(func() {
		auth.StopTokenRefresh()
		auth.ResetAuthPersistenceForTest()
		config.ResetGlobalConfigForTest()
		runtime.ResetGlobalStartupStateForTest()
	})

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/auth/refresh" {
			http.NotFound(w, r)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"code":401,"message":"invalid refresh token","reason":"REFRESH_TOKEN_INVALID"}`))
	}))
	defer server.Close()

	config.SetGlobalConfig(&config.Config{Core: &config.CoreConfig{APIServer: server.URL}})
	config.SetHasLocalUserInfo(true)
	runtime.GetStartupState().SetUserInfo(&auth.UserInfo{Username: "stale-user"})
	runtime.GetStartupState().SetFetchSuccess(true)
	runtime.GetStartupState().SetStatus(runtime.READY)

	if err := auth.SaveUserInfo(&auth.UserInfo{
		AccessToken:  "stale-access-token",
		RefreshToken: "stale-refresh-token",
		TokenType:    "Bearer",
		Username:     "stale-user",
		UpdatedAt:    time.Now(),
	}); err != nil {
		t.Fatalf("SaveUserInfo() error = %v", err)
	}

	result := NewAuthService().RestoreSession()

	if got := result["status"]; got != "no_session" {
		t.Fatalf("status = %#v, want no_session", got)
	}
	if got := result["error"]; got != "session_expired" {
		t.Fatalf("error = %#v, want session_expired", got)
	}
	if got := runtime.GetStartupState().GetStatus(); got != runtime.UNCONFIGURED {
		t.Fatalf("startup status = %s, want %s", got, runtime.UNCONFIGURED)
	}
	if got := runtime.GetStartupState().GetUserInfo(); got != nil {
		t.Fatalf("startup user info = %#v, want nil", got)
	}
}
