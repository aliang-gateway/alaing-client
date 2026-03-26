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

	"nursor.org/nursorgate/common/logger"
)

// UserInfo 用户信息结构（本地存储格式）
type UserInfo struct {
	AccessToken  string    `json:"access_token"`  // 加密存储
	RefreshToken string    `json:"refresh_token"` // 加密存储
	TokenType    string    `json:"token_type,omitempty"`
	ExpiresIn    int       `json:"expires_in,omitempty"`
	Username     string    `json:"username"`
	Email        string    `json:"email,omitempty"`
	Role         string    `json:"role,omitempty"`
	UserID       int64     `json:"user_id,omitempty"`
	PlanName     string    `json:"plan_name"`
	TrafficUsed  int64     `json:"traffic_used"`
	TrafficTotal int64     `json:"traffic_total"`
	AIAskUsed    int       `json:"ai_ask_used"`
	AIAskTotal   int       `json:"ai_ask_total"`
	StartTime    string    `json:"start_time"`
	EndTime      string    `json:"end_time"`
	PlanType     string    `json:"plan_type"`
	InnerToken   string    `json:"inner_token"` // 加密存储
	UpdatedAt    time.Time `json:"updated_at"`  // 最后更新时间
}

// ActivateResponse 激活API响应结构
type ActivateResponse struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		User         struct {
			PlanName     string `json:"plan_name"`
			TrafficUsed  int64  `json:"traffic_used"`
			TrafficTotal int64  `json:"traffic_total"`
			AIAskUsed    int    `json:"ai_ask_used"`
			AIAskTotal   int    `json:"ai_ask_total"`
			StartTime    string `json:"start_time"`
			EndTime      string `json:"end_time"`
			PlanType     string `json:"plan_type"`
			InnerToken   string `json:"inner_token"`
			Username     string `json:"username"`
		} `json:"user"`
	} `json:"data"`
}

// RefreshResponse 刷新API响应结构
// refresh API 返回的 data 直接包含用户信息，不包含 access_token 和 refresh_token
type RefreshResponse struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data struct {
		PlanName     string `json:"plan_name"`
		TrafficUsed  int64  `json:"traffic_used"`
		TrafficTotal int64  `json:"traffic_total"`
		AIAskUsed    int    `json:"ai_ask_used"`
		AIAskTotal   int    `json:"ai_ask_total"`
		StartTime    string `json:"start_time"`
		EndTime      string `json:"end_time"`
		PlanType     string `json:"plan_type"`
		InnerToken   string `json:"inner_token"`
		Username     string `json:"username"`
	} `json:"data"`
}

var (
	userInfoMutex     sync.RWMutex
	currentUserInfo   *UserInfo
	authSessionDBOnce sync.Once
	authSessionDB     *gorm.DB
	authSessionDBErr  error
)

const (
	authSessionDBFile   = "auth_session.db"
	authSessionRecordID = 1
)

type authSessionRecord struct {
	ID        uint      `gorm:"primaryKey"`
	Data      string    `gorm:"type:text;not null"`
	UpdatedAt time.Time `gorm:"autoUpdateTime"`
}

// GetUserInfoPath 获取用户信息文件路径
func GetUserInfoPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	configDir := filepath.Join(homeDir, ".aliang")
	logger.Debug(fmt.Sprintf("get user info path: %s", configDir))
	return filepath.Join(configDir, "userinfo.json"), nil
}

func GetAuthSessionDBPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	return filepath.Join(homeDir, ".aliang", authSessionDBFile), nil
}

// SaveUserInfo 保存用户信息到本地（整文件加密）
func SaveUserInfo(info *UserInfo) error {
	if info == nil {
		return fmt.Errorf("user info cannot be nil")
	}

	info.UpdatedAt = time.Now()

	if err := saveUserInfoToSQLite(info); err != nil {
		return fmt.Errorf("failed to persist user info to sqlite: %w", err)
	}

	logger.Debug("User info saved successfully (sqlite)")

	userInfoMutex.Lock()
	copyInfo := *info
	currentUserInfo = &copyInfo
	userInfoMutex.Unlock()

	return nil
}

func LoadUserInfo() (*UserInfo, error) {
	sqliteInfo, err := loadUserInfoFromSQLite()
	if err == nil {
		userInfoMutex.Lock()
		copyInfo := *sqliteInfo
		currentUserInfo = &copyInfo
		userInfoMutex.Unlock()
		return sqliteInfo, nil
	}

	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("failed to load user info from sqlite: %w", err)
	}

	legacyInfo, legacyErr := loadLegacyUserInfoFromFile()
	if legacyErr != nil {
		return nil, fmt.Errorf("no persisted user info found: %w", legacyErr)
	}

	if saveErr := saveUserInfoToSQLite(legacyInfo); saveErr != nil {
		return nil, fmt.Errorf("failed to migrate legacy user info to sqlite: %w", saveErr)
	}

	logger.Info("Migrated legacy user info file into sqlite auth session store")

	userInfoMutex.Lock()
	copyInfo := *legacyInfo
	currentUserInfo = &copyInfo
	userInfoMutex.Unlock()

	return legacyInfo, nil
}

// UpdateUserInfo 更新用户信息并保存
func UpdateUserInfo(info *UserInfo) error {
	return SaveUserInfo(info)
}

// DeleteUserInfo 删除本地用户信息
func DeleteUserInfo() error {
	if err := deleteUserInfoFromSQLite(); err != nil {
		return err
	}

	filePath, err := GetUserInfoPath()
	if err == nil {
		if err := os.Remove(filePath); err != nil && !os.IsNotExist(err) {
			logger.Warn(fmt.Sprintf("Failed to delete legacy user info file: %v", err))
		}
	}

	userInfoMutex.Lock()
	currentUserInfo = nil
	userInfoMutex.Unlock()

	return nil
}

func HasPersistedUserInfo() (bool, error) {
	dbPath, err := GetAuthSessionDBPath()
	if err != nil {
		return false, err
	}

	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		filePath, pathErr := GetUserInfoPath()
		if pathErr != nil {
			return false, nil
		}
		if _, statErr := os.Stat(filePath); statErr == nil {
			return true, nil
		}
		return false, nil
	}

	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		return false, err
	}

	var count int64
	if err := db.Model(&authSessionRecord{}).Where("id = ?", authSessionRecordID).Count(&count).Error; err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "no such table") {
			return false, nil
		}
		return false, err
	}

	if count > 0 {
		return true, nil
	}

	filePath, pathErr := GetUserInfoPath()
	if pathErr != nil {
		return false, nil
	}

	if _, err := os.Stat(filePath); err == nil {
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

		if err := db.AutoMigrate(&authSessionRecord{}); err != nil {
			authSessionDBErr = fmt.Errorf("failed to migrate auth session table: %w", err)
			return
		}

		authSessionDB = db
	})

	return authSessionDB, authSessionDBErr
}

func saveUserInfoToSQLite(info *UserInfo) error {
	db, err := getAuthSessionDB()
	if err != nil {
		return err
	}

	data, err := json.Marshal(info)
	if err != nil {
		return fmt.Errorf("failed to marshal user info: %w", err)
	}

	record := authSessionRecord{
		ID:        authSessionRecordID,
		Data:      string(data),
		UpdatedAt: info.UpdatedAt,
	}

	return db.Save(&record).Error
}

func loadUserInfoFromSQLite() (*UserInfo, error) {
	db, err := getAuthSessionDB()
	if err != nil {
		return nil, err
	}

	var record authSessionRecord
	if err := db.First(&record, "id = ?", authSessionRecordID).Error; err != nil {
		return nil, err
	}

	info := &UserInfo{}
	if err := json.Unmarshal([]byte(record.Data), info); err != nil {
		return nil, fmt.Errorf("failed to unmarshal user info: %w", err)
	}

	return info, nil
}

func deleteUserInfoFromSQLite() error {
	db, err := getAuthSessionDB()
	if err != nil {
		return err
	}

	if err := db.Delete(&authSessionRecord{}, "id = ?", authSessionRecordID).Error; err != nil {
		return fmt.Errorf("failed to delete user info from sqlite: %w", err)
	}

	return nil
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

	return decryptedInfo, nil
}

// GetCurrentUserInfo 获取当前加载的用户信息（内存）
func GetCurrentUserInfo() *UserInfo {
	userInfoMutex.RLock()
	defer userInfoMutex.RUnlock()
	if currentUserInfo == nil {
		return nil
	}
	// 返回副本以避免并发修改
	info := *currentUserInfo
	return &info
}

// SetCurrentUserInfo 设置当前用户信息（内存）
func SetCurrentUserInfo(info *UserInfo) {
	userInfoMutex.Lock()
	defer userInfoMutex.Unlock()
	currentUserInfo = info
}

func ResetAuthPersistenceForTest() {
	userInfoMutex.Lock()
	currentUserInfo = nil
	userInfoMutex.Unlock()

	authSessionDB = nil
	authSessionDBErr = nil
	authSessionDBOnce = sync.Once{}
}
