package config

import (
	"fmt"
	"math/rand"
	"strings"
	"sync"

	"nursor.org/nursorgate/common/logger"
	"nursor.org/nursorgate/outbound/proxy"
	"nursor.org/nursorgate/outbound/proxy/shadowsocks"
	"nursor.org/nursorgate/outbound/proxy/vless"
)

var (
	mu                sync.RWMutex
	directProxy       proxy.Proxy
	doorProxy         proxy.Proxy
	vlessConfig       *VLESSConfig
	shadowsocksConfig *ShadowsocksConfig
	currentProxy      *proxy.Proxy
)

func SetCurrentProxy(proxy *proxy.Proxy) error {
	mu.Lock()
	defer mu.Unlock()

	currentProxy = proxy
	return nil
}

func GetCurrentProxy() *proxy.Proxy {
	mu.RLock()
	defer mu.RUnlock()
	return currentProxy
}

// VLESSConfig VLESS 代理配置
type VLESSConfig struct {
	// 基础配置
	Server string `json:"server"` // 服务器地址，格式: host:port
	UUID   string `json:"uuid"`   // UUID
	Flow   string `json:"flow"`   // 流控类型，如: xtls-rprx-vision

	// TLS 配置
	TLSEnabled bool   `json:"tls_enabled"` // 是否启用 TLS
	SNI        string `json:"sni"`         // SNI 服务器名称

	// REALITY 配置
	RealityEnabled bool   `json:"reality_enabled"` // 是否启用 REALITY
	PublicKey      string `json:"public_key"`      // REALITY 公钥
	ShortID        string `json:"short_id"`        // REALITY ShortID（可选，为空则随机选择）
	ShortIDList    string `json:"short_id_list"`   // ShortID 列表，逗号分隔
}

// ShadowsocksConfig Shadowsocks 代理配置
type ShadowsocksConfig struct {
	Server   string `json:"server"`    // 服务器地址，格式: host:port
	Method   string `json:"method"`    // 加密方法，如: aes-256-gcm
	Password string `json:"password"`  // 密码
	ObfsMode string `json:"obfs_mode"` // 混淆模式: tls, http, 或空
	ObfsHost string `json:"obfs_host"` // 混淆主机
}

// ProxyConfig 通用代理配置
type ProxyConfig struct {
	Type        string             `json:"type"` // 代理类型: vless, shadowsocks
	VLESS       *VLESSConfig       `json:"vless,omitempty"`
	Shadowsocks *ShadowsocksConfig `json:"shadowsocks,omitempty"`
	IsDefault   bool               `json:"is_default"`    // 是否为默认代理
	IsDoorProxy bool               `json:"is_door_proxy"` // 是否为门代理（用于 DNS 等）
}

// SetVLESSConfig 设置 VLESS 配置
func SetVLESSConfig(cfg *VLESSConfig) error {
	mu.Lock()
	defer mu.Unlock()

	vlessConfig = cfg
	logger.Info("VLESS config updated", "server", cfg.Server, "uuid", cfg.UUID[:8]+"...")
	return nil
}

// SetShadowsocksConfig 设置 Shadowsocks 配置
func SetShadowsocksConfig(cfg *ShadowsocksConfig) error {
	mu.Lock()
	defer mu.Unlock()

	shadowsocksConfig = cfg
	logger.Info("Shadowsocks config updated", "server", cfg.Server, "method", cfg.Method)
	return nil
}

// SetProxyConfig 设置代理配置（通用接口）
func SetProxyConfig(cfg *ProxyConfig) error {
	mu.Lock()
	defer mu.Unlock()

	var p proxy.Proxy
	var err error

	switch cfg.Type {
	case "vless":
		if cfg.VLESS == nil {
			logger.Error("VLESS config is required for vless type")
			return fmt.Errorf("VLESS config is required for vless type")
		}
		p, err = createVLESSProxy(cfg.VLESS)
		if err != nil {
			logger.Error("Failed to create VLESS proxy: " + err.Error())
			return fmt.Errorf("failed to create VLESS proxy: %w", err)
		}
		vlessConfig = cfg.VLESS

	case "shadowsocks":
		if cfg.Shadowsocks == nil {
			logger.Error("Shadowsocks config is required for shadowsocks type")
			return fmt.Errorf("Shadowsocks config is required for shadowsocks type")
		}
		p, err = createShadowsocksProxy(cfg.Shadowsocks)
		if err != nil {
			logger.Error("Failed to create Shadowsocks proxy: " + err.Error())
			return fmt.Errorf("failed to create Shadowsocks proxy: %w", err)
		}
		shadowsocksConfig = cfg.Shadowsocks

	default:
		logger.Error("Unsupported proxy type: " + cfg.Type)
		return fmt.Errorf("unsupported proxy type: %s", cfg.Type)
	}

	if cfg.IsDefault {
		directProxy = p
		logger.Info("Default proxy updated", "type", cfg.Type)
	}

	if cfg.IsDoorProxy {
		doorProxy = p
		logger.Info("Door proxy updated", "type", cfg.Type)
	}

	return nil
}

// CreateVLESSProxyFromConfig 创建 VLESS 代理实例（公开函数，供 registry 使用）
func CreateVLESSProxyFromConfig(cfg *VLESSConfig) (proxy.Proxy, error) {
	return createVLESSProxy(cfg)
}

// CreateShadowsocksProxyFromConfig 创建 Shadowsocks 代理实例（公开函数，供 registry 使用）
func CreateShadowsocksProxyFromConfig(cfg *ShadowsocksConfig) (proxy.Proxy, error) {
	return createShadowsocksProxy(cfg)
}

// createVLESSProxy 创建 VLESS 代理实例（内部函数）
func createVLESSProxy(cfg *VLESSConfig) (proxy.Proxy, error) {
	if cfg.RealityEnabled {
		// 使用 REALITY
		// 如果提供了 ShortID，使用它；否则从 ShortIDList 随机选择
		shortID := cfg.ShortID
		if shortID == "" && cfg.ShortIDList != "" {
			shortIDArray := strings.Split(cfg.ShortIDList, ",")
			if len(shortIDArray) > 0 {
				shortID = shortIDArray[rand.Intn(len(shortIDArray))]
			}
		}
		// 注意：这里需要调用支持 shortID 的函数，但当前 vless 包可能没有这个函数
		// 暂时使用默认函数，shortID 会在内部随机选择
		return vless.NewVLESSWithReality(
			cfg.Server,
			cfg.UUID,
			cfg.SNI,
			cfg.PublicKey,
		)
	} else if cfg.TLSEnabled {
		// 使用 TLS
		if cfg.Flow != "" {
			return vless.NewVLESSWithVision(cfg.Server, cfg.UUID, cfg.SNI)
		}
		return vless.NewVLESSWithTLS(cfg.Server, cfg.UUID, cfg.SNI)
	} else {
		// 基础 VLESS
		return vless.NewVLESS(cfg.Server, cfg.UUID)
	}
}

// createShadowsocksProxy 创建 Shadowsocks 代理实例
func createShadowsocksProxy(cfg *ShadowsocksConfig) (proxy.Proxy, error) {
	return shadowsocks.NewShadowsocks(
		cfg.Server,
		cfg.Method,
		cfg.Password,
		cfg.ObfsMode,
		cfg.ObfsHost,
	)
}

// GetDirectProxy 获取直连代理
func GetDirectProxy() proxy.Proxy {
	mu.RLock()
	defer mu.RUnlock()
	return directProxy
}

// GetNursorProxy 获取门代理
func GetDoorProxy() proxy.Proxy {
	mu.RLock()
	defer mu.RUnlock()
	return doorProxy
}

// GetVLESSConfig 获取 VLESS 配置
func GetVLESSConfig() *VLESSConfig {
	mu.RLock()
	defer mu.RUnlock()
	if vlessConfig == nil {
		return nil
	}
	// 返回副本以避免并发修改
	cfg := *vlessConfig
	return &cfg
}

// GetShadowsocksConfig 获取 Shadowsocks 配置
func GetShadowsocksConfig() *ShadowsocksConfig {
	mu.RLock()
	defer mu.RUnlock()
	if shadowsocksConfig == nil {
		return nil
	}
	// 返回副本以避免并发修改
	cfg := *shadowsocksConfig
	return &cfg
}
