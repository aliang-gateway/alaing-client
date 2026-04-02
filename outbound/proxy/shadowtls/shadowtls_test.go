package shadowtls

import (
	"strings"
	"testing"

	"aliang.one/nursorgate/inbound/tun/metadata"
	"aliang.one/nursorgate/outbound/proxy/proto"
	"aliang.one/nursorgate/processor/config"
)

// TestNew tests the New() factory function for creating ShadowTLS proxy instances
func TestNew(t *testing.T) {
	tests := []struct {
		name    string
		config  *config.ShadowsocksConfig
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid ShadowTLS configuration",
			config: &config.ShadowsocksConfig{
				Server:     "151.242.165.151",
				ServerPort: 443,
				Method:     "chacha20-ietf-poly1305",
				Password:   "I8U3GD4pziEyIeQwTqd52CGLisU5boCwg6FBU9KpARs=",
				Plugin:     "shadow-tls",
				PluginOpts: &config.ShadowTLSPluginOpts{
					Host:     "www.bing.com",
					Password: "I8U3GD4pziEyIeQwTqd52CGLisU5boCwg6FBU9KpARs=",
					Version:  3,
				},
			},
			wantErr: false,
		},
		{
			name:    "nil config",
			config:  nil,
			wantErr: true,
			errMsg:  "is nil",
		},
		{
			name: "missing plugin field",
			config: &config.ShadowsocksConfig{
				Server:     "151.242.165.151",
				ServerPort: 443,
				Method:     "chacha20-ietf-poly1305",
				Password:   "I8U3GD4pziEyIeQwTqd52CGLisU5boCwg6FBU9KpARs=",
				// Plugin field missing
				PluginOpts: &config.ShadowTLSPluginOpts{
					Host:     "www.bing.com",
					Password: "I8U3GD4pziEyIeQwTqd52CGLisU5boCwg6FBU9KpARs=",
					Version:  3,
				},
			},
			wantErr: true,
			errMsg:  "plugin must be 'shadow-tls'",
		},
		{
			name: "wrong plugin type",
			config: &config.ShadowsocksConfig{
				Server:     "151.242.165.151",
				ServerPort: 443,
				Method:     "chacha20-ietf-poly1305",
				Password:   "I8U3GD4pziEyIeQwTqd52CGLisU5boCwg6FBU9KpARs=",
				Plugin:     "v2ray-plugin",
				PluginOpts: &config.ShadowTLSPluginOpts{
					Host:     "www.bing.com",
					Password: "I8U3GD4pziEyIeQwTqd52CGLisU5boCwg6FBU9KpARs=",
					Version:  3,
				},
			},
			wantErr: true,
			errMsg:  "unsupported plugin: v2ray-plugin",
		},
		{
			name: "missing plugin_opts",
			config: &config.ShadowsocksConfig{
				Server:     "151.242.165.151",
				ServerPort: 443,
				Method:     "chacha20-ietf-poly1305",
				Password:   "I8U3GD4pziEyIeQwTqd52CGLisU5boCwg6FBU9KpARs=",
				Plugin:     "shadow-tls",
				PluginOpts: nil,
			},
			wantErr: true,
			errMsg:  "plugin_opts is required",
		},
		{
			name: "invalid server_host (validation error)",
			config: &config.ShadowsocksConfig{
				Server:     "", // Missing server
				ServerPort: 443,
				Method:     "chacha20-ietf-poly1305",
				Password:   "I8U3GD4pziEyIeQwTqd52CGLisU5boCwg6FBU9KpARs=",
				Plugin:     "shadow-tls",
				PluginOpts: &config.ShadowTLSPluginOpts{
					Host:     "www.bing.com",
					Password: "I8U3GD4pziEyIeQwTqd52CGLisU5boCwg6FBU9KpARs=",
					Version:  3,
				},
			},
			wantErr: true,
			errMsg:  "server_host is required",
		},
		{
			name: "invalid plugin_opts (short password)",
			config: &config.ShadowsocksConfig{
				Server:     "151.242.165.151",
				ServerPort: 443,
				Method:     "chacha20-ietf-poly1305",
				Password:   "I8U3GD4pziEyIeQwTqd52CGLisU5boCwg6FBU9KpARs=",
				Plugin:     "shadow-tls",
				PluginOpts: &config.ShadowTLSPluginOpts{
					Host:     "www.bing.com",
					Password: "short", // Too short
					Version:  3,
				},
			},
			wantErr: true,
			errMsg:  "password must be at least 8 characters",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			proxy, err := New(tt.config)

			if (err != nil) != tt.wantErr {
				t.Errorf("New() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err != nil && tt.errMsg != "" {
				// Use substring match for error messages since new error format includes structured info
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("New() error message = %v, should contain %v", err.Error(), tt.errMsg)
				}
			}

			if err == nil {
				// Verify proxy instance is correctly initialized
				if proxy == nil {
					t.Error("New() returned nil proxy without error")
					return
				}

				// Verify Proto() returns correct protocol
				if proxy.Proto() != proto.ShadowTLS {
					t.Errorf("Proto() = %v, want %v", proxy.Proto(), proto.ShadowTLS)
				}

				// Verify Addr() returns correct address
				expectedAddr := "151.242.165.151:443"
				if proxy.Addr() != expectedAddr {
					t.Errorf("Addr() = %v, want %v", proxy.Addr(), expectedAddr)
				}

				// Verify internal fields are set correctly
				if proxy.server != tt.config.Server {
					t.Errorf("server = %v, want %v", proxy.server, tt.config.Server)
				}
				if proxy.port != tt.config.ServerPort {
					t.Errorf("port = %v, want %v", proxy.port, tt.config.ServerPort)
				}
				if proxy.method != tt.config.Method {
					t.Errorf("method = %v, want %v", proxy.method, tt.config.Method)
				}
				if proxy.password != tt.config.Password {
					t.Errorf("password = %v, want %v", proxy.password, tt.config.Password)
				}
				if proxy.tlsHost != tt.config.PluginOpts.Host {
					t.Errorf("tlsHost = %v, want %v", proxy.tlsHost, tt.config.PluginOpts.Host)
				}
				if proxy.tlsPassword != tt.config.PluginOpts.Password {
					t.Errorf("tlsPassword = %v, want %v", proxy.tlsPassword, tt.config.PluginOpts.Password)
				}
				if proxy.version != tt.config.PluginOpts.Version {
					t.Errorf("version = %v, want %v", proxy.version, tt.config.PluginOpts.Version)
				}
			}
		})
	}
}

// TestProxyInterface tests that ShadowTLS implements proxy.Proxy interface correctly
func TestProxyInterface(t *testing.T) {
	cfg := &config.ShadowsocksConfig{
		Server:     "151.242.165.151",
		ServerPort: 443,
		Method:     "chacha20-ietf-poly1305",
		Password:   "I8U3GD4pziEyIeQwTqd52CGLisU5boCwg6FBU9KpARs=",
		Plugin:     "shadow-tls",
		PluginOpts: &config.ShadowTLSPluginOpts{
			Host:     "www.bing.com",
			Password: "I8U3GD4pziEyIeQwTqd52CGLisU5boCwg6FBU9KpARs=",
			Version:  3,
		},
	}

	proxy, err := New(cfg)
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	// Test Addr() method
	t.Run("Addr", func(t *testing.T) {
		addr := proxy.Addr()
		expected := "151.242.165.151:443"
		if addr != expected {
			t.Errorf("Addr() = %v, want %v", addr, expected)
		}
	})

	// Test Proto() method
	t.Run("Proto", func(t *testing.T) {
		p := proxy.Proto()
		if p != proto.ShadowTLS {
			t.Errorf("Proto() = %v, want %v", p, proto.ShadowTLS)
		}

		// Verify proto string representation
		if p.String() != "shadowtls" {
			t.Errorf("Proto().String() = %v, want %v", p.String(), "shadowtls")
		}
	})
}

// TestNewWithDifferentVersions tests ShadowTLS proxy creation with different protocol versions
func TestNewWithDifferentVersions(t *testing.T) {
	versions := []int{1, 2, 3}

	for _, version := range versions {
		t.Run("version_"+string(rune('0'+version)), func(t *testing.T) {
			cfg := &config.ShadowsocksConfig{
				Server:     "151.242.165.151",
				ServerPort: 443,
				Method:     "chacha20-ietf-poly1305",
				Password:   "I8U3GD4pziEyIeQwTqd52CGLisU5boCwg6FBU9KpARs=",
				Plugin:     "shadow-tls",
				PluginOpts: &config.ShadowTLSPluginOpts{
					Host:     "www.bing.com",
					Password: "I8U3GD4pziEyIeQwTqd52CGLisU5boCwg6FBU9KpARs=",
					Version:  version,
				},
			}

			proxy, err := New(cfg)
			if err != nil {
				t.Errorf("New() with version %d failed: %v", version, err)
				return
			}

			if proxy.version != version {
				t.Errorf("proxy.version = %d, want %d", proxy.version, version)
			}
		})
	}
}

// TestNewWithDifferentEncryptionMethods tests ShadowTLS proxy creation with different encryption methods
func TestNewWithDifferentEncryptionMethods(t *testing.T) {
	tests := []struct {
		method      string
		shouldPass  bool
		description string
	}{
		{"aes-128-gcm", true, "AES-128-GCM is supported"},
		{"aes-256-gcm", true, "AES-256-GCM is supported"},
		{"chacha20-ietf-poly1305", true, "ChaCha20-IETF-Poly1305 is supported"},
		// Note: 2022-blake3 methods require additional dependencies
		// and may not be available in the current tun2socks version
		{"2022-blake3-aes-128-gcm", false, "Blake3-based ciphers may not be supported"},
		{"2022-blake3-aes-256-gcm", false, "Blake3-based ciphers may not be supported"},
	}

	for _, tt := range tests {
		t.Run("method_"+tt.method, func(t *testing.T) {
			cfg := &config.ShadowsocksConfig{
				Server:     "151.242.165.151",
				ServerPort: 443,
				Method:     tt.method,
				Password:   "I8U3GD4pziEyIeQwTqd52CGLisU5boCwg6FBU9KpARs=",
				Plugin:     "shadow-tls",
				PluginOpts: &config.ShadowTLSPluginOpts{
					Host:     "www.bing.com",
					Password: "I8U3GD4pziEyIeQwTqd52CGLisU5boCwg6FBU9KpARs=",
					Version:  3,
				},
			}

			proxy, err := New(cfg)

			if tt.shouldPass {
				if err != nil {
					t.Errorf("New() with method %s failed: %v (%s)", tt.method, err, tt.description)
					return
				}

				if proxy.method != tt.method {
					t.Errorf("proxy.method = %s, want %s", proxy.method, tt.method)
				}
			} else {
				// For unsupported methods, we expect an error
				if err == nil {
					t.Errorf("New() with method %s should have failed (%s)", tt.method, tt.description)
					return
				}

				// Check for cipher error message containing method name and "cipher not supported"
				if !strings.Contains(err.Error(), "shadowtls cipher error") ||
					!strings.Contains(err.Error(), tt.method) {
					t.Errorf("expected cipher error for method %s, got: %v", tt.method, err)
				}
			}
		})
	}
}

// TestDialContext tests the DialContext method of ShadowTLS proxy
func TestDialContext(t *testing.T) {
	cfg := &config.ShadowsocksConfig{
		Server:     "127.0.0.1", // Use localhost to fail fast
		ServerPort: 9999,        // Non-existent port
		Method:     "chacha20-ietf-poly1305",
		Password:   "I8U3GD4pziEyIeQwTqd52CGLisU5boCwg6FBU9KpARs=",
		Plugin:     "shadow-tls",
		PluginOpts: &config.ShadowTLSPluginOpts{
			Host:     "www.example.com",
			Password: "I8U3GD4pziEyIeQwTqd52CGLisU5boCwg6FBU9KpARs=",
			Version:  3,
		},
	}

	proxy, err := New(cfg)
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	tests := []struct {
		name       string
		wantErr    bool
		errPattern string // Check for connection error (not auth error since connection fails first)
	}{
		{
			name:       "TLS handshake fails - connection refused",
			wantErr:    true,
			errPattern: "connection error", // Layered error structure
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a minimal metadata object for testing
			var testMeta *metadata.Metadata

			conn, err := proxy.DialContext(nil, testMeta)

			if (err != nil) != tt.wantErr {
				t.Errorf("DialContext() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err != nil && tt.errPattern != "" {
				if !strings.Contains(err.Error(), tt.errPattern) {
					t.Errorf("DialContext() error message = %v, should contain %v", err.Error(), tt.errPattern)
				}
			}

			// Connection should be nil since connection fails
			if conn != nil {
				conn.Close()
			}
		})
	}
}

// TestDialUDP tests the DialUDP method of ShadowTLS proxy
func TestDialUDP(t *testing.T) {
	cfg := &config.ShadowsocksConfig{
		Server:     "151.242.165.151",
		ServerPort: 443,
		Method:     "chacha20-ietf-poly1305",
		Password:   "I8U3GD4pziEyIeQwTqd52CGLisU5boCwg6FBU9KpARs=",
		Plugin:     "shadow-tls",
		PluginOpts: &config.ShadowTLSPluginOpts{
			Host:     "www.bing.com",
			Password: "I8U3GD4pziEyIeQwTqd52CGLisU5boCwg6FBU9KpARs=",
			Version:  3,
		},
	}

	proxy, err := New(cfg)
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	tests := []struct {
		name    string
		target  string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "UDP is not supported",
			target:  "8.8.8.8:53",
			wantErr: true,
			errMsg:  "UDP",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a minimal metadata object for testing
			var testMeta *metadata.Metadata

			conn, err := proxy.DialUDP(testMeta)

			if (err != nil) != tt.wantErr {
				t.Errorf("DialUDP() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err != nil && tt.errMsg != "" {
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("DialUDP() error message = %v, should contain %v", err.Error(), tt.errMsg)
				}
			}

			// Connection should be nil
			if conn != nil {
				t.Error("DialUDP() returned non-nil connection for unsupported protocol")
				conn.Close()
			}
		})
	}
}

// TestConcurrentDialContext tests concurrent DialContext calls for race condition safety
func TestConcurrentDialContext(t *testing.T) {
	cfg := &config.ShadowsocksConfig{
		Server:     "151.242.165.151",
		ServerPort: 443,
		Method:     "chacha20-ietf-poly1305",
		Password:   "I8U3GD4pziEyIeQwTqd52CGLisU5boCwg6FBU9KpARs=",
		Plugin:     "shadow-tls",
		PluginOpts: &config.ShadowTLSPluginOpts{
			Host:     "www.bing.com",
			Password: "I8U3GD4pziEyIeQwTqd52CGLisU5boCwg6FBU9KpARs=",
			Version:  3,
		},
	}

	proxy, err := New(cfg)
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	// Test that concurrent calls to DialContext don't cause race conditions
	// This will fail with "not yet implemented" but should be thread-safe
	done := make(chan bool, 3)

	for i := 0; i < 3; i++ {
		go func() {
			var testMeta *metadata.Metadata
			_, _ = proxy.DialContext(nil, testMeta)
			done <- true
		}()
	}

	// Wait for all goroutines to complete
	for i := 0; i < 3; i++ {
		<-done
	}
}
