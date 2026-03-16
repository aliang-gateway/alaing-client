package config

import (
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
