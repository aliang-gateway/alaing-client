package config

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"sync"

	"nursor.org/nursorgate/common/logger"
)

var (
	globalConfig *Config
	configMutex  sync.RWMutex
)

// SetGlobalConfig sets the global configuration instance
func SetGlobalConfig(cfg *Config) {
	configMutex.Lock()
	defer configMutex.Unlock()
	globalConfig = cfg
}

// GetGlobalConfig returns the global configuration instance
func GetGlobalConfig() *Config {
	configMutex.RLock()
	defer configMutex.RUnlock()
	return globalConfig
}

// UpdateGlobalDoorProxyConfig updates the global configuration's DoorProxy field
// This is the single entry point for updating door proxy configuration, ensuring consistency
func UpdateGlobalDoorProxyConfig(doorConfig *DoorProxyConfig) error {
	cfg := GetGlobalConfig()
	if cfg == nil {
		return fmt.Errorf("global config not initialized")
	}

	configMutex.Lock()
	defer configMutex.Unlock()

	cfg.DoorProxy = doorConfig

	memberCount := 0
	if doorConfig != nil {
		memberCount = len(doorConfig.Members)
	}

	logger.Info(fmt.Sprintf("Updated global DoorProxy config with %d members", memberCount))

	return nil
}

// SaveConfigToFile persists the global configuration to a JSON file
func SaveConfigToFile(cfg *Config, filePath string) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := ioutil.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	logger.Info(fmt.Sprintf("Configuration saved to: %s", filePath))
	return nil
}
