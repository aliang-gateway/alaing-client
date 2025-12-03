package config

import "fmt"

// ProxyConfig represents a proxy configuration
type ProxyConfig struct {
	Type        string             `json:"type"`
	IsDefault   bool               `json:"is_default"`
	IsDoorProxy bool               `json:"is_door_proxy"`
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
	default:
		return fmt.Errorf("unsupported proxy type: %s", c.Type)
	}

	return nil
}
