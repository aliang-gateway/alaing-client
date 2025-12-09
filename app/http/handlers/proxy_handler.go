package handlers

import (
	"net/http"

	"nursor.org/nursorgate/app/http/common"
	"nursor.org/nursorgate/app/http/services"
)

// ProxyHandler handles HTTP requests for proxy operations
type ProxyHandler struct {
	proxyService *services.ProxyService
}

// NewProxyHandler creates a new proxy handler instance with dependency injection
func NewProxyHandler(proxyService *services.ProxyService) *ProxyHandler {
	return &ProxyHandler{
		proxyService: proxyService,
	}
}

// HandleGetCurrentProxy handles GET /api/proxy/current/get
// Returns the current door proxy member information
func (ph *ProxyHandler) HandleGetCurrentProxy(w http.ResponseWriter, r *http.Request) {
	proxyInfo, err := ph.proxyService.GetCurrentProxy()
	if err != nil {
		common.ErrorInternalServer(w, "Failed to get current proxy", err)
		return
	}

	// Check if proxyInfo contains an error (service returns error info in the map)
	if errorMsg, exists := proxyInfo["error"]; exists {
		common.ErrorNotFound(w, errorMsg.(string))
		return
	}

	common.Success(w, proxyInfo)
}

// HandleSetCurrentProxy handles POST /api/proxy/current/set
func (ph *ProxyHandler) HandleSetCurrentProxy(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name string `json:"name"`
	}

	if err := common.DecodeRequest(r, &req); err != nil {
		common.ErrorBadRequest(w, "Invalid request body", nil)
		return
	}

	if req.Name == "" {
		common.ErrorBadRequest(w, "name is required", nil)
		return
	}

	proxyInfo, err := ph.proxyService.SetCurrentProxy(req.Name)
	if err != nil {
		common.ErrorBadRequest(w, err.Error(), nil)
		return
	}

	common.Success(w, proxyInfo)
}
