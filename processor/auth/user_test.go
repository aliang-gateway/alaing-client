package user

import (
	"path/filepath"
	"testing"
)

func TestGetCurrentAuthorizationHeader_UsesCurrentUserInfo(t *testing.T) {
	previous := GetCurrentUserInfo()
	SetCurrentUserInfo(&UserInfo{
		AccessToken: "access-token-1",
		TokenType:   "Bearer",
	})
	defer SetCurrentUserInfo(previous)

	if got := GetCurrentAuthorizationHeader(); got != "Bearer access-token-1" {
		t.Fatalf("GetCurrentAuthorizationHeader() = %q, want %q", got, "Bearer access-token-1")
	}
}

func TestGetCurrentAuthorizationHeader_DefaultsTokenType(t *testing.T) {
	previous := GetCurrentUserInfo()
	SetCurrentUserInfo(&UserInfo{
		AccessToken: "access-token-2",
	})
	defer SetCurrentUserInfo(previous)

	if got := GetCurrentAuthorizationHeader(); got != "Bearer access-token-2" {
		t.Fatalf("GetCurrentAuthorizationHeader() = %q, want %q", got, "Bearer access-token-2")
	}
}

func TestGetCurrentAuthorizationHeader_FallsBackToPersistedUserInfo(t *testing.T) {
	baseDir := t.TempDir()
	t.Setenv("HOME", filepath.Join(baseDir, "home"))
	t.Setenv("ALIANG_CACHE_DIR", filepath.Join(baseDir, "cache"))
	ResetAuthPersistenceForTest()
	t.Cleanup(ResetAuthPersistenceForTest)

	if err := SaveUserInfo(&UserInfo{
		AccessToken: "persisted-access-token",
		TokenType:   "Bearer",
	}); err != nil {
		t.Fatalf("SaveUserInfo() error = %v", err)
	}

	SetCurrentUserInfo(nil)

	if got := GetCurrentAuthorizationHeader(); got != "Bearer persisted-access-token" {
		t.Fatalf("GetCurrentAuthorizationHeader() = %q, want %q", got, "Bearer persisted-access-token")
	}
}
