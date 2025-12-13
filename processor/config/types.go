package config

import (
	"fmt"
	"time"
)

// BaseProxyConfig represents a proxy configuration
type BaseProxyConfig struct {
	Type string `json:"type"`
	// Nonelane 代理专用
	CoreServer string `json:"core_server,omitempty"`
}

// DoorProxyMember represents a member in a door proxy collection
type DoorProxyMember struct {
	ShowName    string             `json:"showname"` // 显示名称
	Type        string             `json:"type"`     // vless/shadowsocks
	Latency     int64              `json:"latency"`  // 延迟（毫秒）
	VLESS       *VLESSConfig       `json:"vless,omitempty"`
	Shadowsocks *ShadowsocksConfig `json:"shadowsocks,omitempty"`
}

// VLESSConfig represents VLESS protocol configuration
type VLESSConfig struct {
	Server         string   `json:"server_host"`
	ServerPort     uint16   `json:"server_port"`
	UUID           string   `json:"uuid"`
	Flow           string   `json:"flow,omitempty"`
	TLSEnabled     bool     `json:"tls_enabled"`
	RealityEnabled bool     `json:"reality_enabled"`
	SNI            string   `json:"sni,omitempty"`
	PublicKey      string   `json:"public_key,omitempty"`
	ShortIDs       []string `json:"short_ids,omitempty"`
}

// ShadowsocksConfig represents Shadowsocks protocol configuration
type ShadowsocksConfig struct {
	Server     string `json:"server_host"`
	ServerPort uint16 `json:"server_port"`
	Method     string `json:"method"`
	Password   string `json:"password"`
	Username   string `json:"username,omitempty"`
	ObfsMode   string `json:"obfs_mode,omitempty"`
	ObfsHost   string `json:"obfs_host,omitempty"`
}

// Validate validates the proxy configuration
func (c *BaseProxyConfig) Validate() error {
	if c.Type == "" {
		return fmt.Errorf("proxy type is required")
	}

	switch c.Type {
	case "direct":
		// Direct proxy doesn't require additional configuration
		// It connects directly without proxy
	case "nonelane":
		// Nonelane (mTLS) proxy - CoreServer is optional with default value
		// If not provided, default will be used in registry
	case "door":
		// Door proxy type - validation will be done during registration process
		// The actual members are stored separately in the door proxy config
	case "vless", "shadowsocks":
		// These types are only valid as door proxy members
		return fmt.Errorf("type '%s' is only valid as a door proxy member", c.Type)
	default:
		return fmt.Errorf("unsupported proxy type: %s", c.Type)
	}

	return nil
}

// RoutingRulesConfig 路由规则总配置
type RoutingRulesConfig struct {
	GeoIP         *GeoIPConfig       `json:"geoip,omitempty"`         // GeoIP 路由配置
	BypassRules   *BypassRulesConfig `json:"bypassRules,omitempty"`   // 旁路规则配置
	IPDomainCache *CacheConfig       `json:"ipDomainCache,omitempty"` // IP-域名缓存配置
}

// GeoIPConfig GeoIP 路由配置
type GeoIPConfig struct {
	Enabled      bool   `json:"enabled"`      // 是否启用 GeoIP 路由
	DatabasePath string `json:"databasePath"` // GeoLite2 数据库路径
	ChinaDirect  bool   `json:"chinaDirect"`  // 中国 IP 是否直连（true=直连，false=加速）
}

// BypassRulesConfig 旁路规则配置
type BypassRulesConfig struct {
	Enabled        bool     `json:"enabled"`                  // 是否启用旁路规则
	Domains        []string `json:"domains,omitempty"`        // 域名列表（支持通配符，如 *.apple.com）
	DomainSuffixes []string `json:"domainSuffixes,omitempty"` // 域名后缀列表（如 .cn, .gov.cn）
	IPRanges       []string `json:"ipRanges,omitempty"`       // IP 段列表（CIDR 格式，如 192.168.0.0/16）
}

// CacheConfig IP-域名缓存配置
type CacheConfig struct {
	Enabled    bool   `json:"enabled"`    // 是否启用缓存
	MaxEntries int    `json:"maxEntries"` // 最大缓存条目数
	TTL        string `json:"ttl"`        // 缓存 TTL（如 "5m", "1h"）
}

// DNSPreResolutionConfig DNS预解析配置
type DNSPreResolutionConfig struct {
	Enabled         bool          `json:"enabled"`          // 是否启用DNS预解析
	Timeout         string        `json:"timeout"`          // 预解析超时时间（如 "10s"）
	ConcurrentLimit int           `json:"concurrentLimit"`  // 并发解析限制
	RetryOnFailure  bool          `json:"retryOnFailure"`   // 失败时是否重试
	CacheResults    bool          `json:"cacheResults"`     // 是否缓存预解析结果
	PreferIPv4      bool          `json:"preferIPv4"`       // 优先使用IPv4地址
	ForceResolve    bool          `json:"forceResolve"`     // 强制解析（即使是IP也尝试）
	MaxCacheTTL     string        `json:"maxCacheTTL"`      // 最大缓存TTL（如 "1h"）
	PrimaryDNS      string        `json:"primaryDNS"`       // 主DNS服务器
	FallbackDNS     string        `json:"fallbackDNS"`      // 回退DNS服务器
	SystemDNSFallback bool         `json:"systemDNSFallback"` // 是否回退到系统DNS
}

// GetTimeout 解析超时时间
func (c *DNSPreResolutionConfig) GetTimeout() time.Duration {
	if c.Timeout == "" {
		return 10 * time.Second
	}
	if duration, err := time.ParseDuration(c.Timeout); err == nil {
		return duration
	}
	return 10 * time.Second
}

// GetMaxCacheTTL 解析最大缓存TTL
func (c *DNSPreResolutionConfig) GetMaxCacheTTL() time.Duration {
	if c.MaxCacheTTL == "" {
		return 1 * time.Hour
	}
	if duration, err := time.ParseDuration(c.MaxCacheTTL); err == nil {
		return duration
	}
	return 1 * time.Hour
}

// GetPrimaryDNS 获取主DNS服务器
func (c *DNSPreResolutionConfig) GetPrimaryDNS() string {
	if c.PrimaryDNS == "" {
		return "8.8.8.8:53"
	}
	return c.PrimaryDNS
}

// GetFallbackDNS 获取回退DNS服务器
func (c *DNSPreResolutionConfig) GetFallbackDNS() string {
	if c.FallbackDNS == "" {
		return "223.5.5.5:53"
	}
	return c.FallbackDNS
}

// Validate 验证DNS预解析配置
func (c *DNSPreResolutionConfig) Validate() error {
	if c.Enabled {
		if c.ConcurrentLimit <= 0 {
			return fmt.Errorf("concurrentLimit must be positive when DNS pre-resolution is enabled")
		}
		if c.ConcurrentLimit > 50 {
			return fmt.Errorf("concurrentLimit should not exceed 50")
		}
		if timeout := c.GetTimeout(); timeout < 1*time.Second || timeout > 60*time.Second {
			return fmt.Errorf("timeout should be between 1s and 60s")
		}
	}
	return nil
}

// DoorProxyConfig Door 代理集合专用配置
type DoorProxyConfig struct {
	Type    string            `json:"type"`
	Members []DoorProxyMember `json:"members,omitempty"`
}

// Config 完整配置结构
type Config struct {
	APIServer        string                      `json:"api_server"`             // 必须配置：Token激活、刷新、Inbound的基础URL
	NacosServer      string                      `json:"nacos_server,omitempty"` // Nacos配置中心，可选，默认为 "http://nacos-config.nursor.org"
	CurrentProxy     string                      `json:"currentProxy"`
	BaseProxies      map[string]*BaseProxyConfig `json:"baseProxies"`
	DoorProxy        *DoorProxyConfig            `json:"doorProxy,omitempty"`            // Door 代理集合配置
	RoutingRules     *RoutingRulesConfig         `json:"routingRules,omitempty"`         // 路由规则配置
	DNSPreResolution *DNSPreResolutionConfig     `json:"dnsPreResolution,omitempty"`     // DNS预解析配置
}

// GetTokenActivateURL returns the complete Token activation URL
func (c *Config) GetTokenActivateURL() string {
	return fmt.Sprintf("%s/api/user/auth/new/activate", c.APIServer)
}

// GetPlanStatusURL returns the complete Plan status URL
func (c *Config) GetPlanStatusURL() string {
	return fmt.Sprintf("%s/api/user/auth/info/plan/info", c.APIServer)
}

// GetInboundsURL returns the complete Inbounds API URL
func (c *Config) GetInboundsURL() string {
	return fmt.Sprintf("%s/api/production/prod/sui/user/sui/inbounds", c.APIServer)
}

// GetRemoteConfigURL returns the complete remote configuration URL
func (c *Config) GetRemoteConfigURL() string {
	return fmt.Sprintf("%s/api/config", c.APIServer)
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if c.APIServer == "" {
		return fmt.Errorf("api_server is required in configuration")
	}

	// Validate DNS pre-resolution configuration
	if c.DNSPreResolution != nil {
		if err := c.DNSPreResolution.Validate(); err != nil {
			return fmt.Errorf("invalid DNS pre-resolution configuration: %w", err)
		}
	}

	return nil
}
