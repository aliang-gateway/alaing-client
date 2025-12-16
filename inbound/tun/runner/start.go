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
	"nursor.org/nursorgate/processor/rules"
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

	// Step 2.5: 初始化 Rule Engine（包括 DNS cache）
	if err := initializeRuleEngineForTUN(); err != nil {
		logger.Warn(fmt.Sprintf("TUN: Rule engine 初始化失败（非致命）: %v", err))
		// 不返回错误，允许 TUN 继续启动（降级为无 cache 模式）
	}

	// Step 3: 获取默认网关
	_dfgw, err := GetDefaultGateway()
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

// initializeRuleEngineForTUN 为 TUN 模式初始化 Rule Engine（包括 DNS cache）
func initializeRuleEngineForTUN() error {
	// 从全局配置获取 routing rules
	globalCfg := config.GetGlobalConfig()
	if globalCfg == nil || globalCfg.RoutingRules == nil {
		logger.Info("TUN: Routing rules not configured, using default DNS cache")

		// 使用默认配置初始化 cache
		// 如果没有配置文件，仍然为 TUN 模式创建 DNS cache
		defaultRules := &config.RoutingRulesConfig{
			IPDomainCache: &config.CacheConfig{
				Enabled:    true,
				MaxEntries: 10000,
				TTL:        "5m",
			},
		}

		ruleEngine := rules.GetEngine()
		err := ruleEngine.Initialize(defaultRules)
		if err != nil {
			return fmt.Errorf("failed to initialize rule engine with default config: %w", err)
		}

		logger.Info("✓ Rule engine initialized with default DNS cache for TUN mode")
		return nil
	}

	// 使用配置文件中的 routing rules
	ruleEngine := rules.GetEngine()
	err := ruleEngine.Initialize(globalCfg.RoutingRules)
	if err != nil {
		return fmt.Errorf("failed to initialize rule engine: %w", err)
	}

	logger.Info("✓ Rule engine initialized with custom routing rules for TUN mode")
	return nil
}
