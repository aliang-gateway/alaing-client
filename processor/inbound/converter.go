package inbound

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"nursor.org/nursorgate/common/logger"
	"nursor.org/nursorgate/processor/config"
	"nursor.org/nursorgate/processor/dns"
)

// Global DNS preloader instance
var globalPreloader *dns.Preloader

// InitDNSPreloader initializes the global DNS preloader
func InitDNSPreloader(resolver dns.DNSResolverInterface, preloadConfig *dns.PreloadConfig) {
	globalPreloader = dns.NewPreloader(resolver, preloadConfig)
	logger.Info("[DNS] Global preloader initialized")
}

// GetDNSPreloader returns the global DNS preloader instance
func GetDNSPreloader() *dns.Preloader {
	return globalPreloader
}

// ShutdownDNSPreloader shuts down the global DNS preloader
func ShutdownDNSPreloader() {
	globalPreloader = nil
	logger.Info("[DNS] Global preloader shut down")
}

// ConvertToProxyConfig converts InboundInfo to DoorProxyMember configuration
func ConvertToProxyConfig(info InboundInfo) (*config.DoorProxyMember, error) {
	if info.InboundType == "" {
		return nil, fmt.Errorf("inbound type is empty")
	}

	if info.Config == nil {
		return nil, fmt.Errorf("inbound config is empty")
	}

	switch strings.ToLower(info.InboundType) {
	case "vless":
		return ConvertVLESS(info.Config, info.Tag)
	case "shadowsocks", "ss":
		return ConvertShadowsocks(info.Config, info.Tag)
	default:
		return nil, fmt.Errorf("unsupported inbound type: %s", info.InboundType)
	}
}

// ConvertVLESS converts raw VLESS config to DoorProxyMember
func ConvertVLESS(rawConfig interface{}, tag string) (*config.DoorProxyMember, error) {
	// Convert to map then to struct for flexible JSON parsing
	configJSON, err := json.Marshal(rawConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal vless config: %w", err)
	}

	var vlCfg VLESSInboundConfig
	if err := json.Unmarshal(configJSON, &vlCfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal vless config: %w", err)
	}

	// Validate required fields
	if err := validateVLESS(&vlCfg); err != nil {
		return nil, err
	}

	// DNS pre-resolution for server host
	resolvedHost := vlCfg.ServerHost
	if globalPreloader != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		resolvedHost = globalPreloader.PreloadServerHost(ctx, vlCfg.ServerHost)
		if resolvedHost != vlCfg.ServerHost {
			logger.Info(fmt.Sprintf("[DNS] VLESS %s: %s -> %s", tag, vlCfg.ServerHost, resolvedHost))
		}
	}

	vlessConfig := &config.VLESSConfig{
		Server:         resolvedHost,
		ServerPort:     uint16(vlCfg.ServerPort),
		UUID:           vlCfg.VlessUUID,
		Flow:           vlCfg.VlessFlow,
		TLSEnabled:     vlCfg.TLSEnabled,
		RealityEnabled: vlCfg.RealityEnabled, // Use value from backend
		SNI:            vlCfg.TLSServerName,
		PublicKey:      vlCfg.PublicKey,
		ShortIDs:       vlCfg.ShortIDs,
	}

	member := &config.DoorProxyMember{
		ShowName: tag,
		Type:     "vless",
		VLESS:    vlessConfig,
	}

	logger.Info(fmt.Sprintf("Converted VLESS inbound: %s -> %s:%d", tag, resolvedHost, vlCfg.ServerPort))
	return member, nil
}

// validateVLESS validates required VLESS configuration fields
func validateVLESS(cfg *VLESSInboundConfig) error {
	if cfg.ServerHost == "" {
		return fmt.Errorf("vless server host required")
	}
	if cfg.ServerPort == 0 {
		return fmt.Errorf("vless server port required")
	}
	if cfg.VlessUUID == "" {
		return fmt.Errorf("vless uuid required")
	}
	return nil
}

// ConvertShadowsocks converts raw Shadowsocks config to DoorProxyMember
func ConvertShadowsocks(rawConfig interface{}, tag string) (*config.DoorProxyMember, error) {
	// Convert to map then to struct for flexible JSON parsing
	configJSON, err := json.Marshal(rawConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal shadowsocks config: %w", err)
	}

	var ssCfg SSInboundConfig
	if err := json.Unmarshal(configJSON, &ssCfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal shadowsocks config: %w", err)
	}

	// Validate required fields
	if err := validateShadowsocks(&ssCfg); err != nil {
		return nil, err
	}

	// DNS pre-resolution for server host
	resolvedHost := ssCfg.ServerHost
	if globalPreloader != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		resolvedHost = globalPreloader.PreloadServerHost(ctx, ssCfg.ServerHost)
		if resolvedHost != ssCfg.ServerHost {
			logger.Info(fmt.Sprintf("[DNS] Shadowsocks %s: %s -> %s", tag, ssCfg.ServerHost, resolvedHost))
		}
	}

	ssConfig := &config.ShadowsocksConfig{
		Server:     resolvedHost,
		ServerPort: uint16(ssCfg.ServerPort),
		Method:     ssCfg.Method,
		Password:   ssCfg.SSPassword,
		Username:   ssCfg.ProxyUser,
	}

	member := &config.DoorProxyMember{
		ShowName:    tag,
		Type:        "shadowsocks",
		Shadowsocks: ssConfig,
	}

	logger.Info(fmt.Sprintf("Converted Shadowsocks inbound: %s -> %s:%d", tag, resolvedHost, ssCfg.ServerPort))
	return member, nil
}

// validateShadowsocks validates required Shadowsocks configuration fields
func validateShadowsocks(cfg *SSInboundConfig) error {
	if cfg.ServerHost == "" || cfg.ServerPort == 0 {
		return fmt.Errorf("shadowsocks server host and port required")
	}
	if cfg.Method == "" || cfg.SSPassword == "" {
		return fmt.Errorf("shadowsocks method and password required")
	}
	return nil
}

// BatchConvertToProxyConfigs converts multiple InboundInfo objects with batch DNS pre-resolution
func BatchConvertToProxyConfigs(inbounds []InboundInfo) ([]*config.DoorProxyMember, error) {
	if len(inbounds) == 0 {
		return []*config.DoorProxyMember{}, nil
	}

	// Collect all unique server hosts for batch pre-resolution
	serverHosts := make(map[string]bool)
	for _, inbound := range inbounds {
		switch strings.ToLower(inbound.InboundType) {
		case "vless":
			var configJSON []byte
			var err error
			configJSON, err = json.Marshal(inbound.Config)
			if err != nil {
				continue
			}
			var vlCfg VLESSInboundConfig
			if err := json.Unmarshal(configJSON, &vlCfg); err == nil {
				serverHosts[vlCfg.ServerHost] = true
			}
		case "shadowsocks", "ss":
			var configJSON []byte
			var err error
			configJSON, err = json.Marshal(inbound.Config)
			if err != nil {
				continue
			}
			var ssCfg SSInboundConfig
			if err := json.Unmarshal(configJSON, &ssCfg); err == nil {
				serverHosts[ssCfg.ServerHost] = true
			}
		}
	}

	// Perform batch DNS pre-resolution if preloader is available
	if globalPreloader != nil && len(serverHosts) > 0 {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		var domains []string
		for host := range serverHosts {
			domains = append(domains, host)
		}

		_ = globalPreloader.PreloadDomains(ctx, domains) // Preload results are cached internally
		logger.Info(fmt.Sprintf("[DNS] Batch pre-resolution completed for %d domains", len(domains)))
	}

	// Convert each inbound with pre-resolved hosts
	var members []*config.DoorProxyMember
	for _, inbound := range inbounds {
		member, err := ConvertToProxyConfig(inbound)
		if err != nil {
			logger.Warn(fmt.Sprintf("Failed to convert inbound %s: %v", inbound.Tag, err))
			continue
		}
		members = append(members, member)
	}

	return members, nil
}
