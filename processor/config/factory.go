package config

import (
	"fmt"
	"math/rand"
	"strings"

	"nursor.org/nursorgate/outbound/proxy"
	"nursor.org/nursorgate/outbound/proxy/direct"
	"nursor.org/nursorgate/outbound/proxy/shadowsocks"
	"nursor.org/nursorgate/outbound/proxy/vless"
)

// CreateProxyFromConfig creates a proxy instance from configuration
func CreateProxyFromConfig(cfg *BaseProxyConfig) (proxy.Proxy, error) {
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	switch cfg.Type {
	case "direct":
		return direct.NewDirect(), nil
	case "nonelane":
		// Nonelane proxy will be handled separately in registry
		return nil, fmt.Errorf("nonelane proxy should be created by registry")
	case "door":
		// Door proxy will be handled separately in registry
		return nil, fmt.Errorf("door proxy should be created by registry")
	default:
		return nil, fmt.Errorf("unsupported proxy type: %s", cfg.Type)
	}
}

// createVLESSProxy creates VLESS proxy instance
func createVLESSProxy(cfg *VLESSConfig) (proxy.Proxy, error) {
	// Handle REALITY
	if cfg.RealityEnabled {
		shortID := cfg.ShortID
		if shortID == "" && cfg.ShortIDList != "" {
			// Random selection from ShortIDList
			shortIDArray := strings.Split(cfg.ShortIDList, ",")
			if len(shortIDArray) > 0 {
				shortID = strings.TrimSpace(shortIDArray[rand.Intn(len(shortIDArray))])
			}
		}
		return vless.NewVLESSWithReality(
			cfg.Server,
			cfg.UUID,
			cfg.SNI,
			cfg.PublicKey,
		)
	}

	// Handle TLS
	if cfg.TLSEnabled {
		if cfg.Flow != "" {
			// VLESS with Vision flow
			return vless.NewVLESSWithVision(cfg.Server, cfg.UUID, cfg.SNI)
		}
		// VLESS with TLS only
		return vless.NewVLESSWithTLS(cfg.Server, cfg.UUID, cfg.SNI)
	}

	// Basic VLESS
	return vless.NewVLESS(cfg.Server, cfg.UUID)
}

// createShadowsocksProxy creates Shadowsocks proxy instance
func createShadowsocksProxy(cfg *ShadowsocksConfig) (proxy.Proxy, error) {
	return shadowsocks.NewShadowsocks(
		cfg.Server,
		cfg.Method,
		cfg.Password,
		cfg.ObfsMode,
		cfg.ObfsHost,
	)
}
