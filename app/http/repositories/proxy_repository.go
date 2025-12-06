package repositories

import (
	"errors"

	"nursor.org/nursorgate/app/http/services"
	proxyRegistry "nursor.org/nursorgate/outbound"
	proxyConfig "nursor.org/nursorgate/processor/config"
)

// ProxyType defines the type of proxy operation
type ProxyType string

const (
	ProxyTypeDefault   ProxyType = "default"
	ProxyTypeDoor      ProxyType = "door"
	ProxyTypeNonelane  ProxyType = "nonelane"
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

// ListProxies lists all proxies
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

// RegisterProxy registers a new proxy
func (pr *ProxyRepositoryImpl) RegisterProxy(name string, config interface{}) error {
	registry := proxyRegistry.GetRegistry()
	if cfg, ok := config.(*proxyConfig.ProxyConfig); ok {
		return registry.RegisterFromConfig(name, cfg)
	}
	return errors.New("invalid config type")
}

// UnregisterProxy unregisters a proxy
func (pr *ProxyRepositoryImpl) UnregisterProxy(name string) error {
	registry := proxyRegistry.GetRegistry()
	return registry.Unregister(name)
}

// Get gets a proxy by type and name
// For ProxyTypeDefault, name can be empty to get current default
func (pr *ProxyRepositoryImpl) Get(proxyType ProxyType, name string) (interface{}, error) {
	registry := proxyRegistry.GetRegistry()

	switch proxyType {
	case ProxyTypeDefault:
		if name != "" {
			// Get specific proxy by name
			return registry.Get(name)
		}
		// Get current default proxy
		return registry.GetDefault()

	case ProxyTypeDoor:
		return registry.GetDoor()

	case ProxyTypeNonelane:
		return registry.GetNonelane()

	default:
		return nil, errors.New("unsupported proxy type")
	}
}

// Set sets a proxy by type and name
// For ProxyTypeDefault, sets the default proxy
// For ProxyTypeDoor, sets the door proxy
func (pr *ProxyRepositoryImpl) Set(proxyType ProxyType, name string) error {
	if name == "" {
		return errors.New("proxy name cannot be empty")
	}

	registry := proxyRegistry.GetRegistry()

	switch proxyType {
	case ProxyTypeDefault:
		return registry.SetDefault(name)

	case ProxyTypeDoor:
		return registry.SetDoor(name)

	default:
		return errors.New("unsupported proxy type")
	}
}

// SetDefaultProxy sets the default proxy (backward compatibility - calls Set)
func (pr *ProxyRepositoryImpl) SetDefaultProxy(name string) error {
	return pr.Set(ProxyTypeDefault, name)
}

// SetDoorProxy sets the door proxy (backward compatibility - calls Set)
func (pr *ProxyRepositoryImpl) SetDoorProxy(name string) error {
	return pr.Set(ProxyTypeDoor, name)
}

// SwitchProxy switches to a proxy (backward compatibility - calls Set)
func (pr *ProxyRepositoryImpl) SwitchProxy(name string) error {
	return pr.Set(ProxyTypeDefault, name)
}