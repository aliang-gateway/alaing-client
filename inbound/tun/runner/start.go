package runner

import (
	"fmt"
	"os/signal"
	"syscall"
	"time"

	"aliang.one/nursorgate/common/logger"
	tunDevice "aliang.one/nursorgate/inbound/tun/device/tun"
	"aliang.one/nursorgate/inbound/tun/engine"
	"aliang.one/nursorgate/processor/config"
	"go.uber.org/automaxprocs/maxprocs"
)

// StartupState 追踪 TUN 启动过程中已完成的步骤
type StartupState struct {
	monitorStarted      bool
	engineStarted       bool
	interfaceConfigured bool
	routesConfigured    bool
}

func Start() {
	ResetStartupProgress()
	UpdateStartupProgress("starting", "requested", 5, "Preparing TUN startup.", "", false)

	defer func() {
		if r := recover(); r != nil {
			logger.Error(fmt.Sprintf("Recovered from panic in Start: %v", r))
			FailStartupProgress("panic", fmt.Errorf("Recovered from panic in Start: %v", r))
			RunStatusChan <- map[string]string{"status": "failed", "message": fmt.Sprintf("Recovered from panic in Start: %v", r)}
		}
	}()

	maxprocs.Set(maxprocs.Logger(func(string, ...any) {}))

	defaultConfig = GetDefaultDeviceConfiguration()
	tunDevice.SetCreateTUNAttemptHook(func(name string, attempt int, maxRetries int, err error) {
		SetStartupRetryInfo(attempt, maxRetries)
		if err != nil {
			errMsg := fmt.Sprintf("Create TUN attempt %d/%d failed: %v", attempt, maxRetries, err)
			AppendStartupError(errMsg)
			UpdateStartupProgress("starting", "creating_tun", 25, fmt.Sprintf("Creating the virtual TUN adapter (attempt %d/%d).", attempt, maxRetries), errMsg, isPermissionLikeError(errMsg))
			return
		}
		UpdateStartupProgress("starting", "creating_tun", 35, fmt.Sprintf("Virtual TUN adapter created after %d/%d attempt(s).", attempt, maxRetries), "", false)
	})
	defer tunDevice.SetCreateTUNAttemptHook(nil)

	// 追踪启动状态
	state := &StartupState{}

	// 使用带回滚的启动流程
	if err := startWithRollback(state); err != nil {
		logger.Error(fmt.Sprintf("TUN 启动失败: %v", err))
		AppendStartupError(err.Error())
		FailStartupProgress(GetStartupProgress().Phase, err)
		// 执行回滚
		rollbackStartup(state)
		RunStatusChan <- map[string]string{"status": "failed", "message": err.Error()}
		return
	}

	logger.Info("TUN 服务启动成功，设备名称: ", defaultConfig.Interface)
	CompleteStartupProgress("TUN service started successfully.")
	RunStatusChan <- map[string]string{"status": "success", "message": "TUN service started successfully"}

	signal.Notify(TunSignal, syscall.SIGINT, syscall.SIGTERM)
	<-TunSignal

	// 收到信号后调用 Stop
	stopTun()
}

// startWithRollback 执行启动步骤并追踪状态
func startWithRollback(state *StartupState) error {
	// Step 1: 添加设备状态监控
	UpdateStartupProgress("starting", "monitoring_device", 10, "Starting TUN device monitoring.", "", false)
	go monitorTunDevice(defaultConfig.Device)
	state.monitorStarted = true

	// Step 2: 插入配置并启动 engine
	UpdateStartupProgress("starting", "creating_tun", 25, "Creating the virtual TUN adapter.", "", false)
	config.Insert(&defaultConfig)
	if err := engine.Start(); err != nil {
		AppendStartupError(fmt.Sprintf("engine 启动失败: %v", err))
		return fmt.Errorf("engine 启动失败: %w", err)
	}
	state.engineStarted = true

	// NOTE: Rule engine initialization has been MOVED to cmd/start.go:InitializeGlobalRuleEngine()
	// This ensures the singleton rule engine is initialized only ONCE at startup
	// Previously this was duplicated in both HTTP mode and TUN mode
	logger.Info("TUN: Rule engine has been initialized globally (see cmd/start.go)")

	// Step 3: 获取默认网关
	UpdateStartupProgress("starting", "resolving_gateway", 45, "Resolving the current default gateway.", "", false)
	_dfgw, err := GetDefaultGatewayForTUN()
	if err != nil {
		AppendStartupError(fmt.Sprintf("获取默认网关失败: %v", err))
		return fmt.Errorf("获取默认网关失败: %w", err)
	}
	defaultGateway = _dfgw

	// Step 4: 配置 TUN 接口
	UpdateStartupProgress("starting", "configuring_interface", 60, "Configuring the virtual adapter interface.", "", false)
	if err := ConfigureTunInterface(defaultConfig.Device); err != nil {
		AppendStartupError(fmt.Sprintf("配置 TUN 接口失败: %v", err))
		return fmt.Errorf("配置 TUN 接口失败: %w", err)
	}
	state.interfaceConfigured = true

	// Step 5: 等待设备就绪（最多等待 10 秒）
	UpdateStartupProgress("starting", "waiting_device_ready", 78, "Waiting for the virtual adapter to become ready.", "", false)
	if err := waitForTunDeviceReady(defaultConfig.Device, 10*time.Second); err != nil {
		AppendStartupError(fmt.Sprintf("等待 TUN 设备就绪失败: %v", err))
		return fmt.Errorf("等待 TUN 设备就绪失败: %w", err)
	}

	// Step 6: 配置路由（最关键的步骤）
	UpdateStartupProgress("starting", "configuring_routes", 90, "Configuring TUN routing rules.", "", false)
	if err := ConfigureTunRoute(); err != nil {
		AppendStartupError(fmt.Sprintf("配置 TUN 路由失败: %v", err))
		return fmt.Errorf("配置 TUN 路由失败: %w", err)
	}
	state.routesConfigured = true

	return nil
}

// rollbackStartup 回滚已完成的启动步骤（按逆序清理）
func rollbackStartup(state *StartupState) {
	logger.Info("执行 TUN 启动回滚...")

	// 按逆序回滚
	// Step 6 回滚: 清理路由
	if state.routesConfigured {
		logger.Info("回滚: 清理 TUN 路由")
		if err := CleanupTunRoute(); err != nil {
			logger.Error(fmt.Sprintf("回滚路由配置失败: %v", err))
		} else {
			logger.Info("✓ 路由回滚成功")
		}
	}

	// Step 4 回滚: 清理接口
	if state.interfaceConfigured {
		logger.Info("回滚: 清理 TUN 接口")
		if err := CleanupTunInterface(defaultConfig.Device); err != nil {
			logger.Error(fmt.Sprintf("回滚接口配置失败: %v", err))
		} else {
			logger.Info("✓ 接口回滚成功")
		}
	}

	// Step 2 回滚: 停止 engine
	if state.engineStarted {
		logger.Info("回滚: 停止 engine")
		engine.Stop()
		logger.Info("✓ Engine 停止成功")
	}

	// Note: 监控 goroutine 会在程序结束时自动终止，无需显式清理

	logger.Info("TUN 启动回滚完成")
}

func Stop() {
	TunSignal <- syscall.SIGTERM // 或其他自定义信号
}

// initializeRuleEngineForTUN has been REMOVED - replaced by cmd/start.go:InitializeGlobalRuleEngine()
// This function was causing duplicate initialization of the singleton rule engine
// See: cmd/start.go for the new centralized initialization
