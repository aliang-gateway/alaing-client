package user

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"nursor.org/nursorgate/common/cache"
	"nursor.org/nursorgate/common/logger"
)

// UserInfo is the locally persisted Sub2API session snapshot.
// It keeps token state together with the current user/profile fields returned by Sub2API.
type UserInfo struct {
	AccessToken    string    `json:"access_token"`
	RefreshToken   string    `json:"refresh_token"`
	TokenType      string    `json:"token_type,omitempty"`
	ExpiresIn      int       `json:"expires_in,omitempty"`
	ID             int64     `json:"id,omitempty"`
	Email          string    `json:"email,omitempty"`
	Username       string    `json:"username,omitempty"`
	Role           string    `json:"role,omitempty"`
	Balance        float64   `json:"balance,omitempty"`
	Concurrency    int       `json:"concurrency,omitempty"`
	Status         string    `json:"status,omitempty"`
	AllowedGroups  []int64   `json:"allowed_groups,omitempty"`
	CreatedAt      string    `json:"created_at,omitempty"`
	ProfileUpdated string    `json:"profile_updated_at,omitempty"`
	UpdatedAt      time.Time `json:"updated_at"`
}

var (
	userInfoMutex     sync.RWMutex
	currentUserInfo   *UserInfo
	authSessionDBOnce sync.Once
	authSessionDB     *gorm.DB
	authSessionDBErr  error
)

const (
	authTokenRecordID       = 1
	authProfileRecordID     = 1
	legacyAuthSessionDBFile = "auth_session.db"
)

var errNoLegacyUserInfo = errors.New("no legacy user info found")

type authTokenRecord struct {
	ID           uint      `gorm:"primaryKey"`
	AccessToken  string    `gorm:"type:text;not null"`
	RefreshToken string    `gorm:"type:text"`
	TokenType    string    `gorm:"type:varchar(32)"`
	ExpiresIn    int       `gorm:"not null;default:0"`
	UpdatedAt    time.Time `gorm:"autoUpdateTime"`
}

func (authTokenRecord) TableName() string {
	return "sub2api_auth_tokens"
}

type authProfileRecord struct {
	ID                uint      `gorm:"primaryKey"`
	UserID            int64     `gorm:"not null;default:0;index"`
	Email             string    `gorm:"type:varchar(255);index"`
	Username          string    `gorm:"type:varchar(255)"`
	Role              string    `gorm:"type:varchar(64)"`
	Balance           float64   `gorm:"not null;default:0"`
	Concurrency       int       `gorm:"not null;default:0"`
	Status            string    `gorm:"type:varchar(64);index"`
	AllowedGroupsJSON string    `gorm:"column:allowed_groups_json;type:text"`
	RemoteCreatedAt   string    `gorm:"column:remote_created_at;type:varchar(64)"`
	RemoteUpdatedAt   string    `gorm:"column:remote_updated_at;type:varchar(64)"`
	UpdatedAt         time.Time `gorm:"autoUpdateTime"`
}

func (authProfileRecord) TableName() string {
	return "sub2api_user_profiles"
}

type legacyGateUserInfoRecord struct {
	ID               uint   `gorm:"primaryKey"`
	EncryptedPayload string `gorm:"column:encrypted_payload"`
	UpdatedAt        int64  `gorm:"column:updated_at"`
}

func (legacyGateUserInfoRecord) TableName() string {
	return "user_info"
}

type legacyAuthSessionRecord struct {
	ID        uint      `gorm:"primaryKey"`
	Data      string    `gorm:"type:text;not null"`
	UpdatedAt time.Time `gorm:"autoUpdateTime"`
}

func (legacyAuthSessionRecord) TableName() string {
	return "auth_session_records"
}

type legacyStoredUserInfo struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	TokenType    string    `json:"token_type,omitempty"`
	ExpiresIn    int       `json:"expires_in,omitempty"`
	Username     string    `json:"username"`
	Email        string    `json:"email,omitempty"`
	Role         string    `json:"role,omitempty"`
	UserID       int64     `json:"user_id,omitempty"`
	StartTime    string    `json:"start_time"`
	PlanType     string    `json:"plan_type"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// GetUserInfoPath returns the legacy encrypted user-info file path.
func GetUserInfoPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	configDir := filepath.Join(homeDir, ".aliang")
	logger.Debug(fmt.Sprintf("get user info path: %s", configDir))
	return filepath.Join(configDir, "userinfo.json"), nil
}

// GetAuthSessionDBPath returns the canonical shared SQLite data file path.
func GetAuthSessionDBPath() (string, error) {
	return cache.GetUnifiedDataDBPath()
}

// InitializeAuthPersistence ensures the shared auth tables exist and migrates legacy session state when needed.
func InitializeAuthPersistence() error {
	db, err := getAuthSessionDB()
	if err != nil {
		return err
	}
	return migrateLegacyUserInfoIfNeeded(db)
}

// SaveUserInfo persists the current Sub2API session snapshot.
func SaveUserInfo(info *UserInfo) error {
	if info == nil {
		return fmt.Errorf("user info cannot be nil")
	}

	info.UpdatedAt = time.Now()

	if err := saveUserInfoToSQLite(info); err != nil {
		return fmt.Errorf("failed to persist user info to sqlite: %w", err)
	}

	userInfoMutex.Lock()
	copyInfo := *info
	currentUserInfo = &copyInfo
	userInfoMutex.Unlock()

	logger.Debug("User info saved successfully (sqlite)")
	return nil
}

// LoadUserInfo loads the current Sub2API session snapshot from the shared database.
func LoadUserInfo() (*UserInfo, error) {
	if err := InitializeAuthPersistence(); err != nil {
		return nil, err
	}

	sqliteInfo, err := loadUserInfoFromSQLite()
	if err != nil {
		return nil, fmt.Errorf("failed to load user info from sqlite: %w", err)
	}

	userInfoMutex.Lock()
	copyInfo := *sqliteInfo
	currentUserInfo = &copyInfo
	userInfoMutex.Unlock()

	return sqliteInfo, nil
}

// UpdateUserInfo updates the local user snapshot.
func UpdateUserInfo(info *UserInfo) error {
	return SaveUserInfo(info)
}

// DeleteUserInfo deletes all persisted auth session data, including legacy file remnants.
func DeleteUserInfo() error {
	if err := InitializeAuthPersistence(); err != nil && !errors.Is(err, errNoLegacyUserInfo) {
		return err
	}

	if err := deleteUserInfoFromSQLite(); err != nil {
		return err
	}

	filePath, err := GetUserInfoPath()
	if err == nil {
		if removeErr := os.Remove(filePath); removeErr != nil && !os.IsNotExist(removeErr) {
			logger.Warn(fmt.Sprintf("Failed to delete legacy user info file: %v", removeErr))
		}
	}
	legacyDBPath, err := getLegacyAuthSessionDBPath()
	if err == nil {
		if removeErr := os.Remove(legacyDBPath); removeErr != nil && !os.IsNotExist(removeErr) {
			logger.Warn(fmt.Sprintf("Failed to delete legacy auth session db: %v", removeErr))
		}
	}

	userInfoMutex.Lock()
	currentUserInfo = nil
	userInfoMutex.Unlock()

	return nil
}

// HasPersistedUserInfo reports whether a local auth session exists in the unified database or legacy sources.
func HasPersistedUserInfo() (bool, error) {
	if err := InitializeAuthPersistence(); err != nil {
		return false, err
	}

	db, err := getAuthSessionDB()
	if err != nil {
		return false, err
	}

	var count int64
	if err := db.Model(&authTokenRecord{}).Where("id = ?", authTokenRecordID).Count(&count).Error; err != nil {
		return false, err
	}
	if count > 0 {
		return true, nil
	}

	if _, err := loadLegacyUserInfoFromLegacyAuthSessionDB(); err == nil {
		return true, nil
	}
	if _, err := loadLegacyUserInfoFromGateData(db); err == nil {
		return true, nil
	}
	if _, err := loadLegacyUserInfoFromFile(); err == nil {
		return true, nil
	}

	return false, nil
}

func getAuthSessionDB() (*gorm.DB, error) {
	authSessionDBOnce.Do(func() {
		dbPath, err := GetAuthSessionDBPath()
		if err != nil {
			authSessionDBErr = err
			return
		}

		if err := os.MkdirAll(filepath.Dir(dbPath), 0o700); err != nil {
			authSessionDBErr = fmt.Errorf("failed to create auth session db directory: %w", err)
			return
		}

		db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
		if err != nil {
			authSessionDBErr = fmt.Errorf("failed to open sqlite database: %w", err)
			return
		}

		if err := db.AutoMigrate(&authTokenRecord{}, &authProfileRecord{}); err != nil {
			authSessionDBErr = fmt.Errorf("failed to migrate auth session tables: %w", err)
			return
		}

		authSessionDB = db
	})

	return authSessionDB, authSessionDBErr
}

func migrateLegacyUserInfoIfNeeded(db *gorm.DB) error {
	var count int64
	if err := db.Model(&authTokenRecord{}).Where("id = ?", authTokenRecordID).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	legacyInfo, source, err := loadLegacyUserInfoFromSources(db)
	if err != nil {
		if errors.Is(err, errNoLegacyUserInfo) {
			return nil
		}
		return err
	}

	if err := saveUserInfoWithDB(db, legacyInfo); err != nil {
		return fmt.Errorf("failed to migrate legacy user info from %s: %w", source, err)
	}

	logger.Info(fmt.Sprintf("Migrated legacy user info from %s into unified sqlite store", source))
	return nil
}

func loadLegacyUserInfoFromSources(db *gorm.DB) (*UserInfo, string, error) {
	if info, err := loadLegacyUserInfoFromLegacyAuthSessionDB(); err == nil {
		return info, legacyAuthSessionDBFile, nil
	}

	if info, err := loadLegacyUserInfoFromGateData(db); err == nil {
		return info, cache.UnifiedDataDBFile + ".user_info", nil
	}

	if info, err := loadLegacyUserInfoFromFile(); err == nil {
		return info, "userinfo.json", nil
	}

	return nil, "", errNoLegacyUserInfo
}

func saveUserInfoToSQLite(info *UserInfo) error {
	db, err := getAuthSessionDB()
	if err != nil {
		return err
	}
	return saveUserInfoWithDB(db, info)
}

func saveUserInfoWithDB(db *gorm.DB, info *UserInfo) error {
	allowedGroupsJSON, err := json.Marshal(info.AllowedGroups)
	if err != nil {
		return fmt.Errorf("failed to marshal allowed groups: %w", err)
	}

	token := authTokenRecord{
		ID:           authTokenRecordID,
		AccessToken:  info.AccessToken,
		RefreshToken: info.RefreshToken,
		TokenType:    info.TokenType,
		ExpiresIn:    info.ExpiresIn,
		UpdatedAt:    info.UpdatedAt,
	}

	profile := authProfileRecord{
		ID:                authProfileRecordID,
		UserID:            info.ID,
		Email:             strings.TrimSpace(info.Email),
		Username:          strings.TrimSpace(info.Username),
		Role:              strings.TrimSpace(info.Role),
		Balance:           info.Balance,
		Concurrency:       info.Concurrency,
		Status:            strings.TrimSpace(info.Status),
		AllowedGroupsJSON: string(allowedGroupsJSON),
		RemoteCreatedAt:   strings.TrimSpace(info.CreatedAt),
		RemoteUpdatedAt:   strings.TrimSpace(info.ProfileUpdated),
		UpdatedAt:         info.UpdatedAt,
	}

	return db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(&token).Error; err != nil {
			return err
		}
		if err := tx.Save(&profile).Error; err != nil {
			return err
		}
		return nil
	})
}

func loadUserInfoFromSQLite() (*UserInfo, error) {
	db, err := getAuthSessionDB()
	if err != nil {
		return nil, err
	}

	var token authTokenRecord
	if err := db.First(&token, "id = ?", authTokenRecordID).Error; err != nil {
		return nil, err
	}

	var profile authProfileRecord
	if err := db.First(&profile, "id = ?", authProfileRecordID).Error; err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	var allowedGroups []int64
	if strings.TrimSpace(profile.AllowedGroupsJSON) != "" {
		if err := json.Unmarshal([]byte(profile.AllowedGroupsJSON), &allowedGroups); err != nil {
			return nil, fmt.Errorf("failed to unmarshal allowed groups: %w", err)
		}
	}

	info := &UserInfo{
		AccessToken:    token.AccessToken,
		RefreshToken:   token.RefreshToken,
		TokenType:      token.TokenType,
		ExpiresIn:      token.ExpiresIn,
		ID:             profile.UserID,
		Email:          profile.Email,
		Username:       profile.Username,
		Role:           profile.Role,
		Balance:        profile.Balance,
		Concurrency:    profile.Concurrency,
		Status:         profile.Status,
		AllowedGroups:  allowedGroups,
		CreatedAt:      profile.RemoteCreatedAt,
		ProfileUpdated: profile.RemoteUpdatedAt,
		UpdatedAt:      token.UpdatedAt,
	}

	if info.UpdatedAt.IsZero() {
		info.UpdatedAt = profile.UpdatedAt
	}

	return info, nil
}

func deleteUserInfoFromSQLite() error {
	db, err := getAuthSessionDB()
	if err != nil {
		return err
	}

	return db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Delete(&authTokenRecord{}, "id = ?", authTokenRecordID).Error; err != nil {
			return fmt.Errorf("failed to delete auth token record: %w", err)
		}
		if err := tx.Delete(&authProfileRecord{}, "id = ?", authProfileRecordID).Error; err != nil {
			return fmt.Errorf("failed to delete auth profile record: %w", err)
		}
		if err := tx.Exec("DELETE FROM user_info WHERE id = ?", 1).Error; err != nil && !isMissingSQLiteTableError(err) {
			return fmt.Errorf("failed to delete legacy unified-db user_info row: %w", err)
		}
		return nil
	})
}

func loadLegacyUserInfoFromLegacyAuthSessionDB() (*UserInfo, error) {
	dbPath, err := getLegacyAuthSessionDBPath()
	if err != nil {
		return nil, err
	}
	if _, err := os.Stat(dbPath); err != nil {
		return nil, errNoLegacyUserInfo
	}

	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to open legacy auth session db: %w", err)
	}

	var record legacyAuthSessionRecord
	if err := db.First(&record, "id = ?", authTokenRecordID).Error; err != nil {
		return nil, err
	}

	return parseLegacyStoredUserInfo(record.Data)
}

func loadLegacyUserInfoFromGateData(db *gorm.DB) (*UserInfo, error) {
	var count int64
	if err := db.Table("sqlite_master").Where("type = ? AND name = ?", "table", "user_info").Count(&count).Error; err != nil {
		return nil, err
	}
	if count == 0 {
		return nil, errNoLegacyUserInfo
	}

	var record legacyGateUserInfoRecord
	if err := db.First(&record, "id = ?", 1).Error; err != nil {
		return nil, err
	}

	decryptedInfo, err := DecryptUserInfoFile([]byte(record.EncryptedPayload))
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt legacy unified-db payload: %w", err)
	}

	return mapLegacyUserInfo(decryptedInfo), nil
}

func loadLegacyUserInfoFromFile() (*UserInfo, error) {
	filePath, err := GetUserInfoPath()
	if err != nil {
		return nil, err
	}

	if _, err := os.Stat(filePath); err != nil {
		return nil, err
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read legacy user info file: %w", err)
	}

	decryptedInfo, err := DecryptUserInfoFile(data)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt legacy user info file: %w", err)
	}

	return mapLegacyUserInfo(decryptedInfo), nil
}

func parseLegacyStoredUserInfo(raw string) (*UserInfo, error) {
	var legacy legacyStoredUserInfo
	if err := json.Unmarshal([]byte(raw), &legacy); err != nil {
		return nil, fmt.Errorf("failed to unmarshal legacy auth session payload: %w", err)
	}
	return mapLegacyUserInfo(&legacy), nil
}

func mapLegacyUserInfo(legacy *legacyStoredUserInfo) *UserInfo {
	if legacy == nil {
		return nil
	}

	username := strings.TrimSpace(legacy.Username)
	if username == "" {
		username = strings.TrimSpace(legacy.Email)
	}

	status := strings.TrimSpace(legacy.PlanType)
	if status == "" {
		status = "active"
	}

	updatedAt := legacy.UpdatedAt
	if updatedAt.IsZero() {
		updatedAt = time.Now()
	}

	return &UserInfo{
		AccessToken:  strings.TrimSpace(legacy.AccessToken),
		RefreshToken: strings.TrimSpace(legacy.RefreshToken),
		TokenType:    strings.TrimSpace(legacy.TokenType),
		ExpiresIn:    legacy.ExpiresIn,
		ID:           legacy.UserID,
		Email:        strings.TrimSpace(legacy.Email),
		Username:     username,
		Role:         strings.TrimSpace(legacy.Role),
		Status:       status,
		CreatedAt:    strings.TrimSpace(legacy.StartTime),
		UpdatedAt:    updatedAt,
	}
}

func getLegacyAuthSessionDBPath() (string, error) {
	cacheDir, err := cache.GetCacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(cacheDir, legacyAuthSessionDBFile), nil
}

func isMissingSQLiteTableError(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(strings.ToLower(err.Error()), "no such table")
}

// GetCurrentUserInfo returns a copy of the currently loaded user info.
func GetCurrentUserInfo() *UserInfo {
	userInfoMutex.RLock()
	defer userInfoMutex.RUnlock()
	if currentUserInfo == nil {
		return nil
	}
	info := *currentUserInfo
	return &info
}

// SetCurrentUserInfo sets the in-memory user session.
func SetCurrentUserInfo(info *UserInfo) {
	userInfoMutex.Lock()
	defer userInfoMutex.Unlock()
	currentUserInfo = info
}

// ResetAuthPersistenceForTest resets auth persistence singletons for isolated tests.
func ResetAuthPersistenceForTest() {
	userInfoMutex.Lock()
	currentUserInfo = nil
	userInfoMutex.Unlock()

	authSessionDB = nil
	authSessionDBErr = nil
	authSessionDBOnce = sync.Once{}
	cache.ResetCacheDirForTest()
}
