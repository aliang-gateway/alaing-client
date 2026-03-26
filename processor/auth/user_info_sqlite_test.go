package user

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestUserInfoSQLitePersistenceAndLegacyMigration(t *testing.T) {
	baseDir := t.TempDir()
	t.Setenv("HOME", filepath.Join(baseDir, "home"))
	t.Setenv("NURSOR_CACHE_DIR", filepath.Join(baseDir, "cache"))

	if err := DeleteUserInfo(); err != nil {
		t.Fatalf("failed to clean existing user info: %v", err)
	}

	original := &UserInfo{
		AccessToken:  "access-token-1",
		RefreshToken: "refresh-token-1",
		TokenType:    "Bearer",
		ExpiresIn:    3600,
		Username:     "tester",
		Email:        "tester@example.com",
		Role:         "user",
		UserID:       42,
		PlanName:     "Sub2API",
		PlanType:     "active",
		InnerToken:   "inner-1",
		UpdatedAt:    time.Now().UTC(),
	}

	if err := SaveUserInfo(original); err != nil {
		t.Fatalf("failed to save user info to sqlite: %v", err)
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

	legacy := &UserInfo{
		AccessToken:  "legacy-access",
		RefreshToken: "legacy-refresh",
		Username:     "legacy-user",
		Email:        "legacy@example.com",
		InnerToken:   "legacy-inner",
		PlanName:     "Legacy",
		PlanType:     "active",
		UpdatedAt:    time.Now().UTC(),
	}

	legacyPath, err := GetUserInfoPath()
	if err != nil {
		t.Fatalf("failed to get legacy path: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(legacyPath), 0o700); err != nil {
		t.Fatalf("failed to create legacy directory: %v", err)
	}

	encrypted, err := EncryptUserInfoFile(legacy)
	if err != nil {
		t.Fatalf("failed to encrypt legacy payload: %v", err)
	}
	if err := os.WriteFile(legacyPath, encrypted, 0o600); err != nil {
		t.Fatalf("failed to write legacy file: %v", err)
	}

	migrated, err := LoadUserInfo()
	if err != nil {
		t.Fatalf("failed to load and migrate legacy user info: %v", err)
	}
	if migrated.Username != legacy.Username || migrated.AccessToken != legacy.AccessToken || migrated.RefreshToken != legacy.RefreshToken {
		t.Fatalf("migrated user info mismatch: got %#v want %#v", migrated, legacy)
	}

	if err := os.Remove(legacyPath); err != nil && !os.IsNotExist(err) {
		t.Fatalf("failed to remove legacy file after migration check: %v", err)
	}

	migratedReload, err := LoadUserInfo()
	if err != nil {
		t.Fatalf("failed to load migrated sqlite user info: %v", err)
	}
	if migratedReload.Username != legacy.Username || migratedReload.AccessToken != legacy.AccessToken {
		t.Fatalf("sqlite migrated reload mismatch: got %#v want %#v", migratedReload, legacy)
	}
}
