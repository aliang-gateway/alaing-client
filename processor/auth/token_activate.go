package user

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"nursor.org/nursorgate/common/logger"
	"nursor.org/nursorgate/processor/inbound"
)

const (
	// 激活API地址
	activateAPIURL = "https://api.nonelane.com/api/user/auth/new/activate"
	// API调用超时
	apiTimeout = 10 * time.Second
)

// ActivateToken 激活Token并获取用户信息
// 如果激活失败，尝试加载本地之前保存的用户信息
func ActivateToken(token string) (*UserInfo, error) {
	if token == "" {
		return nil, fmt.Errorf("token cannot be empty")
	}

	logger.Info(fmt.Sprintf("Activating token: %s...", maskToken(token)))

	// 调用外部激活API
	userInfo, err := callActivateAPI(token)
	if err == nil {
		logger.Info(fmt.Sprintf("Token activated successfully. User: %s", userInfo.Username))

		// 保存到本地
		if err := SaveUserInfo(userInfo); err != nil {
			logger.Warn(fmt.Sprintf("Failed to save user info locally: %v", err))
			// 不返回错误，因为激活已经成功
		}

		// 更新运行时状态
		SetInnerToken(userInfo.InnerToken)

		// 获取并更新Door代理信息（网络优先策略）
		if err := inbound.UpdateDoorProxies(userInfo.AccessToken); err != nil {
			logger.Warn(fmt.Sprintf("Failed to update inbound proxies: %v", err))
			// 不返回错误，因为激活已经成功，缺少代理不是致命错误
		} else {
			logger.Info("Successfully updated inbound proxies after token activation")
		}

		// 启动定时刷新
		startTokenRefresh()

		return userInfo, nil
	}

	// 激活失败，尝试加载本地用户信息
	logger.Warn(fmt.Sprintf("Token activation failed: %v, trying to load local user info", err))

	localUserInfo, err := LoadUserInfo()
	if err == nil {
		logger.Info("Using locally saved user info as fallback")

		// 更新运行时状态
		SetInnerToken(localUserInfo.InnerToken)

		// 尝试更新Door代理信息（使用本地用户的AccessToken）
		if localUserInfo.AccessToken != "" {
			if err := inbound.UpdateDoorProxies(localUserInfo.AccessToken); err != nil {
				logger.Warn(fmt.Sprintf("Failed to update inbound proxies from fallback user info: %v", err))
				// 不返回错误，继续启动
			} else {
				logger.Info("Successfully updated inbound proxies from fallback user info")
			}
		}

		// 启动定时刷新（尝试在后续刷新时重新激活）
		startTokenRefresh()

		return localUserInfo, nil
	}

	logger.Error(fmt.Sprintf("No local user info found, activation failed: %v", err))
	return nil, fmt.Errorf("failed to activate token and no local user info found: %w", err)
}

// callActivateAPI 调用外部激活API
func callActivateAPI(token string) (*UserInfo, error) {
	// 构建请求
	requestBody := map[string]string{
		"access_token": token,
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", activateAPIURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

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
	var response ActivateResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// 检查API响应码
	if response.Code != 0 {
		return nil, fmt.Errorf("api error: %s (code: %d)", response.Msg, response.Code)
	}

	// 提取用户信息
	userInfo := &UserInfo{
		AccessToken:  response.Data.AccessToken,
		RefreshToken: response.Data.RefreshToken,
		Username:     response.Data.User.Username,
		PlanName:     response.Data.User.PlanName,
		TrafficUsed:  response.Data.User.TrafficUsed,
		TrafficTotal: response.Data.User.TrafficTotal,
		AIAskUsed:    response.Data.User.AIAskUsed,
		AIAskTotal:   response.Data.User.AIAskTotal,
		StartTime:    response.Data.User.StartTime,
		EndTime:      response.Data.User.EndTime,
		PlanType:     response.Data.User.PlanType,
		InnerToken:   response.Data.User.InnerToken,
		UpdatedAt:    time.Now(),
	}

	return userInfo, nil
}

// maskToken 掩盖Token用于日志显示（只显示前后几个字符）
func maskToken(token string) string {
	if len(token) <= 8 {
		return "***"
	}
	return token[:4] + "..." + token[len(token)-4:]
}

var (
	tokenRefresher *TokenRefresher
)

// startTokenRefresh 启动定时刷新
func startTokenRefresh() {
	if tokenRefresher != nil && tokenRefresher.IsRunning() {
		return // 已经在运行
	}

	tokenRefresher = NewTokenRefresher()
	if err := tokenRefresher.Start(); err != nil {
		logger.Warn(fmt.Sprintf("Failed to start token refresher: %v", err))
	}
}

// StopTokenRefresh 停止定时刷新
func StopTokenRefresh() {
	if tokenRefresher != nil {
		if err := tokenRefresher.Stop(); err != nil {
			logger.Warn(fmt.Sprintf("Failed to stop token refresher: %v", err))
		}
	}
}

// GetTokenRefresher 获取Token刷新器实例
func GetTokenRefresher() *TokenRefresher {
	return tokenRefresher
}
