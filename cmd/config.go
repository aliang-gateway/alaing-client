package cmd

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"time"

	"nursor.org/nursorgate/common/logger"
	"nursor.org/nursorgate/outbound"
	proxyRegistry "nursor.org/nursorgate/outbound"
	"nursor.org/nursorgate/processor/config"
	geoip "nursor.org/nursorgate/processor/geoip"
	rules "nursor.org/nursorgate/processor/rules"
)

// Re-export config types for backward compatibility
type Config = config.Config
type EngineConfig = config.EngineConfig

// LoadConfig 从文件加载配置
func LoadConfig(configPath string) (*Config, error) {
	data, err := ioutil.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// 自动迁移旧配置到新格式
	if err := migrateOldConfig(&config); err != nil {
		return nil, fmt.Errorf("failed to migrate config: %w", err)
	}

	return &config, nil
}

// migrateOldConfig 迁移旧配置格式到新格式
func migrateOldConfig(cfg *Config) error {
	// 1. coreServer 从顶层迁移到 nonelane 配置中
	if cfg.CoreServer != "" {
		if cfg.Proxies == nil {
			cfg.Proxies = make(map[string]*config.ProxyConfig)
		}

		// 如果 nonelane 不存在，创建并迁移 coreServer
		if _, exists := cfg.Proxies["nonelane"]; !exists {
			cfg.Proxies["nonelane"] = &config.ProxyConfig{
				Type:       "nonelane",
				CoreServer: cfg.CoreServer,
			}
			logger.Info("Migrated coreServer to nonelane proxy config")
		} else if cfg.Proxies["nonelane"].CoreServer == "" {
			// nonelane 存在但没有 CoreServer，设置它
			cfg.Proxies["nonelane"].CoreServer = cfg.CoreServer
			logger.Info("Set CoreServer for existing nonelane proxy")
		}
	}

	// 2. 确保 direct 代理存在
	if cfg.Proxies == nil {
		cfg.Proxies = make(map[string]*config.ProxyConfig)
	}
	if _, exists := cfg.Proxies["direct"]; !exists {
		cfg.Proxies["direct"] = &config.ProxyConfig{Type: "direct"}
		logger.Debug("Added default direct proxy to config")
	}

	// 3. 旧 door 格式转换（单个代理 -> members 数组）
	if doorCfg, exists := cfg.Proxies["door"]; exists {
		// 如果 door 的 type 不是 "door" 且没有 members，说明是旧格式
		if doorCfg.Type != "door" && len(doorCfg.Members) == 0 {
			// 转换为 members 格式
			member := config.DoorProxyMember{
				ShowName:    "Default Node",
				Type:        doorCfg.Type,
				Latency:     999, // 默认延迟
				VLESS:       doorCfg.VLESS,
				Shadowsocks: doorCfg.Shadowsocks,
			}
			doorCfg.Members = []config.DoorProxyMember{member}
			doorCfg.Type = "door"
			// 清除单个代理配置字段
			doorCfg.VLESS = nil
			doorCfg.Shadowsocks = nil
			logger.Info("Migrated old door format to new door collection format")
		}
	}

	return nil
}

// ApplyConfig applies the configuration to the system in clear phases.
//
// Phases:
// 1. Apply engine configuration (network stack, TUN device, etc.)
// 2. Register built-in proxies (direct + nonelane) - always available
// 3. Register door proxy collection if configured
// 4. Register other custom user-defined proxies
// 5. Set the active default proxy for routing
func ApplyConfig(config *Config) error {
	// Phase 1: Apply engine configuration
	if config.Engine != nil {
		if err := applyEngineConfig(config.Engine); err != nil {
			return fmt.Errorf("phase 1 - engine config failed: %w", err)
		}
		logger.Debug("Phase 1: Engine configuration applied")
	}

	// Phase 2: Register built-in proxies (direct + nonelane)
	// These are mandatory and always available
	if err := registerBuiltinProxies(config); err != nil {
		return fmt.Errorf("phase 2 - builtin proxies registration failed: %w", err)
	}
	logger.Debug("Phase 2: Built-in proxies registered")

	// Phase 3: Register door proxy collection if configured
	if err := registerDoorProxy(config.Proxies); err != nil {
		return fmt.Errorf("phase 3 - door proxy registration failed: %w", err)
	}
	logger.Debug("Phase 3: Door proxy collection registered")

	// Phase 4: Register custom user proxies from configuration
	// These are optional and supplement the built-in proxies
	if err := registerCustomProxies(config.Proxies); err != nil {
		return fmt.Errorf("phase 4 - custom proxy registration failed: %w", err)
	}
	logger.Debug("Phase 4: Custom proxies registered")

	// Phase 5: Set the active default proxy for routing decisions
	// Determines which proxy is used when no specific routing rule applies
	if err := setEffectiveDefaultProxy(config.CurrentProxy); err != nil {
		return fmt.Errorf("phase 5 - failed to set default proxy: %w", err)
	}
	logger.Debug("Phase 5: Default proxy set for routing")

	// Phase 6: Initialize GeoIP service if configured
	if err := initializeGeoIP(config.RoutingRules); err != nil {
		logger.Warn(fmt.Sprintf("Phase 6 - GeoIP initialization failed (non-fatal): %v", err))
	} else {
		logger.Debug("Phase 6: GeoIP service initialized")
	}

	// Phase 7: Initialize routing rule engine
	if err := initializeRuleEngine(config.RoutingRules); err != nil {
		logger.Warn(fmt.Sprintf("Phase 7 - Rule engine initialization failed (non-fatal): %v", err))
	} else {
		logger.Debug("Phase 7: Rule engine initialized")
	}

	logger.Info("Configuration applied successfully")
	return nil
}

// setEffectiveDefaultProxy sets the active default proxy.
// Supports "door:showname" format to select specific door member.
// If currentProxy is empty or unavailable, falls back to "direct".
func setEffectiveDefaultProxy(currentProxy string) error {
	registry := outbound.GetRegistry()

	// Determine which proxy to set as default
	proxyName := currentProxy
	if proxyName == "" {
		proxyName = "direct"
		logger.Debug("No default proxy specified, using 'direct'")
	}

	// Check if it's a door member specification (format: "door:memberName")
	if strings.HasPrefix(proxyName, "door:") {
		memberName := strings.TrimPrefix(proxyName, "door:")
		if err := registry.SetDoorMember(memberName); err != nil {
			logger.Warn(fmt.Sprintf("Failed to set door member '%s': %v, using auto-select", memberName, err))
			registry.EnableDoorAutoSelect()
		} else {
			logger.Info(fmt.Sprintf("Door member set to: %s", memberName))
		}
		// Don't set door as default in registry, door is accessed via GetDoor()
		return nil
	}

	// For "door" without member specification, enable auto-select
	if proxyName == "door" {
		registry.EnableDoorAutoSelect()
		logger.Info("Door auto-select enabled")
		// Don't set door as default in registry
		return nil
	}

	// Attempt to set the requested proxy
	if err := registry.SetDefault(proxyName); err != nil {
		logger.Warn(fmt.Sprintf("Failed to set proxy '%s' as default: %v, attempting fallback to 'direct'", proxyName, err))

		// Fallback to direct proxy
		if proxyName != "direct" {
			if err := registry.SetDefault("direct"); err != nil {
				return fmt.Errorf("failed to fallback to direct proxy: %w", err)
			}
			logger.Info("Fallback: Direct proxy set as default")
			return nil
		}
		return err
	}

	logger.Info(fmt.Sprintf("Default proxy set to: %s", proxyName))
	return nil
}

// applyEngineConfig 应用引擎配置
func applyEngineConfig(engineCfg *config.EngineConfig) error {
	// 解析 UDP 超时时间
	udpTimeout, err := time.ParseDuration(engineCfg.UDPTimeout)
	if err != nil {
		// 如果解析失败，使用默认值
		udpTimeout = 60 * time.Second
		logger.Warn(fmt.Sprintf("Failed to parse udp-timeout '%s', using default 60s", engineCfg.UDPTimeout))
	}

	// 转换为 processor/config.EngineConf
	engineConf := &config.EngineConf{
		MTU:                      engineCfg.MTU,
		Mark:                     engineCfg.Mark,
		RestAPI:                  engineCfg.RestAPI,
		Device:                   engineCfg.Device,
		LogLevel:                 engineCfg.LogLevel,
		Interface:                engineCfg.Interface,
		TCPModerateReceiveBuffer: engineCfg.TCPModerateReceiveBuffer,
		TCPSendBufferSize:        engineCfg.TCPSendBufferSize,
		TCPReceiveBufferSize:     engineCfg.TCPReceiveBufferSize,
		MulticastGroups:          engineCfg.MulticastGroups,
		TUNPreUp:                 engineCfg.TUNPreUp,
		TUNPostUp:                engineCfg.TUNPostUp,
		UDPTimeout:               udpTimeout,
	}

	// 插入到配置系统
	config.Insert(engineConf)
	logger.Info("Engine config applied successfully")
	return nil
}

// registerBuiltinProxies 注册内置代理（direct 和 nonelane）
func registerBuiltinProxies(cfg *Config) error {
	registry := proxyRegistry.GetRegistry()

	// 1. 注册 direct 代理
	if err := registry.RegisterDefault(); err != nil {
		return fmt.Errorf("failed to register direct proxy: %w", err)
	}

	// 2. 注册 nonelane 代理
	coreServer := ""
	if cfg.Proxies != nil {
		if nonelaneConfig, exists := cfg.Proxies["nonelane"]; exists && nonelaneConfig != nil {
			coreServer = nonelaneConfig.CoreServer
		}
	}
	// 如果配置中没有指定，使用顶层的 CoreServer（向后兼容）
	if coreServer == "" && cfg.CoreServer != "" {
		coreServer = cfg.CoreServer
	}

	if err := registry.RegisterNonelane(coreServer); err != nil {
		return fmt.Errorf("failed to register nonelane proxy: %w", err)
	}

	// 设置全局 ServerHost（用于其他功能）
	if coreServer != "" {
		config.SetCursorAiGatewayHost(coreServer)
	}

	return nil
}

// registerDoorProxy 注册 door 代理集合
func registerDoorProxy(proxies map[string]*config.ProxyConfig) error {
	if proxies == nil {
		return nil
	}

	doorConfig, exists := proxies["door"]
	if !exists || doorConfig == nil {
		logger.Debug("No door proxy configured")
		return nil
	}

	// 验证 door 配置
	if err := doorConfig.Validate(); err != nil {
		return fmt.Errorf("invalid door proxy config: %w", err)
	}

	// 注册 door 代理集合
	registry := proxyRegistry.GetRegistry()
	if err := registry.RegisterDoorFromConfig(doorConfig); err != nil {
		return fmt.Errorf("failed to register door proxy: %w", err)
	}

	logger.Info("Door proxy collection registered successfully")
	return nil
}

// registerCustomProxies 注册自定义代理（除 direct, nonelane, door 外的其他代理）
func registerCustomProxies(proxies map[string]*config.ProxyConfig) error {
	if len(proxies) == 0 {
		logger.Debug("No custom proxies to register")
		return nil
	}

	registry := proxyRegistry.GetRegistry()

	for name, cfg := range proxies {
		// 跳过内置代理和 door 代理
		if name == "direct" || name == "nonelane" || name == "door" {
			continue
		}

		if cfg == nil {
			logger.Warn(fmt.Sprintf("Nil proxy config for '%s', skipping", name))
			continue
		}

		// 验证配置
		if err := cfg.Validate(); err != nil {
			logger.Error(fmt.Sprintf("Invalid config for proxy '%s': %v", name, err))
			continue
		}

		// 注册代理（创建实例 + 存储配置）
		if err := registry.RegisterFromConfig(name, cfg); err != nil {
			logger.Error(fmt.Sprintf("Failed to register proxy '%s': %v", name, err))
			continue
		}

		logger.Info(fmt.Sprintf("Custom proxy '%s' registered successfully", name))
	}

	return nil
}

// LoadAndApplyConfig 加载并应用配置文件
func LoadAndApplyConfig(configPath string) error {
	// 检查文件是否存在
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		logger.Warn(fmt.Sprintf("Config file not found: %s, using defaults", configPath))
		return nil
	}

	// 加载配置
	config, err := LoadConfig(configPath)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// 应用配置
	if err := ApplyConfig(config); err != nil {
		return fmt.Errorf("failed to apply config: %w", err)
	}

	logger.Info(fmt.Sprintf("Config loaded and applied successfully from: %s", configPath))
	return nil
}

// FetchConfigFromRemote 从远程服务器获取配置
func FetchConfigFromRemote(token string, serverURL string) (*Config, error) {
	if serverURL == "" {
		// 使用默认服务器地址
		serverURL = config.GetCursorAiGatewayHost()
		if serverURL == "" {
			serverURL = "https://api2.nursor.org:12235" // 默认服务器
		}
	}

	// 确保 URL 格式正确
	if !strings.HasPrefix(serverURL, "http://") && !strings.HasPrefix(serverURL, "https://") {
		serverURL = "https://" + serverURL
	}

	// 构建请求 URL
	url := fmt.Sprintf("%s/api/config?token=%s", serverURL, token)

	// 创建 HTTP 请求
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// 设置请求头
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	req.Header.Set("Content-Type", "application/json")

	// 发送请求
	client := &http.Client{
		Timeout: 30 * time.Second,
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch config from remote: %w", err)
	}
	defer resp.Body.Close()

	// 检查响应状态
	if resp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
		return nil, fmt.Errorf("remote server returned status %d: %s", resp.StatusCode, string(body))
	}

	// 读取响应体
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// 解析 JSON
	var config Config
	if err := json.Unmarshal(body, &config); err != nil {
		return nil, fmt.Errorf("failed to parse remote config: %w", err)
	}

	return &config, nil
}

// FetchAndApplyConfigFromRemote 从远程获取配置并应用
func FetchAndApplyConfigFromRemote(token string, serverURL string) error {
	// 从远程获取配置
	config, err := FetchConfigFromRemote(token, serverURL)
	if err != nil {
		return fmt.Errorf("failed to fetch config from remote: %w", err)
	}

	// 应用配置
	if err := ApplyConfig(config); err != nil {
		return fmt.Errorf("failed to apply config: %w", err)
	}

	logger.Info("Config fetched and applied successfully from remote server")
	return nil
}

// LoadConfigFromBytes 从字节数组加载配置（用于从远程获取的配置）
func LoadConfigFromBytes(data []byte) (*Config, error) {
	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}
	return &config, nil
}

// SaveConfigToFile 保存配置到文件
func SaveConfigToFile(config *Config, filePath string) error {
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := ioutil.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// initializeGeoIP 初始化 GeoIP 服务
func initializeGeoIP(routingRules *config.RoutingRulesConfig) error {
	if routingRules == nil || routingRules.GeoIP == nil {
		logger.Info("GeoIP routing not configured, service disabled")
		return nil
	}

	geoipCfg := routingRules.GeoIP
	if !geoipCfg.Enabled {
		logger.Info("GeoIP service disabled in config")
		return nil
	}

	if geoipCfg.DatabasePath == "" {
		return fmt.Errorf("GeoIP enabled but database path not specified")
	}

	// 加载 GeoIP 数据库
	service := geoip.GetService()
	if err := service.LoadDatabase(geoipCfg.DatabasePath); err != nil {
		return fmt.Errorf("failed to load GeoIP database: %w", err)
	}

	logger.Info(fmt.Sprintf("GeoIP service initialized successfully (database: %s, chinaDirect: %v)",
		geoipCfg.DatabasePath, geoipCfg.ChinaDirect))

	return nil
}

// initializeRuleEngine 初始化路由规则引擎
func initializeRuleEngine(routingRules *config.RoutingRulesConfig) error {
	if routingRules == nil {
		logger.Info("Routing rules not configured, rule engine disabled")
		return nil
	}

	// 获取规则引擎实例
	engine := rules.GetEngine()

	// 初始化规则引擎
	if err := engine.Initialize(routingRules); err != nil {
		return fmt.Errorf("failed to initialize rule engine: %w", err)
	}

	logger.Info("Routing rule engine initialized successfully")

	// 打印配置摘要
	if routingRules.GeoIP != nil && routingRules.GeoIP.Enabled {
		logger.Info(fmt.Sprintf("  - GeoIP routing: enabled (chinaDirect=%v)", routingRules.GeoIP.ChinaDirect))
	}
	if routingRules.BypassRules != nil && routingRules.BypassRules.Enabled {
		logger.Info(fmt.Sprintf("  - Bypass rules: enabled (%d domains, %d suffixes, %d IP ranges)",
			len(routingRules.BypassRules.Domains),
			len(routingRules.BypassRules.DomainSuffixes),
			len(routingRules.BypassRules.IPRanges)))
	}
	if routingRules.IPDomainCache != nil && routingRules.IPDomainCache.Enabled {
		logger.Info(fmt.Sprintf("  - IP-Domain cache: enabled (max=%d, TTL=%s)",
			routingRules.IPDomainCache.MaxEntries,
			routingRules.IPDomainCache.TTL))
	}

	return nil
}
