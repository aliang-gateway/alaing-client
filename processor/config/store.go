package config

import (
	"fmt"
	"sync"

	"nursor.org/nursorgate/common/logger"
)

// ConfigStore stores proxy configurations (not instances)
type ConfigStore struct {
	mu      sync.RWMutex
	configs map[string]*BaseProxyConfig
}

var (
	globalConfigStore *ConfigStore
	configStoreOnce   sync.Once
)

// GetConfigStore returns the global config store singleton
func GetConfigStore() *ConfigStore {
	configStoreOnce.Do(func() {
		globalConfigStore = &ConfigStore{
			configs: make(map[string]*BaseProxyConfig),
		}
	})
	return globalConfigStore
}

// Set stores a proxy configuration
func (s *ConfigStore) Set(name string, cfg *BaseProxyConfig) error {
	if name == "" {
		return fmt.Errorf("proxy name cannot be empty")
	}
	if cfg == nil {
		return fmt.Errorf("config cannot be nil")
	}

	// Validate config before storing
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("invalid config: %w", err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Create a deep copy to prevent external modification
	cfgCopy := *cfg

	s.configs[name] = &cfgCopy
	return nil
}

// Get retrieves a proxy configuration
func (s *ConfigStore) Get(name string) (*BaseProxyConfig, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	cfg, exists := s.configs[name]
	if !exists {
		return nil, fmt.Errorf("config '%s' not found", name)
	}

	// Return a copy to prevent external modification
	cfgCopy := *cfg

	return &cfgCopy, nil
}

// List returns all config names
func (s *ConfigStore) List() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	names := make([]string, 0, len(s.configs))
	for name := range s.configs {
		names = append(names, name)
	}
	return names
}

// Delete removes a config
func (s *ConfigStore) Delete(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.configs[name]; !exists {
		return fmt.Errorf("config '%s' not found", name)
	}

	delete(s.configs, name)
	return nil
}

// Clear removes all configs
func (s *ConfigStore) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.configs = make(map[string]*BaseProxyConfig)
}

// GetAll returns all configs (for debugging/listing)
func (s *ConfigStore) GetAll() map[string]*BaseProxyConfig {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make(map[string]*BaseProxyConfig, len(s.configs))
	for name, cfg := range s.configs {
		// Deep copy
		cfgCopy := *cfg
		result[name] = &cfgCopy
	}
	return result
}

// GetDoorProxyMembers returns all Door proxy members from global configuration
// This is the single entry point for retrieving door proxy members from config (not registry)
func GetDoorProxyMembers() ([]DoorProxyMember, error) {
	cfg := GetGlobalConfig()
	if cfg == nil || cfg.DoorProxy == nil {
		return nil, fmt.Errorf("door proxy not configured")
	}

	// Return a copy to prevent external modification
	members := make([]DoorProxyMember, len(cfg.DoorProxy.Members))
	copy(members, cfg.DoorProxy.Members)

	logger.Debug(fmt.Sprintf("Retrieved %d door proxy members from config", len(members)))
	return members, nil
}

// GetDoorProxyMember retrieves a specific Door proxy member by ShowName from global configuration
func GetDoorProxyMember(showName string) (*DoorProxyMember, error) {
	members, err := GetDoorProxyMembers()
	if err != nil {
		return nil, err
	}

	for i := range members {
		if members[i].ShowName == showName {
			return &members[i], nil
		}
	}

	return nil, fmt.Errorf("door proxy member '%s' not found in config", showName)
}

// GetDoorProxyMemberCount returns the number of configured Door proxy members
func GetDoorProxyMemberCount() int {
	cfg := GetGlobalConfig()
	if cfg == nil || cfg.DoorProxy == nil {
		return 0
	}

	count := len(cfg.DoorProxy.Members)
	logger.Debug(fmt.Sprintf("Door proxy member count: %d", count))
	return count
}

// GetProxyConfigInfo retrieves complete proxy configuration information
// Supports both regular proxies and door proxy members (format: "door:ShowName")
// Returns a map containing all configuration details including type, protocol, and config-specific fields
func GetProxyConfigInfo(proxyName string) (map[string]interface{}, error) {
	// Check if it's a door proxy member (format: "door:ShowName")
	if len(proxyName) > 5 && proxyName[:5] == "door:" {
		showName := proxyName[5:]

		// Get the door proxy member from configuration
		member, err := GetDoorProxyMember(showName)
		if err != nil {
			return nil, fmt.Errorf("door proxy member not found: %w", err)
		}

		// Build response with complete member information
		result := map[string]interface{}{
			"name":      proxyName,
			"show_name": member.ShowName,
			"type":      member.Type,
			"latency":   member.Latency,
		}

		// Add protocol-specific configuration details based on type
		switch member.Type {
		case "vless":
			if vlssCfg, err := member.GetVLESSConfig(); err == nil {
				result["config"] = vlssCfg
			} else {
				result["config_error"] = fmt.Sprintf("failed to extract VLESS config: %v", err)
			}

		case "shadowsocks", "ss":
			if ssCfg, err := member.GetShadowsocksConfig(); err == nil {
				result["config"] = ssCfg
			} else {
				result["config_error"] = fmt.Sprintf("failed to extract Shadowsocks config: %v", err)
			}

		case "socks5", "socks":
			if s5Cfg, err := member.GetSocks5Config(); err == nil {
				result["config"] = s5Cfg
			} else {
				result["config_error"] = fmt.Sprintf("failed to extract SOCKS5 config: %v", err)
			}

		default:
			result["config"] = member.Config
		}

		result["source"] = "configuration"
		return result, nil
	}

	// For regular proxies, check ConfigStore
	store := GetConfigStore()
	cfg, err := store.Get(proxyName)
	if err != nil {
		return nil, fmt.Errorf("proxy config not found: %w", err)
	}

	result := map[string]interface{}{
		"name":   proxyName,
		"type":   cfg.Type,
		"config": cfg,
		"source": "configuration",
	}

	return result, nil
}
