package runner

import (
	"fmt"
	"os/signal"
	"syscall"
	"time"

	"go.uber.org/automaxprocs/maxprocs"
	"nursor.org/nursorgate/common/logger"
	"nursor.org/nursorgate/common/model"
	"nursor.org/nursorgate/inbound/tun/engine"
	"nursor.org/nursorgate/processor/config"
)

// StartupState 追踪 TUN 启动过程中已完成的步骤
type StartupState struct {
	monitorStarted      bool
	engineStarted       bool
	interfaceConfigured bool
	routesConfigured    bool
}

func Start() {
	defer func() {
		if r := recover(); r != nil {
			logger.Error(fmt.Sprintf("Recovered from panic in Start: %v", r))
			RunStatusChan <- map[string]string{"status": "failed", "message": fmt.Sprintf("Recovered from panic in Start: %v", r)}
		}
	}()

	domains := model.NewAllowProxyDomain()
	logger.Info(fmt.Sprintf("domain is: %v", domains))

	maxprocs.Set(maxprocs.Logger(func(string, ...any) {}))

	defaultConfig = GetDefaultDeviceConfiguration()

	// 追踪启动状态
	state := &StartupState{}

	// 使用带回滚的启动流程
	if err := startWithRollback(state); err != nil {
		logger.Error(fmt.Sprintf("TUN 启动失败: %v", err))
		// 执行回滚
		rollbackStartup(state)
		RunStatusChan <- map[string]string{"status": "failed", "message": err.Error()}
		return
	}

	logger.Info("TUN 服务启动成功，设备名称: ", defaultConfig.Interface)
	RunStatusChan <- map[string]string{"status": "success", "message": "TUN service started successfully"}

	signal.Notify(TunSignal, syscall.SIGINT, syscall.SIGTERM)
	<-TunSignal

	// 收到信号后调用 Stop
	stopTun()
}

// startWithRollback 执行启动步骤并追踪状态
func startWithRollback(state *StartupState) error {
	// Step 1: 添加设备状态监控
	go monitorTunDevice(defaultConfig.Device)
	state.monitorStarted = true

	// Step 2: 插入配置并启动 engine
	config.Insert(&defaultConfig)
	if err := engine.Start(); err != nil {
		return fmt.Errorf("engine 启动失败: %w", err)
	}
	state.engineStarted = true

	// NOTE: Rule engine initialization has been MOVED to cmd/start.go:InitializeGlobalRuleEngine()
	// This ensures the singleton rule engine is initialized only ONCE at startup
	// Previously this was duplicated in both HTTP mode and TUN mode
	logger.Info("TUN: Rule engine has been initialized globally (see cmd/start.go)")

	// Step 3: 获取默认网关
	_dfgw, err := GetDefaultGatewayForTUN()
	if err != nil {
		return fmt.Errorf("获取默认网关失败: %w", err)
	}
	defaultGateway = _dfgw

	// Step 4: 配置 TUN 接口
	if err := ConfigureTunInterface(defaultConfig.Device); err != nil {
		return fmt.Errorf("配置 TUN 接口失败: %w", err)
	}
	state.interfaceConfigured = true

	// Step 5: 等待设备就绪（最多等待 10 秒）
	if err := waitForTunDeviceReady(defaultConfig.Device, 10*time.Second); err != nil {
		return fmt.Errorf("等待 TUN 设备就绪失败: %w", err)
	}

	// Step 6: 配置路由（最关键的步骤）
	if err := ConfigureTunRoute(); err != nil {
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
