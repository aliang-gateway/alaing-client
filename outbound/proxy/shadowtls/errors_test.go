package shadowtls

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"aliang.one/nursorgate/processor/config"
)

// TestConfigError tests the ConfigError type
func TestConfigError(t *testing.T) {
	tests := []struct {
		name         string
		err          error
		wantContains []string
		wantUnwrap   error
	}{
		{
			name:         "nil config error",
			err:          newConfigError("config", "configuration is nil", ErrNilConfig),
			wantContains: []string{"shadowtls config error", "[config]", "configuration is nil"},
			wantUnwrap:   ErrNilConfig,
		},
		{
			name:         "invalid plugin error",
			err:          newConfigError("plugin", "plugin must be 'shadow-tls', got 'v2ray'", ErrInvalidPlugin),
			wantContains: []string{"shadowtls config error", "[plugin]", "shadow-tls", "v2ray"},
			wantUnwrap:   ErrInvalidPlugin,
		},
		{
			name:         "missing plugin_opts error",
			err:          newConfigError("plugin_opts", "required when using shadow-tls plugin", ErrMissingPluginOpts),
			wantContains: []string{"shadowtls config error", "[plugin_opts]", "required"},
			wantUnwrap:   ErrMissingPluginOpts,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errMsg := tt.err.Error()

			// Check that all expected strings are in the error message
			for _, want := range tt.wantContains {
				if !strings.Contains(errMsg, want) {
					t.Errorf("error message %q should contain %q", errMsg, want)
				}
			}

			// Check unwrap
			if tt.wantUnwrap != nil {
				var configErr *ConfigError
				if errors.As(tt.err, &configErr) {
					unwrapped := errors.Unwrap(configErr)
					if !errors.Is(unwrapped, tt.wantUnwrap) {
						t.Errorf("Unwrap() = %v, want %v", unwrapped, tt.wantUnwrap)
					}
				} else {
					t.Error("error is not a ConfigError")
				}
			}
		})
	}
}

// TestConnectionError tests the ConnectionError type
func TestConnectionError(t *testing.T) {
	tests := []struct {
		name         string
		err          error
		wantContains []string
	}{
		{
			name:         "tls_handshake stage error",
			err:          newConnectionError("tls_handshake", "151.242.165.151:443", ErrTLSHandshakeFailed),
			wantContains: []string{"shadowtls connection error", "stage=tls_handshake", "addr=151.242.165.151:443"},
		},
		{
			name:         "auth stage error",
			err:          newConnectionError("auth", "151.242.165.151:443", ErrAuthFailed),
			wantContains: []string{"shadowtls connection error", "stage=auth", "addr=151.242.165.151:443"},
		},
		{
			name:         "cipher_init stage error",
			err:          newConnectionError("cipher_init", "151.242.165.151:443", ErrCipherInitFailed),
			wantContains: []string{"shadowtls connection error", "stage=cipher_init", "addr=151.242.165.151:443"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errMsg := tt.err.Error()

			for _, want := range tt.wantContains {
				if !strings.Contains(errMsg, want) {
					t.Errorf("error message %q should contain %q", errMsg, want)
				}
			}
		})
	}
}

// TestTLSError tests the TLSError type
func TestTLSError(t *testing.T) {
	tests := []struct {
		name         string
		err          error
		wantContains []string
	}{
		{
			name:         "tcp_dial stage error",
			err:          newTLSError("tcp_dial", "www.bing.com", fmt.Errorf("connection refused")),
			wantContains: []string{"shadowtls TLS error", "stage=tcp_dial", "host=www.bing.com", "connection refused"},
		},
		{
			name:         "handshake stage error",
			err:          newTLSError("handshake", "www.bing.com", fmt.Errorf("certificate verify failed")),
			wantContains: []string{"shadowtls TLS error", "stage=handshake", "host=www.bing.com", "certificate verify failed"},
		},
		{
			name:         "cert_verify stage error",
			err:          newTLSError("cert_verify", "www.bing.com", ErrTLSCertInvalid),
			wantContains: []string{"shadowtls TLS error", "stage=cert_verify", "host=www.bing.com"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errMsg := tt.err.Error()

			for _, want := range tt.wantContains {
				if !strings.Contains(errMsg, want) {
					t.Errorf("error message %q should contain %q", errMsg, want)
				}
			}
		})
	}
}

// TestAuthError tests the AuthError type
func TestAuthError(t *testing.T) {
	tests := []struct {
		name         string
		version      int
		err          error
		wantContains []string
	}{
		{
			name:         "version 3 not implemented",
			version:      3,
			err:          newAuthError(3, ErrAuthNotImplemented),
			wantContains: []string{"shadowtls auth error", "version=3", "not yet implemented"},
		},
		{
			name:         "version 2 auth failed",
			version:      2,
			err:          newAuthError(2, ErrAuthFailed),
			wantContains: []string{"shadowtls auth error", "version=2", "authentication failed"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errMsg := tt.err.Error()

			for _, want := range tt.wantContains {
				if !strings.Contains(errMsg, want) {
					t.Errorf("error message %q should contain %q", errMsg, want)
				}
			}
		})
	}
}

// TestCipherError tests the CipherError type
func TestCipherError(t *testing.T) {
	tests := []struct {
		name         string
		err          error
		wantContains []string
	}{
		{
			name:         "init operation error",
			err:          newCipherError("chacha20-ietf-poly1305", "init", fmt.Errorf("cipher not supported")),
			wantContains: []string{"shadowtls cipher error", "method=chacha20-ietf-poly1305", "op=init", "cipher not supported"},
		},
		{
			name:         "encrypt operation error",
			err:          newCipherError("aes-256-gcm", "encrypt", ErrEncryptionFailed),
			wantContains: []string{"shadowtls cipher error", "method=aes-256-gcm", "op=encrypt"},
		},
		{
			name:         "decrypt operation error",
			err:          newCipherError("aes-128-gcm", "decrypt", ErrDecryptionFailed),
			wantContains: []string{"shadowtls cipher error", "method=aes-128-gcm", "op=decrypt"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errMsg := tt.err.Error()

			for _, want := range tt.wantContains {
				if !strings.Contains(errMsg, want) {
					t.Errorf("error message %q should contain %q", errMsg, want)
				}
			}
		})
	}
}

// TestNewErrorsWithRealConfig tests error messages with real proxy creation
func TestNewErrorsWithRealConfig(t *testing.T) {
	tests := []struct {
		name         string
		cfg          *config.ShadowsocksConfig
		wantContains []string
	}{
		{
			name:         "nil config",
			cfg:          nil,
			wantContains: []string{"shadowtls config error", "[config]", "configuration is nil"},
		},
		{
			name: "wrong plugin",
			cfg: &config.ShadowsocksConfig{
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
			// Plugin validation happens during config.Validate(), which is wrapped in a broader config error
			wantContains: []string{"shadowtls config error", "[config]", "unsupported plugin", "v2ray-plugin"},
		},
		{
			name: "unsupported cipher",
			cfg: &config.ShadowsocksConfig{
				Server:     "151.242.165.151",
				ServerPort: 443,
				Method:     "unsupported-cipher-999",
				Password:   "I8U3GD4pziEyIeQwTqd52CGLisU5boCwg6FBU9KpARs=",
				Plugin:     "shadow-tls",
				PluginOpts: &config.ShadowTLSPluginOpts{
					Host:     "www.bing.com",
					Password: "I8U3GD4pziEyIeQwTqd52CGLisU5boCwg6FBU9KpARs=",
					Version:  3,
				},
			},
			wantContains: []string{"shadowtls cipher error", "method=unsupported-cipher-999", "op=init"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := New(tt.cfg)
			if err == nil {
				t.Fatal("expected error, got nil")
			}

			errMsg := err.Error()
			for _, want := range tt.wantContains {
				if !strings.Contains(errMsg, want) {
					t.Errorf("error message %q should contain %q", errMsg, want)
				}
			}
		})
	}
}

// TestErrorUnwrapping tests that errors can be properly unwrapped
func TestErrorUnwrapping(t *testing.T) {
	// Test ConfigError unwrapping
	configErr := newConfigError("config", "test", ErrNilConfig)
	if !errors.Is(configErr, ErrNilConfig) {
		t.Error("ConfigError should unwrap to ErrNilConfig")
	}

	// Test ConnectionError unwrapping
	connErr := newConnectionError("tls_handshake", "addr", ErrTLSHandshakeFailed)
	if !errors.Is(connErr, ErrTLSHandshakeFailed) {
		t.Error("ConnectionError should unwrap to ErrTLSHandshakeFailed")
	}

	// Test TLSError unwrapping
	tlsErr := newTLSError("handshake", "host", ErrTLSHandshakeFailed)
	if !errors.Is(tlsErr, ErrTLSHandshakeFailed) {
		t.Error("TLSError should unwrap to ErrTLSHandshakeFailed")
	}

	// Test AuthError unwrapping
	authErr := newAuthError(3, ErrAuthNotImplemented)
	if !errors.Is(authErr, ErrAuthNotImplemented) {
		t.Error("AuthError should unwrap to ErrAuthNotImplemented")
	}

	// Test CipherError unwrapping
	cipherErr := newCipherError("method", "init", ErrCipherInitFailed)
	if !errors.Is(cipherErr, ErrCipherInitFailed) {
		t.Error("CipherError should unwrap to ErrCipherInitFailed")
	}
}
