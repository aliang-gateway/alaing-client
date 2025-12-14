package user

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"nursor.org/nursorgate/common/logger"
)

// UserInfo 用户信息结构（本地存储格式）
type UserInfo struct {
	AccessToken  string    `json:"access_token"`  // 加密存储
	RefreshToken string    `json:"refresh_token"` // 加密存储
	Username     string    `json:"username"`
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
	userInfoMutex   sync.RWMutex
	currentUserInfo *UserInfo
)

// GetUserInfoPath 获取用户信息文件路径
func GetUserInfoPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	configDir := filepath.Join(homeDir, ".nonelane")
	logger.Debug("get user info path: %s", configDir)
	return filepath.Join(configDir, "userinfo.json"), nil
}

// ensureConfigDir 确保配置目录存在
func ensureConfigDir() error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	configDir := filepath.Join(homeDir, ".nonelane")
	if err := os.MkdirAll(configDir, 0700); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	return nil
}

// SaveUserInfo 保存用户信息到本地（整文件加密）
func SaveUserInfo(info *UserInfo) error {
	if info == nil {
		return fmt.Errorf("user info cannot be nil")
	}

	// 确保配置目录存在
	if err := ensureConfigDir(); err != nil {
		return err
	}

	// 更新时间戳
	info.UpdatedAt = time.Now()

	// 使用整文件加密（新格式）
	encryptedData, err := EncryptUserInfoFile(info)
	if err != nil {
		return fmt.Errorf("failed to encrypt user info file: %w", err)
	}

	// 获取文件路径
	filePath, err := GetUserInfoPath()
	if err != nil {
		return err
	}

	// 写入文件（权限0600只有所有者可读写）
	if err := os.WriteFile(filePath, encryptedData, 0600); err != nil {
		return fmt.Errorf("failed to write user info file: %w", err)
	}

	logger.Debug("User info saved successfully (whole-file encryption)")

	// 更新内存中的用户信息
	userInfoMutex.Lock()
	currentUserInfo = info
	userInfoMutex.Unlock()

	return nil
}

// LoadUserInfo 从本地加载用户信息（自动解密，支持新旧格式迁移）
func LoadUserInfo() (*UserInfo, error) {
	filePath, err := GetUserInfoPath()
	if err != nil {
		return nil, err
	}

	// 检查文件是否存在
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil, fmt.Errorf("user info file does not exist: %s", filePath)
	}

	// 读取文件
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read user info file: %w", err)
	}

	// 尝试新格式（整文件加密）
	decryptedInfo, err := DecryptUserInfoFile(data)
	if err == nil {
		logger.Debug("User info loaded successfully (whole-file encryption format)")

		// 更新内存中的用户信息
		userInfoMutex.Lock()
		currentUserInfo = decryptedInfo
		userInfoMutex.Unlock()

		return decryptedInfo, nil
	}

	// 新格式失败，尝试旧格式（字段级加密）
	logger.Debug("New format decryption failed, attempting old format")

	// 检查是否是旧格式（JSON结构）
	var encryptedInfo UserInfo
	if err := json.Unmarshal(data, &encryptedInfo); err != nil {
		return nil, fmt.Errorf("failed to unmarshal user info (both formats): %w", err)
	}

	// 尝试解密旧格式的字段
	accessToken, accessTokenErr := DecryptField(encryptedInfo.AccessToken)
	refreshToken, refreshTokenErr := DecryptField(encryptedInfo.RefreshToken)
	innerToken, innerTokenErr := DecryptField(encryptedInfo.InnerToken)

	// 如果字段级解密失败，说明不是旧格式
	if accessTokenErr != nil || refreshTokenErr != nil || innerTokenErr != nil {
		return nil, fmt.Errorf("user info file format not recognized (tried both new and old formats)")
	}

	// 成功解密旧格式，创建解密后的用户信息
	decryptedInfo = &UserInfo{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		Username:     encryptedInfo.Username,
		PlanName:     encryptedInfo.PlanName,
		TrafficUsed:  encryptedInfo.TrafficUsed,
		TrafficTotal: encryptedInfo.TrafficTotal,
		AIAskUsed:    encryptedInfo.AIAskUsed,
		AIAskTotal:   encryptedInfo.AIAskTotal,
		StartTime:    encryptedInfo.StartTime,
		EndTime:      encryptedInfo.EndTime,
		PlanType:     encryptedInfo.PlanType,
		InnerToken:   innerToken,
		UpdatedAt:    encryptedInfo.UpdatedAt,
	}

	logger.Info("User info loaded from old format (field-level encryption), migrating to new format...")

	// 自动迁移到新格式
	if err := SaveUserInfo(decryptedInfo); err != nil {
		logger.Warn(fmt.Sprintf("Failed to migrate user info to new format: %v", err))
		// 继续返回解密的用户信息，迁移失败不应该阻止启动
	} else {
		logger.Info("User info successfully migrated to new format (whole-file encryption)")
	}

	// 更新内存中的用户信息
	userInfoMutex.Lock()
	currentUserInfo = decryptedInfo
	userInfoMutex.Unlock()

	return decryptedInfo, nil
}

// UpdateUserInfo 更新用户信息并保存
func UpdateUserInfo(info *UserInfo) error {
	return SaveUserInfo(info)
}

// DeleteUserInfo 删除本地用户信息
func DeleteUserInfo() error {
	filePath, err := GetUserInfoPath()
	if err != nil {
		return err
	}

	// 删除文件
	if err := os.Remove(filePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete user info file: %w", err)
	}

	// 清空内存中的用户信息
	userInfoMutex.Lock()
	currentUserInfo = nil
	userInfoMutex.Unlock()

	return nil
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
