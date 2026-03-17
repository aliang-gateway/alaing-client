package config

import (
	"fmt"
	"time"
)

// BaseProxyConfig represents a proxy configuration
type BaseProxyConfig struct {
	Type string `json:"type"`
	// Aliang 代理专用
	CoreServer string `json:"core_server,omitempty"`
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

	// ShadowTLS plugin support
	Plugin     string               `json:"plugin,omitempty"`
	PluginOpts *ShadowTLSPluginOpts `json:"plugin_opts,omitempty"`
}

// Socks5Config represents SOCKS5 protocol configuration
type Socks5Config struct {
	Server     string `json:"server_host"`
	ServerPort uint16 `json:"server_port"`
	Username   string `json:"username,omitempty"`
	Password   string `json:"password,omitempty"`
}

// Validate validates SOCKS5 configuration
func (c *Socks5Config) Validate() error {
	if c.Server == "" {
		return fmt.Errorf("server_host is required")
	}
	if c.ServerPort == 0 {
		return fmt.Errorf("server_port is required")
	}
	if (c.Username == "") != (c.Password == "") {
		return fmt.Errorf("username and password must be provided together")
	}
	return nil
}

// ShadowTLSPluginOpts represents ShadowTLS plugin configuration
type ShadowTLSPluginOpts struct {
	Host     string `json:"host"`     // TLS camouflage domain (e.g., www.bing.com)
	Password string `json:"password"` // ShadowTLS authentication password
	Version  int    `json:"version"`  // Protocol version (1, 2, or 3)
}

// Validate validates ShadowTLS plugin options
func (o *ShadowTLSPluginOpts) Validate() error {
	if o == nil {
		return fmt.Errorf("plugin_opts is required when plugin='shadow-tls'")
	}
	if o.Host == "" {
		return fmt.Errorf("plugin_opts.host is required")
	}
	if o.Password == "" {
		return fmt.Errorf("plugin_opts.password is required and cannot be empty")
	}
	if len(o.Password) < 8 {
		return fmt.Errorf("plugin_opts.password must be at least 8 characters")
	}
	if o.Version != 1 && o.Version != 2 && o.Version != 3 {
		return fmt.Errorf("plugin_opts.version must be 1, 2, or 3")
	}
	return nil
}

// Validate validates Shadowsocks configuration including plugin settings
func (c *ShadowsocksConfig) Validate() error {
	if c.Server == "" {
		return fmt.Errorf("server_host is required")
	}
	if c.ServerPort == 0 {
		return fmt.Errorf("server_port is required")
	}
	if c.Method == "" {
		return fmt.Errorf("method is required")
	}
	if c.Password == "" {
		return fmt.Errorf("password is required")
	}

	// Validate plugin configuration
	if c.Plugin != "" {
		if c.Plugin == "shadow-tls" {
			if c.PluginOpts == nil {
				return fmt.Errorf("plugin_opts is required when plugin='shadow-tls'")
			}
			if err := c.PluginOpts.Validate(); err != nil {
				return err
			}
		} else {
			return fmt.Errorf("unsupported plugin: %s", c.Plugin)
		}
	}

	return nil
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
	case "aliang":
		// Aliang (mTLS) proxy - CoreServer is optional with default value
		// If not provided, default will be used in registry
	default:
		return fmt.Errorf("unsupported proxy type: %s", c.Type)
	}

	return nil
}

// DNSPreResolutionConfig DNS预解析配置
type DNSPreResolutionConfig struct {
	Enabled           bool   `json:"enabled"`           // 是否启用DNS预解析
	Timeout           string `json:"timeout"`           // 预解析超时时间（如 "10s"）
	ConcurrentLimit   int    `json:"concurrentLimit"`   // 并发解析限制
	RetryOnFailure    bool   `json:"retryOnFailure"`    // 失败时是否重试
	CacheResults      bool   `json:"cacheResults"`      // 是否缓存预解析结果
	PreferIPv4        bool   `json:"preferIPv4"`        // 优先使用IPv4地址
	ForceResolve      bool   `json:"forceResolve"`      // 强制解析（即使是IP也尝试）
	MaxCacheTTL       string `json:"maxCacheTTL"`       // 最大缓存TTL（如 "1h"）
	PrimaryDNS        string `json:"primaryDNS"`        // 主DNS服务器
	FallbackDNS       string `json:"fallbackDNS"`       // 回退DNS服务器
	SystemDNSFallback bool   `json:"systemDNSFallback"` // 是否回退到系统DNS
}

// GetDNSPreResolutionConfig 获取DNS预解析配置
func GetDNSPreResolutionConfig() *DNSPreResolutionConfig {
	return &DNSPreResolutionConfig{
		Enabled:           true,
		Timeout:           "10s",
		ConcurrentLimit:   10,
		RetryOnFailure:    true,
		CacheResults:      true,
		PreferIPv4:        true,
		ForceResolve:      false,
		MaxCacheTTL:       "1h",
		PrimaryDNS:        "8.8.8.8:53",
		FallbackDNS:       "223.5.5.5:53",
		SystemDNSFallback: true,
	}
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

// Config 完整配置结构
type Config struct {
	APIServer        string                      `json:"api_server"` // 必须配置：Token激活、刷新、Inbound的基础URL
	CurrentProxy     string                      `json:"currentProxy"`
	BaseProxies      map[string]*BaseProxyConfig `json:"baseProxies"`
	DNSPreResolution *DNSPreResolutionConfig     `json:"dnsPreResolution,omitempty"` // DNS预解析配置
	SocksProxy       *Socks5Config               `json:"socksProxy,omitempty"`       // 可选：默认 SOCKS5 出站
	SNIAllowlist     []string                    `json:"sni_allowlist,omitempty"`    // SNI 允许列表（命中则 MITM 并转发到 Aliang）
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

	if c.SocksProxy != nil {
		if err := c.SocksProxy.Validate(); err != nil {
			return fmt.Errorf("invalid socksProxy configuration: %w", err)
		}
	}

	return nil
}
