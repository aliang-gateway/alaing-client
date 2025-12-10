package config

import "fmt"

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
	Server         string `json:"server"`
	UUID           string `json:"uuid"`
	Flow           string `json:"flow,omitempty"`
	TLSEnabled     bool   `json:"tls_enabled"`
	SNI            string `json:"sni,omitempty"`
	RealityEnabled bool   `json:"reality_enabled"`
	PublicKey      string `json:"public_key,omitempty"`
	ShortID        string `json:"short_id,omitempty"`
	ShortIDList    string `json:"short_id_list,omitempty"`
}

// ShadowsocksConfig represents Shadowsocks protocol configuration
type ShadowsocksConfig struct {
	Server   string `json:"server"`
	Method   string `json:"method"`
	Password string `json:"password"`
	Username string `json:"username,omitempty"`
	ObfsMode string `json:"obfs_mode,omitempty"`
	ObfsHost string `json:"obfs_host,omitempty"`
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

// EngineConfig 引擎配置（对应 processor/config.EngineConf），以前的tun2socks配置key
type EngineConfig struct {
	MTU                      int    `json:"mtu"`
	Mark                     int    `json:"fwmark"`
	RestAPI                  string `json:"restapi"`
	Device                   string `json:"device"`
	LogLevel                 string `json:"loglevel"`
	Interface                string `json:"interface"`
	TCPModerateReceiveBuffer bool   `json:"tcp-moderate-receive-buffer"`
	TCPSendBufferSize        string `json:"tcp-send-buffer-size"`
	TCPReceiveBufferSize     string `json:"tcp-receive-buffer-size"`
	MulticastGroups          string `json:"multicast-groups"`
	TUNPreUp                 string `json:"tun-pre-up"`
	TUNPostUp                string `json:"tun-post-up"`
	UDPTimeout               string `json:"udp-timeout"` // 字符串格式，需要解析为 time.Duration
}

// DoorProxyConfig Door 代理集合专用配置
type DoorProxyConfig struct {
	Type    string            `json:"type"`
	Members []DoorProxyMember `json:"members,omitempty"`
}

// Config 完整配置结构
type Config struct {
	Engine       *EngineConfig               `json:"engine"`
	CurrentProxy string                      `json:"currentProxy"`
	BaseProxies  map[string]*BaseProxyConfig `json:"baseProxies"`
	DoorProxy    *DoorProxyConfig            `json:"doorProxy,omitempty"`    // Door 代理集合配置
	RoutingRules *RoutingRulesConfig         `json:"routingRules,omitempty"` // 路由规则配置
}
