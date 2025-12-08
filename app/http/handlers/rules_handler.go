package handlers

import (
	"net/http"

	"nursor.org/nursorgate/app/http/common"
	"nursor.org/nursorgate/processor/geoip"
	"nursor.org/nursorgate/processor/rules"
)

// RulesHandler handles HTTP requests for routing rules operations
type RulesHandler struct{}

// NewRulesHandler creates a new rules handler instance
func NewRulesHandler() *RulesHandler {
	return &RulesHandler{}
}

// HandleGetGeoIPStatus handles GET /api/rules/geoip/status
// Returns the current status of the GeoIP service
func (rh *RulesHandler) HandleGetGeoIPStatus(w http.ResponseWriter, r *http.Request) {
	service := geoip.GetService()

	status := map[string]interface{}{
		"enabled":      service.IsEnabled(),
		"databasePath": service.GetDatabasePath(),
	}

	common.Success(w, status)
}

// HandleGeoIPLookup handles POST /api/rules/geoip/lookup
// Performs a GeoIP lookup for the provided IP address
func (rh *RulesHandler) HandleGeoIPLookup(w http.ResponseWriter, r *http.Request) {
	var req struct {
		IP string `json:"ip"`
	}

	if err := common.DecodeJSON(r.Body, &req); err != nil {
		common.ErrorBadRequest(w, "Invalid request format", nil)
		return
	}

	if req.IP == "" {
		common.ErrorBadRequest(w, "IP address is required", nil)
		return
	}

	// Parse IP
	ip := common.ParseIP(req.IP)
	if ip == nil {
		common.ErrorBadRequest(w, "Invalid IP address format", nil)
		return
	}

	// Lookup country
	service := geoip.GetService()
	if !service.IsEnabled() {
		common.ErrorServiceUnavailable(w, "GeoIP service is not enabled")
		return
	}

	country, err := service.LookupCountry(ip)
	if err != nil {
		common.ErrorInternalServer(w, "GeoIP lookup failed: "+err.Error(), nil)
		return
	}

	data := map[string]interface{}{
		"ip":      req.IP,
		"country": country.ISOCode,
		"name":    country.Name,
		"isChina": country.ISOCode == "CN",
	}

	common.Success(w, data)
}

// HandleGetCacheStats handles GET /api/rules/cache/stats
// Returns cache statistics (hit rate, size, etc.)
func (rh *RulesHandler) HandleGetCacheStats(w http.ResponseWriter, r *http.Request) {
	engine := rules.GetEngine()
	if engine == nil {
		common.ErrorNotFound(w, "Rule engine not initialized")
		return
	}

	stats := engine.GetCacheStats()
	common.Success(w, stats)
}

// HandleClearCache handles POST /api/rules/cache/clear
// Clears all cached routing decisions
func (rh *RulesHandler) HandleClearCache(w http.ResponseWriter, r *http.Request) {
	engine := rules.GetEngine()
	if engine == nil {
		common.ErrorNotFound(w, "Rule engine not initialized")
		return
	}

	engine.ClearCache()

	common.Success(w, map[string]string{
		"status": "cache cleared successfully",
	})
}

// HandleGetRuleEngineStatus handles GET /api/rules/engine/status
// Returns the overall status of the rule engine
func (rh *RulesHandler) HandleGetRuleEngineStatus(w http.ResponseWriter, r *http.Request) {
	engine := rules.GetEngine()
	geoipService := geoip.GetService()

	status := map[string]interface{}{
		"engineEnabled": engine != nil && engine.IsEnabled(),
		"geoipEnabled":  geoipService != nil && geoipService.IsEnabled(),
	}

	if engine != nil && engine.IsEnabled() {
		stats := engine.GetCacheStats()
		status["cache"] = stats
	}

	common.Success(w, status)
}

// HandleEnableRuleEngine handles POST /api/rules/engine/enable
// Enables the rule engine
func (rh *RulesHandler) HandleEnableRuleEngine(w http.ResponseWriter, r *http.Request) {
	engine := rules.GetEngine()
	if engine == nil {
		common.ErrorNotFound(w, "Rule engine not initialized")
		return
	}

	engine.Enable()

	common.Success(w, map[string]interface{}{
		"status":  "success",
		"enabled": true,
	})
}

// HandleDisableRuleEngine handles POST /api/rules/engine/disable
// Disables the rule engine
func (rh *RulesHandler) HandleDisableRuleEngine(w http.ResponseWriter, r *http.Request) {
	engine := rules.GetEngine()
	if engine == nil {
		common.ErrorNotFound(w, "Rule engine not initialized")
		return
	}

	engine.Disable()

	common.Success(w, map[string]interface{}{
		"status":  "success",
		"enabled": false,
	})
}
