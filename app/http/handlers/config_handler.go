package handlers

import (
	"fmt"
	"io"
	"net/http"
	"strings"

	"nursor.org/nursorgate/app/http/common"
	"nursor.org/nursorgate/common/logger"
	"nursor.org/nursorgate/processor/config"
	"nursor.org/nursorgate/processor/routing"
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
	common.ErrorBadRequest(w, "Legacy mutable rule toggle write path is removed; update canonical routing config via /api/config/routing", map[string]interface{}{"rule_id": ruleID})
}

func (h *ConfigHandler) HandleAutoUpdateStatus(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		common.Success(w, map[string]interface{}{
			"auto_update": false,
			"source":      "canonical",
			"updated_at":  "",
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
	common.Success(w, activeCanonicalRoutingConfig())
}

func (h *ConfigHandler) handleUpdateRoutingConfig(w http.ResponseWriter, r *http.Request) {
	raw, err := ioReadAll(r)
	if err != nil {
		common.ErrorBadRequest(w, "Invalid JSON format", map[string]interface{}{"error": err.Error()})
		return
	}

	applyResult, err := config.GetRoutingApplyStore().Apply(raw, func(canonical *config.CanonicalRoutingSchema) (any, error) {
		return routing.CompileRuntimeSnapshot(canonical)
	})
	if err != nil {
		common.ErrorBadRequest(w, "Configuration validation failed", map[string]interface{}{"error": err.Error()})
		return
	}

	logger.Info("Routing configuration updated (canonical apply store)")
	common.Success(w, map[string]interface{}{
		"message": "Configuration updated successfully",
		"source":  "canonical",
		"version": applyResult.Version,
		"hash":    applyResult.Hash,
	})
}

func ioReadAll(r *http.Request) ([]byte, error) {
	if r == nil || r.Body == nil {
		return nil, fmt.Errorf("request body is empty")
	}
	defer r.Body.Close()
	return io.ReadAll(r.Body)
}

func activeCanonicalRoutingConfig() *config.CanonicalRoutingSchema {
	canonical := config.GetRoutingApplyStore().ActiveCanonicalSchema()
	if canonical == nil {
		return &config.CanonicalRoutingSchema{
			Version: config.CanonicalRoutingSchemaVersion,
			Ingress: config.CanonicalIngressConfig{Mode: "tun"},
			Egress: config.CanonicalEgressConfig{
				Direct:   config.CanonicalEgressBranch{Enabled: true},
				ToAliang: config.CanonicalEgressBranch{Enabled: false},
				ToSocks:  config.CanonicalSocksEgressBranch{Enabled: false, Upstream: config.CanonicalSocksUpstream{Type: "socks"}},
			},
			Routing: config.CanonicalRoutingConfig{Rules: []config.CanonicalRoutingRule{}},
		}
	}
	return canonical
}
