package config

import (
	"encoding/json"
	"strings"
	"testing"
)

// TestShadowTLSPluginOptsValidate tests ShadowTLSPluginOpts.Validate()
func TestShadowTLSPluginOptsValidate(t *testing.T) {
	tests := []struct {
		name    string
		opts    *ShadowTLSPluginOpts
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid configuration",
			opts: &ShadowTLSPluginOpts{
				Host:     "www.bing.com",
				Password: "SecurePassword123",
				Version:  3,
			},
			wantErr: false,
		},
		{
			name:    "nil opts",
			opts:    nil,
			wantErr: true,
			errMsg:  "plugin_opts is required when plugin='shadow-tls'",
		},
		{
			name: "missing host",
			opts: &ShadowTLSPluginOpts{
				Password: "SecurePassword123",
				Version:  3,
			},
			wantErr: true,
			errMsg:  "plugin_opts.host is required",
		},
		{
			name: "empty password",
			opts: &ShadowTLSPluginOpts{
				Host:     "www.bing.com",
				Password: "",
				Version:  3,
			},
			wantErr: true,
			errMsg:  "plugin_opts.password is required and cannot be empty",
		},
		{
			name: "password too short",
			opts: &ShadowTLSPluginOpts{
				Host:     "www.bing.com",
				Password: "short",
				Version:  3,
			},
			wantErr: true,
			errMsg:  "plugin_opts.password must be at least 8 characters",
		},
		{
			name: "invalid version 0",
			opts: &ShadowTLSPluginOpts{
				Host:     "www.bing.com",
				Password: "SecurePassword123",
				Version:  0,
			},
			wantErr: true,
			errMsg:  "plugin_opts.version must be 1, 2, or 3",
		},
		{
			name: "invalid version 4",
			opts: &ShadowTLSPluginOpts{
				Host:     "www.bing.com",
				Password: "SecurePassword123",
				Version:  4,
			},
			wantErr: true,
			errMsg:  "plugin_opts.version must be 1, 2, or 3",
		},
		{
			name: "version 1 is valid",
			opts: &ShadowTLSPluginOpts{
				Host:     "www.bing.com",
				Password: "SecurePassword123",
				Version:  1,
			},
			wantErr: false,
		},
		{
			name: "version 2 is valid",
			opts: &ShadowTLSPluginOpts{
				Host:     "www.bing.com",
				Password: "SecurePassword123",
				Version:  2,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.opts.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil && tt.errMsg != "" && err.Error() != tt.errMsg {
				t.Errorf("Validate() error message = %v, want %v", err.Error(), tt.errMsg)
			}
		})
	}
}

// TestShadowsocksConfigValidate tests ShadowsocksConfig.Validate()
func TestShadowsocksConfigValidate(t *testing.T) {
	tests := []struct {
		name    string
		config  *ShadowsocksConfig
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid config without plugin",
			config: &ShadowsocksConfig{
				Server:     "192.168.1.100",
				ServerPort: 8388,
				Method:     "aes-256-gcm",
				Password:   "MyPassword123",
			},
			wantErr: false,
		},
		{
			name: "valid config with shadow-tls plugin",
			config: &ShadowsocksConfig{
				Server:     "151.242.165.151",
				ServerPort: 443,
				Method:     "chacha20-ietf-poly1305",
				Password:   "I8U3GD4pziEyIeQwTqd52CGLisU5boCwg6FBU9KpARs=",
				Plugin:     "shadow-tls",
				PluginOpts: &ShadowTLSPluginOpts{
					Host:     "www.bing.com",
					Password: "I8U3GD4pziEyIeQwTqd52CGLisU5boCwg6FBU9KpARs=",
					Version:  3,
				},
			},
			wantErr: false,
		},
		{
			name: "missing server_host",
			config: &ShadowsocksConfig{
				ServerPort: 8388,
				Method:     "aes-256-gcm",
				Password:   "MyPassword123",
			},
			wantErr: true,
			errMsg:  "server_host is required",
		},
		{
			name: "missing server_port",
			config: &ShadowsocksConfig{
				Server:   "192.168.1.100",
				Method:   "aes-256-gcm",
				Password: "MyPassword123",
			},
			wantErr: true,
			errMsg:  "server_port is required",
		},
		{
			name: "missing method",
			config: &ShadowsocksConfig{
				Server:     "192.168.1.100",
				ServerPort: 8388,
				Password:   "MyPassword123",
			},
			wantErr: true,
			errMsg:  "method is required",
		},
		{
			name: "missing password",
			config: &ShadowsocksConfig{
				Server:     "192.168.1.100",
				ServerPort: 8388,
				Method:     "aes-256-gcm",
			},
			wantErr: true,
			errMsg:  "password is required",
		},
		{
			name: "shadow-tls plugin without plugin_opts",
			config: &ShadowsocksConfig{
				Server:     "151.242.165.151",
				ServerPort: 443,
				Method:     "chacha20-ietf-poly1305",
				Password:   "MyPassword123",
				Plugin:     "shadow-tls",
			},
			wantErr: true,
			errMsg:  "plugin_opts is required when plugin='shadow-tls'",
		},
		{
			name: "unsupported plugin",
			config: &ShadowsocksConfig{
				Server:     "151.242.165.151",
				ServerPort: 443,
				Method:     "chacha20-ietf-poly1305",
				Password:   "MyPassword123",
				Plugin:     "unsupported-plugin",
			},
			wantErr: true,
			errMsg:  "unsupported plugin: unsupported-plugin",
		},
		{
			name: "shadow-tls plugin with invalid plugin_opts",
			config: &ShadowsocksConfig{
				Server:     "151.242.165.151",
				ServerPort: 443,
				Method:     "chacha20-ietf-poly1305",
				Password:   "MyPassword123",
				Plugin:     "shadow-tls",
				PluginOpts: &ShadowTLSPluginOpts{
					Host:     "",
					Password: "short",
					Version:  3,
				},
			},
			wantErr: true,
			errMsg:  "plugin_opts.host is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil && tt.errMsg != "" && err.Error() != tt.errMsg {
				t.Errorf("Validate() error message = %v, want %v", err.Error(), tt.errMsg)
			}
		})
	}
}

// TestSocks5ConfigValidate tests Socks5Config.Validate()
func TestSocks5ConfigValidate(t *testing.T) {
	tests := []struct {
		name    string
		config  *Socks5Config
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid without auth",
			config: &Socks5Config{
				Server:     "127.0.0.1",
				ServerPort: 1080,
			},
			wantErr: false,
		},
		{
			name: "valid with auth",
			config: &Socks5Config{
				Server:     "127.0.0.1",
				ServerPort: 1080,
				Username:   "user",
				Password:   "pass",
			},
			wantErr: false,
		},
		{
			name: "missing server_host",
			config: &Socks5Config{
				ServerPort: 1080,
			},
			wantErr: true,
			errMsg:  "server_host is required",
		},
		{
			name: "missing server_port",
			config: &Socks5Config{
				Server: "127.0.0.1",
			},
			wantErr: true,
			errMsg:  "server_port is required",
		},
		{
			name: "username without password",
			config: &Socks5Config{
				Server:     "127.0.0.1",
				ServerPort: 1080,
				Username:   "user",
			},
			wantErr: true,
			errMsg:  "username and password must be provided together",
		},
		{
			name: "password without username",
			config: &Socks5Config{
				Server:     "127.0.0.1",
				ServerPort: 1080,
				Password:   "pass",
			},
			wantErr: true,
			errMsg:  "username and password must be provided together",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil && tt.errMsg != "" && err.Error() != tt.errMsg {
				t.Errorf("Validate() error message = %v, want %v", err.Error(), tt.errMsg)
			}
		})
	}
}

func TestConfigValidate_NewModelBridge_MapsRuntimeFields(t *testing.T) {
	payload := []byte(`{
		"api_server": "https://api.example.com",
		"core": {
			"aliangServer": {
				"type": "aliang",
				"core_server": "ai-gateway.nursor.org:443"
			}
		},
		"customer": {
			"proxy": {
				"type": "socks",
				"server": "127.0.0.1:1080",
				"username": "u",
				"password": "p"
			},
			"ai_rules": {
				"openai": {
					"enable": true,
					"exclude": ["api.openai.com", "cdn.openai.com"]
				},
				"claude": {
					"enable": false,
					"exclude": ["claude.ai"]
				}
			},
			"proxy_rules": ["domains,cursor.com,proxy"]
		}
	}`)

	var cfg Config
	if err := json.Unmarshal(payload, &cfg); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}

	if cfg.CurrentProxy != "socks" {
		t.Fatalf("CurrentProxy = %q, want socks", cfg.CurrentProxy)
	}
	if cfg.SocksProxy == nil {
		t.Fatal("SocksProxy is nil, want bridged socks config")
	}
	if cfg.SocksProxy.Server != "127.0.0.1" || cfg.SocksProxy.ServerPort != 1080 {
		t.Fatalf("SocksProxy = %#v, want host 127.0.0.1 port 1080", cfg.SocksProxy)
	}
	if cfg.BaseProxies == nil || cfg.BaseProxies["aliang"] == nil {
		t.Fatal("BaseProxies[\"aliang\"] missing after bridge")
	}
	if got := cfg.BaseProxies["aliang"].CoreServer; got != "ai-gateway.nursor.org:443" {
		t.Fatalf("BaseProxies[\"aliang\"].CoreServer = %q", got)
	}
	if len(cfg.SNIAllowlist) != 2 {
		t.Fatalf("SNIAllowlist len = %d, want 2", len(cfg.SNIAllowlist))
	}

	// Verify backward-compatible proxy_rules deserialization (legacy []string)
	if cfg.Customer.ProxyRules == nil {
		t.Fatal("ProxyRules is nil, want backward-compatible parsing")
	}
	if !cfg.Customer.ProxyRules.Enabled {
		t.Fatal("ProxyRules.Enabled = false, want true (legacy array defaults to enabled)")
	}
	if len(cfg.Customer.ProxyRules.Rules) != 1 || cfg.Customer.ProxyRules.Rules[0] != "domains,cursor.com,proxy" {
		t.Fatalf("ProxyRules.Rules = %v, want [domains,cursor.com,proxy]", cfg.Customer.ProxyRules.Rules)
	}
}

func TestProxyRulesConfig_Unmarshal(t *testing.T) {
	t.Run("legacy array format", func(t *testing.T) {
		data := []byte(`["a","b","c"]`)
		var cfg CustomerProxyRulesConfig
		if err := json.Unmarshal(data, &cfg); err != nil {
			t.Fatalf("Unmarshal() error = %v", err)
		}
		if !cfg.Enabled {
			t.Fatal("Enabled = false, want true")
		}
		if len(cfg.Rules) != 3 {
			t.Fatalf("Rules len = %d, want 3", len(cfg.Rules))
		}
	})

	t.Run("object format", func(t *testing.T) {
		data := []byte(`{"enabled":false,"rules":["x","y"]}`)
		var cfg CustomerProxyRulesConfig
		if err := json.Unmarshal(data, &cfg); err != nil {
			t.Fatalf("Unmarshal() error = %v", err)
		}
		if cfg.Enabled {
			t.Fatal("Enabled = true, want false")
		}
		if len(cfg.Rules) != 2 {
			t.Fatalf("Rules len = %d, want 2", len(cfg.Rules))
		}
	})

	t.Run("null value", func(t *testing.T) {
		data := []byte(`null`)
		var cfg CustomerProxyRulesConfig
		if err := json.Unmarshal(data, &cfg); err != nil {
			t.Fatalf("Unmarshal() error = %v", err)
		}
	})
}

func TestConfigValidate_CustomerProxyTypeEnum(t *testing.T) {
	payload := []byte(`{
		"api_server": "https://api.example.com",
		"customer": {
			"proxy": {
				"type": "socks5",
				"server": "127.0.0.1:1080"
			}
		}
	}`)

	var cfg Config
	if err := json.Unmarshal(payload, &cfg); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	err := cfg.Validate()
	if err == nil {
		t.Fatal("Validate() error = nil, want enum validation error")
	}
	if !strings.Contains(err.Error(), "customer.proxy.type must be one of [http socks]") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestConfigValidate_ForbidUnknownCustomerField(t *testing.T) {
	payload := []byte(`{
		"api_server": "https://api.example.com",
		"customer": {
			"proxy": {
				"type": "http",
				"server": "127.0.0.1:1080"
			},
			"forbidden": true
		}
	}`)

	var cfg Config
	if err := json.Unmarshal(payload, &cfg); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	err := cfg.Validate()
	if err == nil {
		t.Fatal("Validate() error = nil, want forbidden field error")
	}
	if !strings.Contains(err.Error(), "customer.forbidden is forbidden") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestConfigValidate_ForbidUnknownAIRulesField(t *testing.T) {
	payload := []byte(`{
		"api_server": "https://api.example.com",
		"customer": {
			"proxy": {
				"type": "http",
				"server": "127.0.0.1:1080"
			},
			"ai_rules": {
				"openai": {
					"enable": true,
					"exclude": ["api.openai.com"],
					"mode": "all"
				}
			}
		}
	}`)

	var cfg Config
	if err := json.Unmarshal(payload, &cfg); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	err := cfg.Validate()
	if err == nil {
		t.Fatal("Validate() error = nil, want forbidden ai_rules field error")
	}
	if !strings.Contains(err.Error(), "customer.ai_rules.openai.mode is forbidden") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestConfigValidate_AIRulesEnableRequired(t *testing.T) {
	payload := []byte(`{
		"api_server": "https://api.example.com",
		"customer": {
			"proxy": {
				"type": "http",
				"server": "127.0.0.1:1080"
			},
			"ai_rules": {
				"openai": {
					"exclude": ["api.openai.com"]
				}
			}
		}
	}`)

	var cfg Config
	if err := json.Unmarshal(payload, &cfg); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	err := cfg.Validate()
	if err == nil {
		t.Fatal("Validate() error = nil, want missing enable error")
	}
	if !strings.Contains(err.Error(), "customer.ai_rules.openai.enable is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}
