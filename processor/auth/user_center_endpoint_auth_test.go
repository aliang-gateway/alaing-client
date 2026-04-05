package user

import "testing"

func TestResolveAuthTokenForEndpoint_UsesPrimaryAccessTokenForPortalHost(t *testing.T) {
	previous := GetCurrentUserInfo()
	SetCurrentUserInfo(&UserInfo{
		AccessToken:        "sub2api-access-token",
		AliangSessionToken: "aliang-session-token",
	})
	t.Cleanup(func() {
		SetCurrentUserInfo(previous)
	})

	token, err := resolveAuthTokenForEndpoint("https://api.aliang.one/api/v1/keys?page=1&page_size=20&timezone=Asia%2FShanghai")
	if err != nil {
		t.Fatalf("resolveAuthTokenForEndpoint returned error: %v", err)
	}
	if token != "sub2api-access-token" {
		t.Fatalf("resolveAuthTokenForEndpoint token = %q, want %q", token, "sub2api-access-token")
	}
}

func TestExtractAliangSessionToken(t *testing.T) {
	body := []byte(`{"data":{"session_token":"portal-session-123"}}`)
	if got := extractAliangSessionToken(body); got != "portal-session-123" {
		t.Fatalf("extractAliangSessionToken() = %q, want %q", got, "portal-session-123")
	}
}
