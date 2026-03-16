package outbound

import (
	"strings"
	"testing"

	"nursor.org/nursorgate/outbound/proxy"
	"nursor.org/nursorgate/outbound/proxy/proto"
	"nursor.org/nursorgate/processor/config"
)

// TestCreateShadowsocksProxyWithShadowTLSPlugin tests protocol selection for ShadowTLS plugin
func TestCreateShadowsocksProxyWithShadowTLSPlugin(t *testing.T) {
	tests := []struct {
		name         string
		member       *config.DoorProxyMember
		expectedType proto.Proto
		wantErr      bool
		errMsg       string
	}{
		{
			name: "create ShadowTLS proxy with shadow-tls plugin",
			member: &config.DoorProxyMember{
				ShowName: "Japan Tokyo",
				Type:     "shadowsocks",
				Config: map[string]interface{}{
					"server_host": "151.242.165.151",
					"server_port": 443,
					"method":      "chacha20-ietf-poly1305",
					"password":    "I8U3GD4pziEyIeQwTqd52CGLisU5boCwg6FBU9KpARs=",
					"plugin":      "shadow-tls",
					"plugin_opts": map[string]interface{}{
						"host":     "www.bing.com",
						"password": "I8U3GD4pziEyIeQwTqd52CGLisU5boCwg6FBU9KpARs=",
						"version":  3,
					},
				},
			},
			expectedType: proto.ShadowTLS,
			wantErr:      false,
		},
		{
			name: "create standard Shadowsocks proxy without plugin",
			member: &config.DoorProxyMember{
				ShowName: "US New York",
				Type:     "shadowsocks",
				Config: map[string]interface{}{
					"server_host": "192.168.1.100",
					"server_port": 8388,
					"method":      "aes-256-gcm",
					"password":    "MyPassword123",
				},
			},
			expectedType: proto.Shadowsocks,
			wantErr:      false,
		},
		{
			name: "fail with missing plugin_opts",
			member: &config.DoorProxyMember{
				ShowName: "Invalid",
				Type:     "shadowsocks",
				Config: map[string]interface{}{
					"server_host": "151.242.165.151",
					"server_port": 443,
					"method":      "chacha20-ietf-poly1305",
					"password":    "I8U3GD4pziEyIeQwTqd52CGLisU5boCwg6FBU9KpARs=",
					"plugin":      "shadow-tls",
					// plugin_opts missing
				},
			},
			wantErr: true,
			errMsg:  "plugin_opts is required",
		},
		{
			name: "fail with invalid ShadowTLS password (too short)",
			member: &config.DoorProxyMember{
				ShowName: "Invalid",
				Type:     "shadowsocks",
				Config: map[string]interface{}{
					"server_host": "151.242.165.151",
					"server_port": 443,
					"method":      "chacha20-ietf-poly1305",
					"password":    "I8U3GD4pziEyIeQwTqd52CGLisU5boCwg6FBU9KpARs=",
					"plugin":      "shadow-tls",
					"plugin_opts": map[string]interface{}{
						"host":     "www.bing.com",
						"password": "short", // Too short
						"version":  3,
					},
				},
			},
			wantErr: true,
			errMsg:  "password must be at least 8 characters",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, err := createShadowsocksProxy(tt.member)

			if (err != nil) != tt.wantErr {
				t.Errorf("createShadowsocksProxy() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err != nil && tt.errMsg != "" {
				// Use substring match since error format includes structured information
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("createShadowsocksProxy() error message = %v, should contain %v", err.Error(), tt.errMsg)
				}
			}

			if err == nil {
				if p == nil {
					t.Error("createShadowsocksProxy() returned nil proxy without error")
					return
				}

				if p.Proto() != tt.expectedType {
					t.Errorf("createShadowsocksProxy() proto = %v, want %v", p.Proto(), tt.expectedType)
				}

				// Verify proxy address is correct
				if p.Addr() == "" {
					t.Error("createShadowsocksProxy() returned proxy with empty address")
				}
			}
		})
	}
}

// TestDoorProxyRegistrationWithShadowTLS tests registering door proxy members with ShadowTLS
func TestDoorProxyRegistrationWithShadowTLS(t *testing.T) {
	doorCfg := &config.DoorProxyConfig{
		Type: "door",
		Members: []config.DoorProxyMember{
			{
				ShowName: "Japan Tokyo ShadowTLS",
				Type:     "shadowsocks",
				Latency:  100,
				Status:   "success",
				Config: map[string]interface{}{
					"server_host": "151.242.165.151",
					"server_port": 443,
					"method":      "chacha20-ietf-poly1305",
					"password":    "I8U3GD4pziEyIeQwTqd52CGLisU5boCwg6FBU9KpARs=",
					"plugin":      "shadow-tls",
					"plugin_opts": map[string]interface{}{
						"host":     "www.bing.com",
						"password": "I8U3GD4pziEyIeQwTqd52CGLisU5boCwg6FBU9KpARs=",
						"version":  3,
					},
				},
			},
			{
				ShowName: "US New York Standard",
				Type:     "shadowsocks",
				Latency:  150,
				Status:   "success",
				Config: map[string]interface{}{
					"server_host": "192.168.1.100",
					"server_port": 8388,
					"method":      "aes-256-gcm",
					"password":    "MyPassword123",
				},
			},
		},
	}

	reg := &Registry{
		proxies: make(map[string]proxy.Proxy),
	}

	// Register door proxy
	err := reg.RegisterDoorFromConfig(doorCfg)
	if err != nil {
		t.Fatalf("RegisterDoorFromConfig() failed: %v", err)
	}

	// Verify door group was registered
	if reg.doorGroup == nil {
		t.Fatal("door group not registered")
	}

	// Verify both members were registered
	if reg.doorGroup.Count() != 2 {
		t.Errorf("expected 2 door members, got %d", reg.doorGroup.Count())
	}

	// Get and verify ShadowTLS member
	shadowtlsProxy, err := reg.GetDoor("Japan Tokyo ShadowTLS")
	if err != nil {
		t.Fatalf("GetDoor(Japan Tokyo ShadowTLS) failed: %v", err)
	}
	if shadowtlsProxy.Proto() != proto.ShadowTLS {
		t.Errorf("expected ShadowTLS proxy, got %v", shadowtlsProxy.Proto())
	}

	// Get and verify standard Shadowsocks member
	standardProxy, err := reg.GetDoor("US New York Standard")
	if err != nil {
		t.Fatalf("GetDoor(US New York Standard) failed: %v", err)
	}
	if standardProxy.Proto() != proto.Shadowsocks {
		t.Errorf("expected Shadowsocks proxy, got %v", standardProxy.Proto())
	}
}

// TestProtocolSelectionLogic tests that correct proxy type is selected based on plugin field
func TestProtocolSelectionLogic(t *testing.T) {
	tests := []struct {
		name         string
		plugin       string
		expectedType proto.Proto
	}{
		{
			name:         "ShadowTLS plugin creates ShadowTLS proxy",
			plugin:       "shadow-tls",
			expectedType: proto.ShadowTLS,
		},
		{
			name:         "no plugin creates Shadowsocks proxy",
			plugin:       "",
			expectedType: proto.Shadowsocks,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			member := &config.DoorProxyMember{
				ShowName: "Test",
				Type:     "shadowsocks",
				Config: map[string]interface{}{
					"server_host": "151.242.165.151",
					"server_port": 443,
					"method":      "chacha20-ietf-poly1305",
					"password":    "I8U3GD4pziEyIeQwTqd52CGLisU5boCwg6FBU9KpARs=",
				},
			}

			// Add plugin field if specified
			if tt.plugin != "" {
				member.Config.(map[string]interface{})["plugin"] = tt.plugin
				member.Config.(map[string]interface{})["plugin_opts"] = map[string]interface{}{
					"host":     "www.bing.com",
					"password": "I8U3GD4pziEyIeQwTqd52CGLisU5boCwg6FBU9KpARs=",
					"version":  3,
				}
			}

			p, err := createShadowsocksProxy(member)
			if err != nil {
				t.Fatalf("createShadowsocksProxy() failed: %v", err)
			}

			if p.Proto() != tt.expectedType {
				t.Errorf("expected proto %v, got %v", tt.expectedType, p.Proto())
			}
		})
	}
}

// TestCreateSocks5Proxy tests SOCKS5 proxy creation and validation
func TestCreateSocks5Proxy(t *testing.T) {
	tests := []struct {
		name         string
		member       *config.DoorProxyMember
		expectedType proto.Proto
		wantErr      bool
		errMsg       string
	}{
		{
			name: "create socks5 proxy without auth",
			member: &config.DoorProxyMember{
				ShowName: "SOCKS5 NoAuth",
				Type:     "socks5",
				Config: map[string]interface{}{
					"server_host": "127.0.0.1",
					"server_port": 1080,
				},
			},
			expectedType: proto.Socks5,
			wantErr:      false,
		},
		{
			name: "create socks5 proxy with auth",
			member: &config.DoorProxyMember{
				ShowName: "SOCKS5 Auth",
				Type:     "socks5",
				Config: map[string]interface{}{
					"server_host": "127.0.0.1",
					"server_port": 1080,
					"username":    "user",
					"password":    "pass",
				},
			},
			expectedType: proto.Socks5,
			wantErr:      false,
		},
		{
			name: "fail missing server_host",
			member: &config.DoorProxyMember{
				ShowName: "SOCKS5 Invalid",
				Type:     "socks5",
				Config: map[string]interface{}{
					"server_port": 1080,
				},
			},
			wantErr: true,
			errMsg:  "server_host is required",
		},
		{
			name: "fail username without password",
			member: &config.DoorProxyMember{
				ShowName: "SOCKS5 Invalid",
				Type:     "socks5",
				Config: map[string]interface{}{
					"server_host": "127.0.0.1",
					"server_port": 1080,
					"username":    "user",
				},
			},
			wantErr: true,
			errMsg:  "username and password must be provided together",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, err := createSocks5Proxy(tt.member)

			if (err != nil) != tt.wantErr {
				t.Errorf("createSocks5Proxy() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err != nil && tt.errMsg != "" {
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("createSocks5Proxy() error message = %v, should contain %v", err.Error(), tt.errMsg)
				}
			}

			if err == nil {
				if p == nil {
					t.Error("createSocks5Proxy() returned nil proxy without error")
					return
				}
				if p.Proto() != tt.expectedType {
					t.Errorf("createSocks5Proxy() proto = %v, want %v", p.Proto(), tt.expectedType)
				}
				if p.Addr() == "" {
					t.Error("createSocks5Proxy() returned proxy with empty address")
				}
			}
		})
	}
}

// TestDoorProxyRegistrationWithSocks5 tests registering door proxy members with SOCKS5
func TestDoorProxyRegistrationWithSocks5(t *testing.T) {
	doorCfg := &config.DoorProxyConfig{
		Type: "door",
		Members: []config.DoorProxyMember{
			{
				ShowName: "SOCKS5 Node",
				Type:     "socks5",
				Latency:  50,
				Status:   "success",
				Config: map[string]interface{}{
					"server_host": "127.0.0.1",
					"server_port": 1080,
				},
			},
		},
	}

	reg := &Registry{
		proxies: make(map[string]proxy.Proxy),
	}

	err := reg.RegisterDoorFromConfig(doorCfg)
	if err != nil {
		t.Fatalf("RegisterDoorFromConfig() failed: %v", err)
	}

	if reg.doorGroup == nil {
		t.Fatal("door group not registered")
	}

	if reg.doorGroup.Count() != 1 {
		t.Errorf("expected 1 door member, got %d", reg.doorGroup.Count())
	}

	s5Proxy, err := reg.GetDoor("SOCKS5 Node")
	if err != nil {
		t.Fatalf("GetDoor(SOCKS5 Node) failed: %v", err)
	}
	if s5Proxy.Proto() != proto.Socks5 {
		t.Errorf("expected Socks5 proxy, got %v", s5Proxy.Proto())
	}
}
