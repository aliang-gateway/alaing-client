package config

import (
	"fmt"
	"sync"
)

// ConfigStore stores proxy configurations (not instances)
type ConfigStore struct {
	mu      sync.RWMutex
	configs map[string]*ProxyConfig
}

var (
	globalConfigStore *ConfigStore
	configStoreOnce   sync.Once
)

// GetConfigStore returns the global config store singleton
func GetConfigStore() *ConfigStore {
	configStoreOnce.Do(func() {
		globalConfigStore = &ConfigStore{
			configs: make(map[string]*ProxyConfig),
		}
	})
	return globalConfigStore
}

// Set stores a proxy configuration
func (s *ConfigStore) Set(name string, cfg *ProxyConfig) error {
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
	if cfg.VLESS != nil {
		vlessCopy := *cfg.VLESS
		cfgCopy.VLESS = &vlessCopy
	}
	if cfg.Shadowsocks != nil {
		ssCopy := *cfg.Shadowsocks
		cfgCopy.Shadowsocks = &ssCopy
	}

	s.configs[name] = &cfgCopy
	return nil
}

// Get retrieves a proxy configuration
func (s *ConfigStore) Get(name string) (*ProxyConfig, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	cfg, exists := s.configs[name]
	if !exists {
		return nil, fmt.Errorf("config '%s' not found", name)
	}

	// Return a copy to prevent external modification
	cfgCopy := *cfg
	if cfg.VLESS != nil {
		vlessCopy := *cfg.VLESS
		cfgCopy.VLESS = &vlessCopy
	}
	if cfg.Shadowsocks != nil {
		ssCopy := *cfg.Shadowsocks
		cfgCopy.Shadowsocks = &ssCopy
	}

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

	s.configs = make(map[string]*ProxyConfig)
}

// GetAll returns all configs (for debugging/listing)
func (s *ConfigStore) GetAll() map[string]*ProxyConfig {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make(map[string]*ProxyConfig, len(s.configs))
	for name, cfg := range s.configs {
		// Deep copy
		cfgCopy := *cfg
		if cfg.VLESS != nil {
			vlessCopy := *cfg.VLESS
			cfgCopy.VLESS = &vlessCopy
		}
		if cfg.Shadowsocks != nil {
			ssCopy := *cfg.Shadowsocks
			cfgCopy.Shadowsocks = &ssCopy
		}
		result[name] = &cfgCopy
	}
	return result
}
