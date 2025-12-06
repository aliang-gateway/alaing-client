package handlers

import (
	"net/http"

	"nursor.org/nursorgate/app/http/common"
	"nursor.org/nursorgate/app/http/repositories"
	"nursor.org/nursorgate/processor/config"
)

// ProxyRegistryHandler handles HTTP requests for proxy registry operations
type ProxyRegistryHandler struct {
	proxyRepository *repositories.ProxyRepositoryImpl
}

// NewProxyRegistryHandler creates a new proxy registry handler instance with dependency injection
func NewProxyRegistryHandler(proxyRepository *repositories.ProxyRepositoryImpl) *ProxyRegistryHandler {
	return &ProxyRegistryHandler{
		proxyRepository: proxyRepository,
	}
}

// HandleProxyRegistryList handles GET /api/proxy/registry/list
func (prh *ProxyRegistryHandler) HandleProxyRegistryList(w http.ResponseWriter, r *http.Request) {
	result, err := prh.proxyRepository.ListProxies()
	if err != nil {
		common.ErrorInternalServer(w, "Failed to list proxies", nil)
		return
	}

	common.Success(w, result)
}

// HandleProxyRegistryGet handles GET /api/proxy/registry/get
func (prh *ProxyRegistryHandler) HandleProxyRegistryGet(w http.ResponseWriter, r *http.Request) {
	name := common.GetQueryParamString(r, "name", "")
	if name == "" {
		common.ErrorBadRequest(w, "name parameter is required", nil)
		return
	}

	proxyInfo, err := prh.proxyRepository.GetProxy(name)
	if err != nil {
		common.ErrorNotFound(w, "proxy info not found")
		return
	}

	common.Success(w, proxyInfo)
}

// HandleProxyRegistryRegister handles POST /api/proxy/registry/register
func (prh *ProxyRegistryHandler) HandleProxyRegistryRegister(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name   string                 `json:"name"`
		Config *config.ProxyConfig `json:"config"`
	}

	if err := common.DecodeRequest(r, &req); err != nil {
		common.ErrorBadRequest(w, "Invalid request body", nil)
		return
	}

	if req.Name == "" {
		common.ErrorBadRequest(w, "name is required", nil)
		return
	}

	if req.Config == nil {
		common.ErrorBadRequest(w, "config is required", nil)
		return
	}

	if err := prh.proxyRepository.RegisterProxy(req.Name, req.Config); err != nil {
		common.ErrorBadRequest(w, err.Error(), nil)
		return
	}

	common.Success(w, map[string]string{"status": "success", "name": req.Name})
}

// HandleProxyRegistryUnregister handles POST /api/proxy/registry/unregister
func (prh *ProxyRegistryHandler) HandleProxyRegistryUnregister(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name string `json:"name"`
	}

	if err := common.DecodeRequest(r, &req); err != nil {
		common.ErrorBadRequest(w, "Invalid request body", nil)
		return
	}

	if err := prh.proxyRepository.UnregisterProxy(req.Name); err != nil {
		common.ErrorBadRequest(w, err.Error(), nil)
		return
	}

	common.Success(w, map[string]string{"status": "success"})
}

// HandleProxyRegistrySetDefault handles POST /api/proxy/registry/set-default
func (prh *ProxyRegistryHandler) HandleProxyRegistrySetDefault(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name string `json:"name"`
	}

	if err := common.DecodeRequest(r, &req); err != nil {
		common.ErrorBadRequest(w, "Invalid request body", nil)
		return
	}

	if err := prh.proxyRepository.SetDefaultProxy(req.Name); err != nil {
		common.ErrorBadRequest(w, err.Error(), nil)
		return
	}

	common.Success(w, map[string]string{"status": "success", "default": req.Name})
}

// HandleProxyRegistrySetDoor handles POST /api/proxy/registry/set-door
func (prh *ProxyRegistryHandler) HandleProxyRegistrySetDoor(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name string `json:"name"`
	}

	if err := common.DecodeRequest(r, &req); err != nil {
		common.ErrorBadRequest(w, "Invalid request body", nil)
		return
	}

	if err := prh.proxyRepository.SetDoorProxy(req.Name); err != nil {
		common.ErrorBadRequest(w, err.Error(), nil)
		return
	}

	common.Success(w, map[string]string{"status": "success", "door": req.Name})
}

// HandleProxyRegistrySwitch handles POST /api/proxy/registry/switch
func (prh *ProxyRegistryHandler) HandleProxyRegistrySwitch(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name string `json:"name"`
	}

	if err := common.DecodeRequest(r, &req); err != nil {
		common.ErrorBadRequest(w, "Invalid request body", nil)
		return
	}

	if err := prh.proxyRepository.SwitchProxy(req.Name); err != nil {
		common.ErrorBadRequest(w, err.Error(), nil)
		return
	}

	proxyInfo, err := prh.proxyRepository.GetProxy(req.Name)
	if err != nil {
		common.ErrorNotFound(w, "proxy not found after switch")
		return
	}

	common.Success(w, proxyInfo)
}
