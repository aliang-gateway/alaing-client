package user

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestUserInfoSQLitePersistenceAndLegacyMigration(t *testing.T) {
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

	legacy := &legacyStoredUserInfo{
		AccessToken:  "legacy-access",
		RefreshToken: "legacy-refresh",
		Username:     "legacy-user",
		Email:        "legacy@example.com",
		Role:         "user",
		UserID:       77,
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
	if migrated.ID != legacy.UserID {
		t.Fatalf("migrated id mismatch: got %d want %d", migrated.ID, legacy.UserID)
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

func TestUserInfoMigratesFromLegacyAuthSessionDBIntoUnifiedGateData(t *testing.T) {
	baseDir := t.TempDir()
	t.Setenv("HOME", filepath.Join(baseDir, "home"))
	t.Setenv("NURSOR_CACHE_DIR", filepath.Join(baseDir, "cache"))
	ResetAuthPersistenceForTest()
	t.Cleanup(ResetAuthPersistenceForTest)

	legacyDBPath, err := getLegacyAuthSessionDBPath()
	if err != nil {
		t.Fatalf("failed to get legacy auth session db path: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(legacyDBPath), 0o700); err != nil {
		t.Fatalf("failed to create legacy auth db dir: %v", err)
	}

	legacyDB, err := gorm.Open(sqlite.Open(legacyDBPath), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open legacy auth session db: %v", err)
	}
	if err := legacyDB.AutoMigrate(&legacyAuthSessionRecord{}); err != nil {
		t.Fatalf("failed to migrate legacy auth session table: %v", err)
	}

	legacy := legacyStoredUserInfo{
		AccessToken:  "legacy-db-access",
		RefreshToken: "legacy-db-refresh",
		TokenType:    "Bearer",
		ExpiresIn:    1800,
		UserID:       101,
		Email:        "legacy-db@example.com",
		Username:     "legacy-db-user",
		Role:         "user",
		PlanType:     "active",
		UpdatedAt:    time.Now().UTC(),
	}
	raw, err := json.Marshal(legacy)
	if err != nil {
		t.Fatalf("failed to marshal legacy auth payload: %v", err)
	}
	if err := legacyDB.Save(&legacyAuthSessionRecord{ID: 1, Data: string(raw), UpdatedAt: legacy.UpdatedAt}).Error; err != nil {
		t.Fatalf("failed to save legacy auth session record: %v", err)
	}

	if err := InitializeAuthPersistence(); err != nil {
		t.Fatalf("failed to initialize auth persistence: %v", err)
	}

	unifiedPath, err := GetAuthSessionDBPath()
	if err != nil {
		t.Fatalf("failed to get unified auth db path: %v", err)
	}
	if filepath.Base(unifiedPath) != "gate.data" {
		t.Fatalf("expected unified auth db path to use gate.data, got %s", unifiedPath)
	}

	loaded, err := LoadUserInfo()
	if err != nil {
		t.Fatalf("failed to load migrated user info: %v", err)
	}
	if loaded.AccessToken != legacy.AccessToken || loaded.RefreshToken != legacy.RefreshToken {
		t.Fatalf("migrated legacy auth tokens mismatch: got %#v want %#v", loaded, legacy)
	}
	if loaded.ID != legacy.UserID || loaded.Email != legacy.Email || loaded.Username != legacy.Username {
		t.Fatalf("migrated legacy auth profile mismatch: got %#v want %#v", loaded, legacy)
	}
}

func TestUserInfoMigratesFromLegacyGateDataTable(t *testing.T) {
	baseDir := t.TempDir()
	t.Setenv("HOME", filepath.Join(baseDir, "home"))
	t.Setenv("NURSOR_CACHE_DIR", filepath.Join(baseDir, "cache"))
	ResetAuthPersistenceForTest()
	t.Cleanup(ResetAuthPersistenceForTest)

	gateDataPath, err := GetAuthSessionDBPath()
	if err != nil {
		t.Fatalf("failed to get gate.data path: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(gateDataPath), 0o700); err != nil {
		t.Fatalf("failed to create gate.data dir: %v", err)
	}

	db, err := gorm.Open(sqlite.Open(gateDataPath), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open gate.data: %v", err)
	}
	if err := db.Exec(`CREATE TABLE user_info (
			id INTEGER PRIMARY KEY CHECK (id = 1),
			encrypted_payload TEXT NOT NULL,
			updated_at INTEGER NOT NULL
		)`).Error; err != nil {
		t.Fatalf("failed to create legacy gate.data table: %v", err)
	}

	legacy := &legacyStoredUserInfo{
		AccessToken:  "legacy-gate-access",
		RefreshToken: "legacy-gate-refresh",
		UserID:       202,
		Email:        "legacy-gate@example.com",
		Username:     "legacy-gate-user",
		Role:         "user",
		PlanType:     "active",
		UpdatedAt:    time.Now().UTC(),
	}
	encrypted, err := EncryptUserInfoFile(legacy)
	if err != nil {
		t.Fatalf("failed to encrypt legacy gate.data payload: %v", err)
	}
	if err := db.Exec(
		"INSERT INTO user_info (id, encrypted_payload, updated_at) VALUES (?, ?, ?)",
		1,
		string(encrypted),
		time.Now().Unix(),
	).Error; err != nil {
		t.Fatalf("failed to insert legacy gate.data row: %v", err)
	}

	ResetAuthPersistenceForTest()

	loaded, err := LoadUserInfo()
	if err != nil {
		t.Fatalf("failed to load migrated gate.data user info: %v", err)
	}
	if loaded.AccessToken != legacy.AccessToken || loaded.RefreshToken != legacy.RefreshToken {
		t.Fatalf("migrated gate.data tokens mismatch: got %#v want %#v", loaded, legacy)
	}
	if loaded.ID != legacy.UserID || loaded.Email != legacy.Email || loaded.Username != legacy.Username {
		t.Fatalf("migrated gate.data profile mismatch: got %#v want %#v", loaded, legacy)
	}
}
