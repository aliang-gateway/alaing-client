package inbound

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"strings"
	"time"

	"nursor.org/nursorgate/common/logger"
	"nursor.org/nursorgate/outbound/proxy"
	"nursor.org/nursorgate/processor/config"
	"nursor.org/nursorgate/processor/dns"
)

// Global DNS resolver instance
var globalResolver dns.DNSResolverInterface

// InitGlobalResolver initializes the global DNS resolver
// Should be called in ApplyConfig after door and direct proxies are registered
func InitGlobalResolver(doorProxy, directProxy proxy.Proxy, cfg *config.Config) error {
	if cfg == nil || cfg.DNSPreResolution == nil || !cfg.DNSPreResolution.Enabled {
		logger.Info("[DNS] DNS resolution disabled in config")
		return nil
	}

	// Create hybrid DNS resolver using door and direct proxies
	hybridResolver := dns.NewHybridResolver(
		&dns.DNSConfig{
			Type:             dns.ResolverTypeHybrid,
			PrimaryDNS:       cfg.DNSPreResolution.GetPrimaryDNS(),
			FallbackDNS:      cfg.DNSPreResolution.GetFallbackDNS(),
			SystemDNSEnabled: cfg.DNSPreResolution.SystemDNSFallback,
			Timeout:          cfg.DNSPreResolution.GetTimeout(),
			MaxTTL:           cfg.DNSPreResolution.GetMaxCacheTTL(),
			CacheEnabled:     cfg.DNSPreResolution.CacheResults,
		},
		doorProxy,      // primary dialer (implements proxy.Dialer)
		directProxy,    // fallback dialer (implements proxy.Dialer)
	)

	globalResolver = hybridResolver
	logger.Info("[DNS] Global DNS resolver initialized successfully")
	return nil
}

// GetGlobalResolver returns the global DNS resolver instance
func GetGlobalResolver() dns.DNSResolverInterface {
	return globalResolver
}

// UpdateGlobalResolverWithDoorConfig updates the global resolver after door config changes
func UpdateGlobalResolverWithDoorConfig(doorConfig *config.DoorProxyConfig) {
	if globalResolver == nil {
		return
	}

	if doorConfig == nil || len(doorConfig.Members) == 0 {
		logger.Debug("[DNS] No door members, skipping resolver sync")
		return
	}

	logger.Info(fmt.Sprintf("[DNS] Door config updated with %d members", len(doorConfig.Members)))
}

// CleanupGlobalResolver cleans up the global DNS resolver
func CleanupGlobalResolver() {
	if globalResolver != nil {
		if clearable, ok := globalResolver.(interface{ ClearCache() }); ok {
			clearable.ClearCache()
		}
		globalResolver = nil
		logger.Info("[DNS] Global DNS resolver cleaned up")
	}
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
	if globalResolver != nil && isHostName(vlCfg.ServerHost) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		result, _ := globalResolver.ResolveIP(ctx, vlCfg.ServerHost)
		cancel()

		if result != nil && result.Success && len(result.IPs) > 0 {
			resolvedHost = result.IPs[0].String()
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
	if globalResolver != nil && isHostName(ssCfg.ServerHost) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		result, _ := globalResolver.ResolveIP(ctx, ssCfg.ServerHost)
		cancel()

		if result != nil && result.Success && len(result.IPs) > 0 {
			resolvedHost = result.IPs[0].String()
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

	// Perform batch DNS pre-resolution if resolver is available
	if globalResolver != nil && len(serverHosts) > 0 {
		logger.Info(fmt.Sprintf("[DNS] Batch DNS resolution ready for %d domains", len(serverHosts)))
		// Individual resolution happens in ConvertVLESS/ConvertShadowsocks using globalResolver
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

// isHostName checks if a string is a hostname (not an IP address)
func isHostName(addr string) bool {
	// Check if it's an IP address
	if ip := net.ParseIP(addr); ip != nil {
		return false
	}
	// If it contains colons, it might be IPv6 or an IP:port
	if strings.Contains(addr, ":") {
		if ip := net.ParseIP(strings.Split(addr, ":")[0]); ip != nil {
			return false
		}
	}
	// Otherwise, treat it as a hostname
	return true
}
