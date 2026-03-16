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
	"nursor.org/nursorgate/processor/proxyserver"
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

	// Phase 2: Register optional SOCKS proxy (if configured)
	if err := registerSocksProxy(cfg); err != nil {
		return fmt.Errorf("phase 2 - socks proxy registration failed: %w", err)
	}
	logger.Debug("Phase 2: Optional SOCKS proxy registered")

	// Phase 3: Register door proxy collection if configured (legacy/optional)
	if err := registerDoorProxy(cfg); err != nil {
		return fmt.Errorf("phase 3 - door proxy registration failed: %w", err)
	}
	logger.Debug("Phase 3: Door proxy collection registered")

	// Phase 4: Initialize global DNS resolver
	// Must be done after proxies are registered
	registry := outbound.GetRegistry()
	primaryProxy, _ := registry.Get("socks")
	directProxy, _ := registry.Get("direct")
	if err := proxyserver.InitGlobalResolver(primaryProxy, directProxy, cfg); err != nil {
		logger.Warn(fmt.Sprintf("Phase 4 - Failed to initialize DNS resolver: %v", err))
	} else {
		logger.Debug("Phase 4: Global DNS resolver initialized")
	}

	// Phase 5: Set the active default proxy for routing decisions
	// Determines which proxy is used when no specific routing rule applies
	if err := setEffectiveDefaultProxy(cfg.CurrentProxy); err != nil {
		return fmt.Errorf("phase 5 - failed to set default proxy: %w", err)
	}
	logger.Debug("Phase 5: Default proxy set for routing")

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

// registerSocksProxy registers optional SOCKS5 proxy
func registerSocksProxy(cfg *config.Config) error {
	if cfg.SocksProxy == nil {
		logger.Debug("No socks proxy configured")
		return nil
	}

	// Validate config (already validated in cfg.Validate, but keep safe)
	if err := cfg.SocksProxy.Validate(); err != nil {
		return fmt.Errorf("invalid socks proxy config: %w", err)
	}

	addr := fmt.Sprintf("%s:%d", cfg.SocksProxy.Server, cfg.SocksProxy.ServerPort)
	proxyInstance := outbound.GetRegistry()

	socksProxy, err := outbound.CreateSocksProxy(addr, cfg.SocksProxy.Username, cfg.SocksProxy.Password)
	if err != nil {
		return fmt.Errorf("failed to create socks proxy: %w", err)
	}

	if err := proxyInstance.Register("socks", socksProxy); err != nil {
		return fmt.Errorf("failed to register socks proxy: %w", err)
	}

	logger.Info(fmt.Sprintf("SOCKS proxy registered at %s", addr))
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
// 注意：如果用户显式指定了配置文件路径但文件不存在，应返回错误
func LoadAndApplyConfig(configPath string) error {
	// 检查文件是否存在
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return fmt.Errorf("config file not found: %s", configPath)
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

	// 标记使用的是自定义配置（非默认配置）
	setUseDefaultConfig(false)

	logger.Info(fmt.Sprintf("Config loaded and applied successfully from: %s", configPath))
	return nil
}
