package handlers

import (
	"net/http"

	"nursor.org/nursorgate/app/http/common"
	proxyRegistry "nursor.org/nursorgate/outbound"
)

// ProxyHandler handles HTTP requests for proxy operations
type ProxyHandler struct{}

// NewProxyHandler creates a new proxy handler instance
func NewProxyHandler() *ProxyHandler {
	return &ProxyHandler{}
}

// HandleGetCurrentProxy handles GET /api/proxy/current/get
func (ph *ProxyHandler) HandleGetCurrentProxy(w http.ResponseWriter, r *http.Request) {
	registry := proxyRegistry.GetRegistry()
	currentProxy, err := registry.GetHardcodedDefault()
	if err != nil {
		common.ErrorNotFound(w, "No proxy configured")
		return
	}

	proxyInfo := map[string]interface{}{
		"name": "direct",
		"type": currentProxy.Proto().String(),
		"addr": currentProxy.Addr(),
	}

	common.Success(w, proxyInfo)
}

// HandleSetCurrentProxy handles POST /api/proxy/current/set
func (ph *ProxyHandler) HandleSetCurrentProxy(w http.ResponseWriter, r *http.Request) {
	common.ErrorBadRequest(w, "set current proxy is no longer supported", nil)
}
