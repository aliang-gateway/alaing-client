package config

import "sync"

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
