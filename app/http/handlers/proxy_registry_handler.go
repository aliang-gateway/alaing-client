package handlers

import (
	"net/http"

	"nursor.org/nursorgate/app/http/common"
	"nursor.org/nursorgate/app/http/repositories"
	"nursor.org/nursorgate/outbound"
	proxyConfig "nursor.org/nursorgate/processor/config"
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
// Returns all proxies including direct, nonelane, custom proxies, and door virtual members (door:xxx format)
// Also includes current proxy information and door auto-select mode status
func (prh *ProxyRegistryHandler) HandleProxyRegistryList(w http.ResponseWriter, r *http.Request) {
	result, err := prh.proxyRepository.ListProxies()
	if err != nil {
		common.ErrorInternalServer(w, "Failed to list proxies", nil)
		return
	}

	// Get current proxy information from registry
	registry := outbound.GetRegistry()
	currentProxyName := ""
	currentProxyType := ""
	currentProxyAddr := ""
	currentProxyShowName := ""

	// Check if door auto-select is enabled first
	isDoorAutoMode := registry.IsDoorAutoSelect()

	// Try to get current door member
	currentMember := registry.GetDoorCurrentMember()
	if currentMember != "" {
		// Current is a door member
		currentProxyName = "door:" + currentMember
		currentProxyShowName = currentMember

		// Get door proxy to extract type and addr
		if doorProxy, err := registry.GetDoor(); err == nil {
			currentProxyType = doorProxy.Proto().String()
			currentProxyAddr = doorProxy.Addr()
		}
	} else if isDoorAutoMode {
		// Door auto-select is enabled, so current is door proxy in auto mode
		if doorProxy, err := registry.GetDoor(); err == nil {
			currentProxyName = "door"
			currentProxyType = doorProxy.Proto().String()
			currentProxyAddr = doorProxy.Addr()
		}
	} else {
		// For non-door proxies, use the hardcoded default (direct)
		if defaultProxy, err := registry.GetHardcodedDefault(); err == nil {
			currentProxyName = "direct"
			currentProxyType = defaultProxy.Proto().String()
			currentProxyAddr = defaultProxy.Addr()
		}
	}

	// Build response with all information
	responseData := map[string]interface{}{
		"proxies":           result["proxies"],
		"count":             result["count"],
		"is_door_auto_mode": isDoorAutoMode,
	}

	// Add current proxy information if available
	if currentProxyName != "" {
		currentProxyInfo := map[string]interface{}{
			"name": currentProxyName,
			"type": currentProxyType,
			"addr": currentProxyAddr,
		}
		if currentProxyShowName != "" {
			currentProxyInfo["show_name"] = currentProxyShowName
		}
		responseData["current_proxy"] = currentProxyInfo
	}

	common.Success(w, responseData)
}

// HandleProxyRegistryGet handles GET /api/proxy/registry/get
// Query parameter: name (required) - get specific proxy by name
// Returns complete proxy configuration information including all config details
// Supported formats:
//   - "direct" - direct proxy
//   - "nonelane" - nonelane proxy
//   - "custom_name" - custom proxy
//   - "door:ShowName" - door proxy member (e.g., "door:日本 Tokyo")
//
// Returns configuration from globalConfig along with runtime proxy information
func (prh *ProxyRegistryHandler) HandleProxyRegistryGet(w http.ResponseWriter, r *http.Request) {
	name := common.GetQueryParamString(r, "name", "")
	if name == "" {
		common.ErrorBadRequest(w, "name parameter is required", nil)
		return
	}

	// First, try to get complete configuration information from global config
	configInfo, err := proxyConfig.GetProxyConfigInfo(name)
	if err != nil {
		// If not found in config, try to get from registry (for dynamically created proxies)
		proxyInstance, repoErr := prh.proxyRepository.GetByName(name)
		if repoErr != nil {
			common.ErrorNotFound(w, "proxy not found in configuration or registry")
			return
		}

		// Build fallback proxy info from runtime instance
		proxyInfo := map[string]interface{}{
			"name": name,
		}

		// Get basic info from proxy instance
		if baseProxy, ok := proxyInstance.(interface {
			Addr() string
			Proto() interface{ String() string }
		}); ok {
			proxyInfo["type"] = baseProxy.Proto().String()
			proxyInfo["addr"] = baseProxy.Addr()
		} else {
			proxyInfo["type"] = "unknown"
			proxyInfo["error"] = "could not extract proxy details"
		}
		proxyInfo["source"] = "runtime"

		common.Success(w, proxyInfo)
		return
	}

	// Also fetch runtime proxy info to include address and runtime details
	registry := outbound.GetRegistry()
	proxyInstance, err := registry.Get(name)
	if err == nil && proxyInstance != nil {
		// Add runtime information to config info
		configInfo["addr"] = proxyInstance.Addr()
		configInfo["proto"] = proxyInstance.Proto().String()
	}

	common.Success(w, configInfo)
}
