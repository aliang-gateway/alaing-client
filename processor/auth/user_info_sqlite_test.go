package user

import (
	"path/filepath"
	"testing"
	"time"
)

func TestUserInfoSQLitePersistence(t *testing.T) {
	baseDir := t.TempDir()
	t.Setenv("HOME", filepath.Join(baseDir, "home"))
	t.Setenv("NURSOR_CACHE_DIR", filepath.Join(baseDir, "cache"))
	ResetAuthPersistenceForTest()
	t.Cleanup(ResetAuthPersistenceForTest)

	if err := DeleteUserInfo(); err != nil {
		t.Fatalf("failed to clean existing user info: %v", err)
	}

	original := &UserInfo{
		AccessToken:   "access-token-1",
		RefreshToken:  "refresh-token-1",
		TokenType:     "Bearer",
		ExpiresIn:     3600,
		Username:      "tester",
		Email:         "tester@example.com",
		Role:          "user",
		ID:            42,
		Status:        "active",
		Balance:       12.5,
		Concurrency:   3,
		AllowedGroups: []int64{1, 3},
		UpdatedAt:     time.Now().UTC(),
	}

	if err := SaveUserInfo(original); err != nil {
		t.Fatalf("failed to save user info to sqlite: %v", err)
	}

	dbPath, err := GetAuthSessionDBPath()
	if err != nil {
		t.Fatalf("failed to get auth db path: %v", err)
	}
	if filepath.Base(dbPath) != "aliang.db" {
		t.Fatalf("expected auth db path to use aliang.db, got %s", dbPath)
	}

	hasPersisted, err := HasPersistedUserInfo()
	if err != nil {
		t.Fatalf("failed to check persisted user info: %v", err)
	}
	if !hasPersisted {
		t.Fatalf("expected persisted user info after save")
	}

	loaded, err := LoadUserInfo()
	if err != nil {
		t.Fatalf("failed to load user info from sqlite: %v", err)
	}
	if loaded.Username != original.Username || loaded.AccessToken != original.AccessToken || loaded.RefreshToken != original.RefreshToken {
		t.Fatalf("loaded user info mismatch: got %#v want %#v", loaded, original)
	}

	if err := DeleteUserInfo(); err != nil {
		t.Fatalf("failed to delete sqlite user info: %v", err)
	}

	hasPersisted, err = HasPersistedUserInfo()
	if err != nil {
		t.Fatalf("failed to re-check persisted user info: %v", err)
	}
	if hasPersisted {
		t.Fatalf("expected no persisted user info after delete")
	}
}
