package user

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"aliang.one/nursorgate/processor/config"
)

func TestRefreshSession_SerializesTokenRotation(t *testing.T) {
	baseDir, err := os.MkdirTemp("", "aliang-refresh-session-*")
	if err != nil {
		t.Fatalf("MkdirTemp() error = %v", err)
	}
	t.Setenv("HOME", filepath.Join(baseDir, "home"))
	t.Setenv("ALIANG_CACHE_DIR", filepath.Join(baseDir, "cache"))
	defer os.RemoveAll(baseDir)

	defer StopTokenRefresh()
	defer ResetAuthPersistenceForTest()
	defer config.ResetGlobalConfigForTest()

	ResetAuthPersistenceForTest()
	StopTokenRefresh()
	config.ResetGlobalConfigForTest()

	var refreshCalls int32
	var refreshTokensMu sync.Mutex
	refreshTokens := make([]string, 0, 4)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/auth/refresh":
			var payload struct {
				RefreshToken string `json:"refresh_token"`
			}
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Fatalf("decode refresh payload failed: %v", err)
			}

			refreshTokensMu.Lock()
			refreshTokens = append(refreshTokens, payload.RefreshToken)
			refreshTokensMu.Unlock()

			callIndex := atomic.AddInt32(&refreshCalls, 1)
			if payload.RefreshToken == "refresh-1" && callIndex == 1 {
				time.Sleep(120 * time.Millisecond)
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"data":{"access_token":"access-2","refresh_token":"refresh-2","expires_in":3600,"token_type":"Bearer"}}`))
				return
			}
			if payload.RefreshToken == "refresh-1" {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				_, _ = w.Write([]byte(`{"code":401,"message":"invalid refresh token","reason":"REFRESH_TOKEN_INVALID"}`))
				return
			}
			if payload.RefreshToken == "refresh-2" {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"data":{"access_token":"access-3","refresh_token":"refresh-3","expires_in":3600,"token_type":"Bearer"}}`))
				return
			}

			t.Fatalf("unexpected refresh token: %q", payload.RefreshToken)

		case "/api/v1/user/profile":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"data":{"id":1,"email":"user@example.com","username":"user","role":"member","balance":12.5,"concurrency":2,"status":"active","allowed_groups":[1,2],"created_at":"2026-01-01T00:00:00Z","updated_at":"2026-04-09T00:00:00Z"}}`))
			return
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	config.SetGlobalConfig(&config.Config{Core: &config.CoreConfig{APIServer: server.URL}})

	if err := SaveUserInfo(&UserInfo{
		AccessToken:  "access-1",
		RefreshToken: "refresh-1",
		TokenType:    "Bearer",
		Username:     "user",
		Email:        "user@example.com",
		UpdatedAt:    time.Now().Add(-30 * time.Minute),
		ExpiresIn:    3600,
	}); err != nil {
		t.Fatalf("SaveUserInfo() error = %v", err)
	}

	start := make(chan struct{})
	var wg sync.WaitGroup
	results := make([]*UserInfo, 2)
	errs := make([]error, 2)

	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			<-start
			results[index], errs[index] = RefreshSession("refresh-1")
		}(i)
	}

	close(start)
	wg.Wait()

	for i, refreshErr := range errs {
		if refreshErr != nil {
			t.Fatalf("RefreshSession() #%d error = %v", i, refreshErr)
		}
		if results[i] == nil {
			t.Fatalf("RefreshSession() #%d returned nil user info", i)
		}
		if results[i].RefreshToken != "refresh-2" {
			t.Fatalf("RefreshSession() #%d refresh token = %q, want refresh-2", i, results[i].RefreshToken)
		}
	}

	if got := atomic.LoadInt32(&refreshCalls); got != 1 {
		t.Fatalf("expected exactly 1 refresh HTTP call, got %d (tokens=%v)", got, refreshTokens)
	}

	saved, err := LoadUserInfo()
	if err != nil {
		t.Fatalf("LoadUserInfo() error = %v", err)
	}
	if saved.RefreshToken != "refresh-2" {
		t.Fatalf("saved refresh token = %q, want refresh-2", saved.RefreshToken)
	}
}
