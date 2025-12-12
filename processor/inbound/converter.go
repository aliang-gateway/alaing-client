package inbound

import (
	"encoding/json"
	"fmt"
	"strings"

	"nursor.org/nursorgate/common/logger"
	"nursor.org/nursorgate/processor/config"
)

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

	vlessConfig := &config.VLESSConfig{
		Server:         vlCfg.ServerHost,
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

	logger.Info(fmt.Sprintf("Converted VLESS inbound: %s -> %s:%d", tag, vlCfg.ServerHost, vlCfg.ServerPort))
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

	ssConfig := &config.ShadowsocksConfig{
		Server:     ssCfg.ServerHost,
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

	logger.Info(fmt.Sprintf("Converted Shadowsocks inbound: %s -> %s:%d", tag, ssCfg.ServerHost, ssCfg.ServerPort))
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
