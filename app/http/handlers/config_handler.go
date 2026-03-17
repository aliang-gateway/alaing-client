package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"nursor.org/nursorgate/app/http/common"
	"nursor.org/nursorgate/common/logger"
	"nursor.org/nursorgate/common/model"
)

var (
	routingConfigStoreMu sync.RWMutex
	routingConfigStore   = model.NewRoutingRulesConfig()
)

// ConfigHandler provides a local (non-Nacos) routing-config compatibility API.
type ConfigHandler struct{}

func NewConfigHandler() *ConfigHandler {
	return &ConfigHandler{}
}

func (h *ConfigHandler) HandleRoutingConfig(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.handleGetRoutingConfig(w)
	case http.MethodPost:
		h.handleUpdateRoutingConfig(w, r)
	default:
		common.Error(w, http.StatusMethodNotAllowed, "Method not allowed", nil)
	}
}

func (h *ConfigHandler) HandleToggleRuleStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		common.Error(w, http.StatusMethodNotAllowed, "Method not allowed", nil)
		return
	}

	pathParts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/config/routing/rules/"), "/")
	if len(pathParts) < 2 {
		common.ErrorBadRequest(w, "Invalid URL format", nil)
		return
	}
	ruleID := pathParts[0]

	var req struct {
		Enabled bool `json:"enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		common.ErrorBadRequest(w, "Invalid JSON format", map[string]interface{}{"error": err.Error()})
		return
	}

	routingConfigStoreMu.Lock()
	defer routingConfigStoreMu.Unlock()

	cfg := cloneRoutingConfigLocked()
	ruleFound := false
	ruleSets := []*model.RoutingRuleSet{&cfg.ToSocks, &cfg.BlackList, &cfg.Aliang}
	for _, rs := range ruleSets {
		for i := range rs.Rules {
			if rs.Rules[i].ID == ruleID {
				rs.Rules[i].Enabled = req.Enabled
				rs.Rules[i].UpdatedAt = time.Now()
				ruleFound = true
				break
			}
		}
		if ruleFound {
			break
		}
	}

	if !ruleFound {
		common.ErrorNotFound(w, fmt.Sprintf("Rule with id '%s' not found", ruleID))
		return
	}

	cfg.UpdatedAt = time.Now()
	routingConfigStore = cfg
	common.Success(w, map[string]interface{}{
		"message": "Rule toggled successfully",
		"rule_id": ruleID,
		"enabled": req.Enabled,
	})
}

func (h *ConfigHandler) HandleAutoUpdateStatus(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		routingConfigStoreMu.RLock()
		defer routingConfigStoreMu.RUnlock()
		common.Success(w, map[string]interface{}{
			"auto_update": false,
			"source":      "local",
			"updated_at":  routingConfigStore.UpdatedAt,
		})
	case http.MethodPut:
		// Compatibility endpoint: auto-update is no longer supported in simplified mode.
		common.Success(w, map[string]interface{}{
			"message":     "Auto-update is disabled in simplified mode",
			"auto_update": false,
		})
	default:
		common.Error(w, http.StatusMethodNotAllowed, "Method not allowed", nil)
	}
}

func (h *ConfigHandler) handleGetRoutingConfig(w http.ResponseWriter) {
	routingConfigStoreMu.RLock()
	defer routingConfigStoreMu.RUnlock()
	common.Success(w, routingConfigStore)
}

func (h *ConfigHandler) handleUpdateRoutingConfig(w http.ResponseWriter, r *http.Request) {
	var cfg model.RoutingRulesConfig
	if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
		common.ErrorBadRequest(w, "Invalid JSON format", map[string]interface{}{"error": err.Error()})
		return
	}
	if err := cfg.Validate(); err != nil {
		common.ErrorBadRequest(w, "Configuration validation failed", map[string]interface{}{"error": err.Error()})
		return
	}

	routingConfigStoreMu.Lock()
	defer routingConfigStoreMu.Unlock()
	cfg.UpdatedAt = time.Now()
	routingConfigStore = &cfg
	logger.Info("Routing configuration updated (local store)")
	common.Success(w, map[string]interface{}{
		"message": "Configuration updated successfully",
		"source":  "local",
	})
}

func cloneRoutingConfigLocked() *model.RoutingRulesConfig {
	data, err := routingConfigStore.ToJSON()
	if err != nil {
		return model.NewRoutingRulesConfig()
	}
	cfg, err := model.NewRoutingRulesConfigFromJSON(data)
	if err != nil {
		return model.NewRoutingRulesConfig()
	}
	return cfg
}
