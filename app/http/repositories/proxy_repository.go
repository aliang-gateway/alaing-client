package repositories

import (
	"errors"

	"nursor.org/nursorgate/app/http/services"
	proxyRegistry "nursor.org/nursorgate/outbound"
	proxyConfig "nursor.org/nursorgate/processor/config"
)

// ProxyRepositoryImpl provides access to proxy functionality
type ProxyRepositoryImpl struct {
	proxyService *services.ProxyService
}

// NewProxyRepository creates a new proxy repository instance
func NewProxyRepository() *ProxyRepositoryImpl {
	return &ProxyRepositoryImpl{
		proxyService: services.NewProxyService(),
	}
}

// GetCurrentProxy gets the current default proxy (backward compatibility)
func (pr *ProxyRepositoryImpl) GetCurrentProxy() (map[string]interface{}, error) {
	return pr.proxyService.GetCurrentProxy()
}

// SetCurrentProxy sets the current default proxy (backward compatibility)
func (pr *ProxyRepositoryImpl) SetCurrentProxy(name string) (map[string]interface{}, error) {
	return pr.proxyService.SetCurrentProxy(name)
}

// ListProxies lists all registered proxies
func (pr *ProxyRepositoryImpl) ListProxies() (map[string]interface{}, error) {
	registry := proxyRegistry.GetRegistry()
	info := registry.ListWithInfo()
	return map[string]interface{}{
		"proxies": info,
		"count":   len(info),
	}, nil
}

// GetProxy gets a specific proxy by name
func (pr *ProxyRepositoryImpl) GetProxy(name string) (interface{}, error) {
	registry := proxyRegistry.GetRegistry()
	info := registry.ListWithInfo()
	proxyInfo, exists := info[name]
	if !exists {
		return nil, errors.New("proxy not found")
	}
	return proxyInfo, nil
}

// Register registers a new proxy instance
func (pr *ProxyRepositoryImpl) Register(name string, config interface{}) error {
	registry := proxyRegistry.GetRegistry()
	if cfg, ok := config.(*proxyConfig.ProxyConfig); ok {
		return registry.RegisterFromConfig(name, cfg)
	}
	return errors.New("invalid config type")
}

// Unregister removes a proxy from the registry
func (pr *ProxyRepositoryImpl) Unregister(name string) error {
	registry := proxyRegistry.GetRegistry()
	return registry.Unregister(name)
}

// GetByName gets a specific proxy by name
// Supports both regular proxies (direct, nonelane, custom names) and door proxy members
// Door proxy members use format: "door:ShowName" (e.g., "door:日本 Tokyo")
func (pr *ProxyRepositoryImpl) GetByName(name string) (interface{}, error) {
	if name == "" {
		return nil, errors.New("proxy name cannot be empty")
	}

	registry := proxyRegistry.GetRegistry()
	return registry.Get(name)
}

