package user

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"nursor.org/nursorgate/common/logger"
	"nursor.org/nursorgate/processor/config"
)

// TokenRefresher 定时刷新用户Token和信息
type TokenRefresher struct {
	ticker          *time.Ticker
	done            chan bool
	lastError       error
	lastErrorTime   time.Time
	refreshDuration time.Duration
	mu              sync.RWMutex
	isRunning       bool
}

const (
	// 默认刷新间隔（1分钟）
	defaultRefreshDuration = 1 * time.Minute
)

// NewTokenRefresher 创建新的Token刷新器
func NewTokenRefresher() *TokenRefresher {
	return &TokenRefresher{
		refreshDuration: defaultRefreshDuration,
		done:            make(chan bool, 1),
		isRunning:       false,
	}
}

// Start 启动定时刷新
func (tr *TokenRefresher) Start() error {
	tr.mu.Lock()
	defer tr.mu.Unlock()

	if tr.isRunning {
		return nil // 已经在运行
	}

	tr.ticker = time.NewTicker(tr.refreshDuration)
	tr.isRunning = true

	// 在后台运行刷新任务
	go tr.refreshLoop()

	logger.Info(fmt.Sprintf("Token refresher started (interval: %v)", tr.refreshDuration))
	return nil
}

// Stop 停止定时刷新
func (tr *TokenRefresher) Stop() error {
	tr.mu.Lock()
	defer tr.mu.Unlock()

	if !tr.isRunning {
		return nil // 已经停止
	}

	tr.isRunning = false
	if tr.ticker != nil {
		tr.ticker.Stop()
	}

	// 发送停止信号
	select {
	case tr.done <- true:
	default:
	}

	logger.Info("Token refresher stopped")
	return nil
}

// IsRunning 检查是否在运行
func (tr *TokenRefresher) IsRunning() bool {
	tr.mu.RLock()
	defer tr.mu.RUnlock()
	return tr.isRunning
}

// RefreshNow 立即刷新一次
func (tr *TokenRefresher) RefreshNow() error {
	return tr.refreshUserInfo()
}

// GetLastError 获取最后的刷新错误
func (tr *TokenRefresher) GetLastError() error {
	tr.mu.RLock()
	defer tr.mu.RUnlock()
	return tr.lastError
}

// GetLastErrorTime 获取最后错误的时间
func (tr *TokenRefresher) GetLastErrorTime() time.Time {
	tr.mu.RLock()
	defer tr.mu.RUnlock()
	return tr.lastErrorTime
}

// refreshLoop 刷新循环
func (tr *TokenRefresher) refreshLoop() {
	for {
		select {
		case <-tr.done:
			return
		case <-tr.ticker.C:
			if err := tr.refreshUserInfo(); err != nil {
				tr.mu.Lock()
				tr.lastError = err
				tr.lastErrorTime = time.Now()
				tr.mu.Unlock()

				logger.Warn(fmt.Sprintf("Failed to refresh user info: %v", err))
				// 继续运行，下次刷新时重试
			} else {
				tr.mu.Lock()
				tr.lastError = nil
				tr.mu.Unlock()

				logger.Debug("User info refreshed successfully")
			}
		}
	}
}

// refreshUserInfo 刷新用户信息
func (tr *TokenRefresher) refreshUserInfo() error {
	// 获取当前用户信息
	currentInfo := GetCurrentUserInfo()
	if currentInfo == nil {
		return fmt.Errorf("no user info to refresh")
	}

	if currentInfo.AccessToken == "" {
		return fmt.Errorf("no access token available")
	}

	// 调用刷新API，使用 AccessToken (JWT) 作为 Bearer token
	newInfo, err := callRefreshTokenAPI(currentInfo.AccessToken)
	if err != nil {
		return fmt.Errorf("failed to refresh token: %w", err)
	}

	// 保存新的用户信息
	if err := SaveUserInfo(newInfo); err != nil {
		return fmt.Errorf("failed to save refreshed user info: %w", err)
	}

	// 更新运行时状态
	SetInnerToken(newInfo.InnerToken)

	return nil
}

// callRefreshTokenAPI 调用Token刷新API
// accessToken 是 JWT token，需要在 Authorization header 中作为 Bearer token 传递
func callRefreshTokenAPI(accessToken string) (*UserInfo, error) {
	if accessToken == "" {
		return nil, fmt.Errorf("access token cannot be empty")
	}

	// 获取 URL 构建器
	urlBuilder, err := config.NewURLBuilder()
	if err != nil {
		return nil, err
	}

	// 获取并验证 URL
	planStatusURL, err := urlBuilder.GetPlanStatusURL()
	if err != nil {
		return nil, err
	}

	// 创建 GET 请求（不需要 body）
	req, err := http.NewRequest("GET", planStatusURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// 设置 Authorization header，使用 Bearer token
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", accessToken))
	req.Header.Set("Content-Type", "application/json")

	// 发送请求
	client := &http.Client{
		Timeout: apiTimeout,
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// 读取响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// 检查HTTP状态码
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("api returned status %d: %s", resp.StatusCode, string(body))
	}

	// 解析响应
	var response RefreshResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// 检查API响应码
	if response.Code != 0 {
		return nil, fmt.Errorf("api error: %s (code: %d)", response.Msg, response.Code)
	}

	// 获取当前用户信息，保留 AccessToken 和 RefreshToken（refresh API 不返回这些）
	currentInfo := GetCurrentUserInfo()
	if currentInfo == nil {
		return nil, fmt.Errorf("no current user info available to preserve tokens")
	}

	// 提取用户信息
	// refresh API 只返回用户信息，不返回新的 access_token 和 refresh_token
	// 所以我们需要保留原有的 tokens
	userInfo := &UserInfo{
		AccessToken:  currentInfo.AccessToken,  // 保留原有的 AccessToken
		RefreshToken: currentInfo.RefreshToken, // 保留原有的 RefreshToken
		Username:     response.Data.Username,
		PlanName:     response.Data.PlanName,
		TrafficUsed:  response.Data.TrafficUsed,
		TrafficTotal: response.Data.TrafficTotal,
		AIAskUsed:    response.Data.AIAskUsed,
		AIAskTotal:   response.Data.AIAskTotal,
		StartTime:    response.Data.StartTime,
		EndTime:      response.Data.EndTime,
		PlanType:     response.Data.PlanType,
		InnerToken:   response.Data.InnerToken,
		UpdatedAt:    time.Now(),
	}

	return userInfo, nil
}
