package config

import "fmt"

// ProxyConfig represents a proxy configuration
type ProxyConfig struct {
	Type        string             `json:"type"`
	IsDefault   bool               `json:"is_default"`
	IsDoorProxy bool               `json:"is_door_proxy"`
	VLESS       *VLESSConfig       `json:"vless,omitempty"`
	Shadowsocks *ShadowsocksConfig `json:"shadowsocks,omitempty"`

	// Door 代理集合专用
	Members     []DoorProxyMember  `json:"members,omitempty"`

	// Nonelane 代理专用
	CoreServer  string             `json:"core_server,omitempty"`
}

// DoorProxyMember represents a member in a door proxy collection
type DoorProxyMember struct {
	ShowName    string             `json:"showname"`     // 显示名称
	Type        string             `json:"type"`         // vless/shadowsocks
	Latency     int64              `json:"latency"`      // 延迟（毫秒）
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
	ObfsMode string `json:"obfs_mode,omitempty"`
	ObfsHost string `json:"obfs_host,omitempty"`
}

// Validate validates the proxy configuration
func (c *ProxyConfig) Validate() error {
	if c.Type == "" {
		return fmt.Errorf("proxy type is required")
	}

	switch c.Type {
	case "vless":
		if c.VLESS == nil {
			return fmt.Errorf("VLESS config is required for vless type")
		}
		if c.VLESS.Server == "" || c.VLESS.UUID == "" {
			return fmt.Errorf("VLESS server and UUID are required")
		}
	case "shadowsocks":
		if c.Shadowsocks == nil {
			return fmt.Errorf("Shadowsocks config is required for shadowsocks type")
		}
		if c.Shadowsocks.Server == "" || c.Shadowsocks.Password == "" {
			return fmt.Errorf("Shadowsocks server and password are required")
		}
	case "direct":
		// Direct proxy doesn't require additional configuration
		// It connects directly without proxy
	case "nonelane":
		// Nonelane (mTLS) proxy - CoreServer is optional with default value
		// If not provided, default will be used in registry
	case "door":
		// Door proxy collection - must have at least one member
		if len(c.Members) == 0 {
			return fmt.Errorf("door proxy must have at least one member")
		}
		// Validate each member
		for i, member := range c.Members {
			if member.ShowName == "" {
				return fmt.Errorf("door member %d: showname is required", i)
			}
			if member.Type == "" {
				return fmt.Errorf("door member %d (%s): type is required", i, member.ShowName)
			}
			// Validate member config based on type
			switch member.Type {
			case "vless":
				if member.VLESS == nil {
					return fmt.Errorf("door member %d (%s): VLESS config is required", i, member.ShowName)
				}
				if member.VLESS.Server == "" || member.VLESS.UUID == "" {
					return fmt.Errorf("door member %d (%s): VLESS server and UUID are required", i, member.ShowName)
				}
			case "shadowsocks":
				if member.Shadowsocks == nil {
					return fmt.Errorf("door member %d (%s): Shadowsocks config is required", i, member.ShowName)
				}
				if member.Shadowsocks.Server == "" || member.Shadowsocks.Password == "" {
					return fmt.Errorf("door member %d (%s): Shadowsocks server and password are required", i, member.ShowName)
				}
			default:
				return fmt.Errorf("door member %d (%s): unsupported type %s", i, member.ShowName, member.Type)
			}
		}
	default:
		return fmt.Errorf("unsupported proxy type: %s", c.Type)
	}

	return nil
}
