package repositories

import (
	"nursor.org/nursorgate/app/http/services"
)

// ConfigRepositoryImpl provides access to configuration functionality
type ConfigRepositoryImpl struct {
	configService *services.ConfigService
}

// NewConfigRepository creates a new config repository instance
func NewConfigRepository() *ConfigRepositoryImpl {
	return &ConfigRepositoryImpl{
		configService: services.NewConfigService(),
	}
}

// GetConfig retrieves stored configuration by name
func (cr *ConfigRepositoryImpl) GetConfig(name string) (interface{}, error) {
	return cr.configService.GetConfig(name)
}

// ListConfigs lists all stored configurations
func (cr *ConfigRepositoryImpl) ListConfigs() interface{} {
	return cr.configService.ListConfigs()
}
