package cmd

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"strings"

	"nursor.org/nursorgate/common/logger"
	"nursor.org/nursorgate/outbound"
	"nursor.org/nursorgate/processor/config"
	geoip "nursor.org/nursorgate/processor/geoip"
	"nursor.org/nursorgate/processor/proxyserver"
	rules "nursor.org/nursorgate/processor/rules"
)

// Embed the default configuration
//
//go:embed config.default.json
var defaultConfigData string

// Re-export config types for backward compatibility
type Config = config.Config

// setUseDefaultConfig marks that the default configuration is being used
func setUseDefaultConfig(value bool) {
	config.SetUsingDefaultConfig(value)
}

// IsUsingDefaultConfig returns whether the default embedded configuration is being used
func IsUsingDefaultConfig() bool {
	return config.IsUsingDefaultConfig()
}

// GetDefaultConfigBytes 返回嵌入的默认配置字节数据
func GetDefaultConfigBytes() []byte {
	return []byte(defaultConfigData)
}

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

	return &config, nil
}

// LoadConfigFromBytes 从字节数据加载配置
func LoadConfigFromBytes(data []byte) (*Config, error) {
	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	return &config, nil
}

// ApplyConfig applies the configuration to the system in clear phases.
//
// Phases:
// 1. Apply engine configuration (network stack, TUN device, etc.)
// 2. Register built-in proxies (direct + nonelane) - always available
// 3. Register door proxy collection if configured
// 4. Register other custom user-defined proxies
// 5. Set the active default proxy for routing
func ApplyConfig(cfg *Config) error {
	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("configuration validation failed: %w", err)
	}

	// Store config globally for access by other modules
	config.SetGlobalConfig(cfg)

	// Phase 1: Register built-in proxies (direct + nonelane)
	// These are mandatory and always available
	if err := registerBuiltinProxies(cfg); err != nil {
		return fmt.Errorf("phase 1 - builtin proxies registration failed: %w", err)
	}
	logger.Debug("Phase 1: Built-in proxies registered")

	// Phase 2: Register door proxy collection if configured
	if err := registerDoorProxy(cfg); err != nil {
		return fmt.Errorf("phase 2 - door proxy registration failed: %w", err)
	}
	logger.Debug("Phase 2: Door proxy collection registered")

	// Phase 3: Initialize global DNS resolver
	// Must be done after door and direct proxies are registered
	registry := outbound.GetRegistry()
	doorProxy, _ := registry.GetDoor()
	directProxy, _ := registry.Get("direct")
	if err := proxyserver.InitGlobalResolver(doorProxy, directProxy, cfg); err != nil {
		logger.Warn(fmt.Sprintf("Phase 3 - Failed to initialize DNS resolver: %v", err))
	} else {
		logger.Debug("Phase 3: Global DNS resolver initialized")
	}

	// Phase 4: Set the active default proxy for routing decisions
	// Determines which proxy is used when no specific routing rule applies
	if err := setEffectiveDefaultProxy(cfg.CurrentProxy); err != nil {
		return fmt.Errorf("phase 4 - failed to set default proxy: %w", err)
	}
	logger.Debug("Phase 4: Default proxy set for routing")

	// Phase 5: Initialize GeoIP service if configured
	if err := initializeGeoIP(cfg.RoutingRules); err != nil {
		logger.Warn(fmt.Sprintf("Phase 5 - GeoIP initialization failed (non-fatal): %v", err))
	} else {
		logger.Debug("Phase 5: GeoIP service initialized")
	}

	// Phase 6: Initialize routing rule engine
	if err := initializeRuleEngine(cfg.RoutingRules); err != nil {
		logger.Warn(fmt.Sprintf("Phase 6 - Rule engine initialization failed (non-fatal): %v", err))
	} else {
		logger.Debug("Phase 6: Rule engine initialized")
	}

	logger.Info("Configuration applied successfully")
	return nil
}

// setEffectiveDefaultProxy sets the active default proxy.
// Supports "door:showname" format to select specific door member.
// If currentProxy is empty or unavailable, falls back to first door member or "direct".
func setEffectiveDefaultProxy(currentProxy string) error {
	registry := outbound.GetRegistry()

	// Check if it's a door member specification (format: "door:memberName")
	if strings.HasPrefix(currentProxy, "door:") {
		memberName := strings.TrimPrefix(currentProxy, "door:")
		if err := registry.SetDoorMember(memberName); err != nil {
			logger.Warn(fmt.Sprintf("Failed to set door member '%s': %v, using auto-select", memberName, err))
			registry.EnableDoorAutoSelect()
		} else {
			logger.Info(fmt.Sprintf("Door member set to: %s", memberName))
		}
		// Don't set door as default in registry, door is accessed via GetDoor()
		return nil
	}

	// Handle specific proxy names
	switch currentProxy {
	case "":
		// When empty, select door member with lowest latency or random if no latency data
		if bestMember := getBestOrRandomDoorMember(registry); bestMember != "" {
			proxyName := "door:" + bestMember
			logger.Debug(fmt.Sprintf("No default proxy specified, using best/random door member: %s", bestMember))
			return setEffectiveDefaultProxy(proxyName) // Recursively handle door:member
		}
		currentProxy = "direct"
		logger.Debug("No default proxy specified and no door members found, using 'direct'")

	case "auto":
		// When "auto", enable door auto-select
		registry.EnableDoorAutoSelect()
		logger.Debug("CurrentProxy is 'auto', enabling door auto-select")
		logger.Info("Door auto-select enabled")
		return nil

	case "door":
		// For "door" without member specification, enable auto-select
		registry.EnableDoorAutoSelect()
		logger.Info("Door auto-select enabled")
		// Don't set door as default in registry
		return nil
	}

	// Log the proxy selection
	if currentProxy != "direct" {
		logger.Info(fmt.Sprintf("Proxy selection: '%s' (routing will always use 'direct')", currentProxy))
	} else {
		logger.Info("Using direct proxy for routing")
	}
	return nil
}

// getBestOrRandomDoorMember returns the door member with lowest latency,
// or a random member if all latencies are 0 or unavailable
func getBestOrRandomDoorMember(registry *outbound.Registry) string {
	// Get the door group
	doorGroup := registry.GetDoorGroup()
	if doorGroup == nil {
		return ""
	}

	// Get all members
	members := doorGroup.ListMembers()
	if len(members) == 0 {
		return ""
	}

	// Check if any member has latency > 0
	var bestMember *outbound.DoorProxyMemberInfo
	hasLatencyData := false

	for i := range members {
		member := &members[i]
		if member.Latency > 0 {
			if !hasLatencyData || member.Latency < bestMember.Latency {
				bestMember = member
				hasLatencyData = true
			}
		}
	}

	// If we have latency data, return the best member
	if hasLatencyData && bestMember != nil {
		return bestMember.ShowName
	}

	// If no latency data, return a random member
	randomIndex := rand.Intn(len(members))
	return members[randomIndex].ShowName
}

// registerBuiltinProxies 注册内置代理（direct 和 nonelane）
func registerBuiltinProxies(cfg *Config) error {
	registry := outbound.GetRegistry()

	// 1. 注册 direct 代理
	if err := registry.RegisterDefault(); err != nil {
		return fmt.Errorf("failed to register direct proxy: %w", err)
	}

	// 2. 注册 nonelane 代理
	coreServer := ""
	// 首先尝试从 NonelaneCoreServer 字段读取
	if cfg.BaseProxies != nil {
		// 其次尝试从 BaseProxies["nonelane"].CoreServer 读取
		if nonelaneConfig, exists := cfg.BaseProxies["nonelane"]; exists && nonelaneConfig != nil {
			coreServer = nonelaneConfig.CoreServer
		}
	}

	// 如果配置中没有指定，使用默认值
	if coreServer == "" {
		coreServer = "ai-gateway.nursor.org:443"
		logger.Debug("Using default Nonelane server address")
	}

	if err := registry.RegisterNonelane(coreServer); err != nil {
		return fmt.Errorf("failed to register nonelane proxy: %w", err)
	} else {
		config.SetCursorAiGatewayHost(coreServer)
	}

	return nil
}

// registerDoorProxy 注册 door 代理集合
func registerDoorProxy(cfg *config.Config) error {
	doorConfig := cfg.DoorProxy
	if doorConfig == nil {
		logger.Debug("No door proxy configured")
		return nil
	}

	// 验证 door 配置
	if doorConfig.Type != "door" {
		return fmt.Errorf("invalid door proxy config: type must be 'door', got '%s'", doorConfig.Type)
	}

	if len(doorConfig.Members) == 0 {
		return fmt.Errorf("door proxy must have at least one member")
	}

	// 注册 door 代理集合
	registry := outbound.GetRegistry()
	if err := registry.RegisterDoorFromConfig(doorConfig); err != nil {
		return fmt.Errorf("failed to register door proxy: %w", err)
	}

	logger.Info(fmt.Sprintf("Door proxy collection registered successfully with %d members", len(doorConfig.Members)))
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
	cfg, err := LoadConfig(configPath)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// 应用配置
	if err := ApplyConfig(cfg); err != nil {
		return fmt.Errorf("failed to apply config: %w", err)
	}

	logger.Info(fmt.Sprintf("Config loaded and applied successfully from: %s", configPath))
	return nil
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
