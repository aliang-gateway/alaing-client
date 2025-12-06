package services

import (
	proxyConfig "nursor.org/nursorgate/processor/config"
)

// ConfigService handles configuration operations
type ConfigService struct{}

// NewConfigService creates a new config service instance
func NewConfigService() *ConfigService {
	return &ConfigService{}
}

// GetConfig retrieves stored configuration by name
func (cs *ConfigService) GetConfig(name string) (interface{}, error) {
	return proxyConfig.GetConfigStore().Get(name)
}

// ListConfigs lists all stored configurations
func (cs *ConfigService) ListConfigs() interface{} {
	store := proxyConfig.GetConfigStore()
	return store.GetAll()
}
