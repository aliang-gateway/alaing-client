package user

import "testing"

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
