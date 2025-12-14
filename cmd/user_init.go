package cmd

import (
	"fmt"

	"nursor.org/nursorgate/common/logger"
	auth "nursor.org/nursorgate/processor/auth"
	"nursor.org/nursorgate/processor/config"
	"nursor.org/nursorgate/processor/proxyserver"
	"nursor.org/nursorgate/processor/runtime"
)

// InitializeUser 初始化用户信息和Token激活
// 返回error仅在token激活失败时触发，导致启动过程失败
// 其他情况（无token或激活成功）都返回nil，允许启动继续
func InitializeUser(token string) error {
	// 获取全局启动状态
	startupState := runtime.GetStartupState()

	// Step 1: 如果提供了token参数，尝试激活
	if token != "" {
		logger.Info("Token provided, attempting to activate...")
		userInfo, err := auth.ActivateToken(token)

		if err == nil {
			// 激活成功（可能是远程激活或本地回退）
			logger.Info(fmt.Sprintf("User activated successfully: %s (Plan: %s)", userInfo.Username, userInfo.PlanName))
			// 标记为有本地用户信息（激活成功后会自动保存到本地）
			config.SetHasLocalUserInfo(true)
			startupState.SetUserInfo(userInfo)

			// 自动fetch proxyserver配置
			fetchSuccess := proxyserver.UpdateDoorProxies(userInfo.AccessToken) == nil
			startupState.SetFetchSuccess(fetchSuccess)

			if fetchSuccess {
				// Fetch成功 → 系统就绪
				startupState.SetStatus(runtime.READY)
				logger.Info("Proxyserver configuration fetched successfully, status: READY")
			} else {
				// Fetch失败但Token激活成功 → 配置态（有用户信息但代理配置不完整）
				startupState.SetStatus(runtime.CONFIGURED)
				logger.Warn("Token activated but proxyserver fetch failed, status: CONFIGURED")
			}
			return nil // 激活成功，继续启动
		} else {
			// 激活失败 → 返回error导致启动失败
			errMsg := fmt.Sprintf("Token activation failed: %v", err)
			logger.Error(errMsg)
			config.SetHasLocalUserInfo(false)
			startupState.SetFetchSuccess(false)
			startupState.SetStatus(runtime.UNCONFIGURED)
			return fmt.Errorf(errMsg) // 返回错误，导致启动失败
		}
	} else {
		// Step 2: 没有提供token，尝试加载本地用户信息
		if err := loadLocalUserInfo(); err == nil {
			userInfo := auth.GetCurrentUserInfo()
			if userInfo != nil {
				logger.Info(fmt.Sprintf("Local user info loaded successfully: %s (Plan: %s)", userInfo.Username, userInfo.PlanName))
				// 标记为有本地用户信息
				config.SetHasLocalUserInfo(true)
				startupState.SetUserInfo(userInfo)
				logger.Debug("Local user info loaded and proxyserver fetch handled in loadLocalUserInfo()")
			}
		} else {
			logger.Debug("No local user info found, starting without user authentication")
			// 标记为没有本地用户信息
			config.SetHasLocalUserInfo(false)
			startupState.SetFetchSuccess(false)
			// 保持UNCONFIGURED状态（由determineInitialStartupStatus()已设置）
		}
		return nil // 无token时总是继续启动（无论是否加载本地用户成功）
	}
}

// loadLocalUserInfo 加载本地用户信息
// 返回error仅指加载失败，不指fetch失败（fetch失败允许系统继续启动）
func loadLocalUserInfo() error {
	userInfo, err := auth.LoadUserInfo()
	if err != nil {
		return err
	}

	// 更新运行时状态
	auth.SetInnerToken(userInfo.InnerToken)

	// 获取启动状态以跟踪fetch结果
	startupState := runtime.GetStartupState()

	// 加载完本地用户信息后，尝试更新Door代理信息
	if userInfo.AccessToken != "" {
		fetchErr := proxyserver.UpdateDoorProxies(userInfo.AccessToken)
		fetchSuccess := fetchErr == nil

		startupState.SetFetchSuccess(fetchSuccess)

		if fetchSuccess {
			logger.Info("Successfully updated proxyserver proxies on startup, status: READY")
			startupState.SetStatus(runtime.READY)
		} else {
			logger.Warn(fmt.Sprintf("Failed to update proxyserver proxies on startup: %v", fetchErr))
			logger.Warn("System has local user info but proxyserver fetch failed, status: CONFIGURED")
			startupState.SetStatus(runtime.CONFIGURED)
			// 不返回错误，允许系统继续启动
		}
	} else {
		logger.Warn("Local user info loaded but AccessToken is empty, status: CONFIGURED")
		startupState.SetStatus(runtime.CONFIGURED)
	}

	return nil
}
