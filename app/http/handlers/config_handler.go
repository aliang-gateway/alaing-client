package handlers

import (
	"net/http"

	"nursor.org/nursorgate/app/http/common"
	"nursor.org/nursorgate/app/http/repositories"
)

// ConfigHandler handles HTTP requests for configuration operations
type ConfigHandler struct {
	configRepository *repositories.ConfigRepositoryImpl
}

// NewConfigHandler creates a new config handler instance with dependency injection
func NewConfigHandler(configRepository *repositories.ConfigRepositoryImpl) *ConfigHandler {
	return &ConfigHandler{
		configRepository: configRepository,
	}
}

// HandleConfigGet handles GET /api/config/get
func (ch *ConfigHandler) HandleConfigGet(w http.ResponseWriter, r *http.Request) {
	name := common.GetQueryParamString(r, "name", "")
	if name == "" {
		common.ErrorBadRequest(w, "name parameter is required", nil)
		return
	}

	cfg, err := ch.configRepository.GetConfig(name)
	if err != nil {
		common.ErrorNotFound(w, err.Error())
		return
	}

	common.Success(w, cfg)
}

// HandleConfigList handles GET /api/config/list
func (ch *ConfigHandler) HandleConfigList(w http.ResponseWriter, r *http.Request) {
	configs := ch.configRepository.ListConfigs()

	common.Success(w, map[string]interface{}{
		"configs": configs,
		"count":   len(configs.(map[string]interface{})),
	})
}
