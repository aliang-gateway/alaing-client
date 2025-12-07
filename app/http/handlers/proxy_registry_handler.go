package handlers

import (
	"net/http"

	"nursor.org/nursorgate/app/http/common"
	"nursor.org/nursorgate/app/http/repositories"
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
// Query parameter: name (required) - get specific proxy by name
func (prh *ProxyRegistryHandler) HandleProxyRegistryGet(w http.ResponseWriter, r *http.Request) {
	name := common.GetQueryParamString(r, "name", "")
	if name == "" {
		common.ErrorBadRequest(w, "name parameter is required", nil)
		return
	}

	proxyInfo, err := prh.proxyRepository.GetByName(name)
	if err != nil {
		common.ErrorNotFound(w, "proxy not found")
		return
	}

	common.Success(w, proxyInfo)
}
