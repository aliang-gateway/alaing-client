package cmd

import (
	"fmt"

	"nursor.org/nursorgate/common/logger"
	auth "nursor.org/nursorgate/processor/auth"
	"nursor.org/nursorgate/processor/config"
	"nursor.org/nursorgate/processor/inbound"
)

// InitializeUser 初始化用户信息和Token激活
// 这个函数在启动时调用，负责处理Token激活和加载本地用户信息
func InitializeUser(token string) {
	// Step 1: 如果提供了token参数，尝试激活
	if token != "" {
		logger.Info("Token provided, attempting to activate...")
		userInfo, err := auth.ActivateToken(token)

		if err == nil {
			// 激活成功（可能是远程激活或本地回退）
			logger.Info(fmt.Sprintf("User activated successfully: %s (Plan: %s)", userInfo.Username, userInfo.PlanName))
			// 标记为有本地用户信息（激活成功后会自动保存到本地）
			config.SetHasLocalUserInfo(true)
		} else {
			// 激活失败
			logger.Warn(fmt.Sprintf("Token activation failed: %v", err))
			config.SetHasLocalUserInfo(false)
		}
	} else {
		// Step 2: 没有提供token，尝试加载本地用户信息
		if err := loadLocalUserInfo(); err == nil {
			userInfo := auth.GetCurrentUserInfo()
			if userInfo != nil {
				logger.Info(fmt.Sprintf("Local user info loaded successfully: %s (Plan: %s)", userInfo.Username, userInfo.PlanName))
				// 标记为有本地用户信息
				config.SetHasLocalUserInfo(true)
			}
		} else {
			logger.Debug("No local user info found, starting without user authentication")
			// 标记为没有本地用户信息
			config.SetHasLocalUserInfo(false)
		}
	}
}

// loadLocalUserInfo 加载本地用户信息
func loadLocalUserInfo() error {
	userInfo, err := auth.LoadUserInfo()
	if err != nil {
		return err
	}

	// 更新运行时状态
	auth.SetInnerToken(userInfo.InnerToken)

	// 加载完本地用户信息后，尝试更新Door代理信息
	if userInfo.AccessToken != "" {
		if err := inbound.UpdateDoorProxies(userInfo.AccessToken); err != nil {
			logger.Warn(fmt.Sprintf("Failed to update inbound proxies on startup: %v", err))
			// 不返回错误，允许系统继续启动
		} else {
			logger.Info("Successfully updated inbound proxies on startup")
		}
	}

	return nil
}
