package services

import (
	proxyRegistry "nursor.org/nursorgate/outbound"
)

// ProxyService handles proxy operations
type ProxyService struct{}

// NewProxyService creates a new proxy service instance
func NewProxyService() *ProxyService {
	return &ProxyService{}
}

// GetCurrentProxy gets the current default proxy
func (ps *ProxyService) GetCurrentProxy() (map[string]interface{}, error) {
	registry := proxyRegistry.GetRegistry()
	currentName := registry.GetDefaultName()
	proxy, err := registry.GetDefault()

	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"name": currentName,
		"type": proxy.Proto().String(),
		"addr": proxy.Addr(),
	}, nil
}

// SetCurrentProxy sets the current default proxy
func (ps *ProxyService) SetCurrentProxy(name string) (map[string]interface{}, error) {
	registry := proxyRegistry.GetRegistry()
	if err := registry.SetDefault(name); err != nil {
		return nil, err
	}

	proxy, _ := registry.GetDefault()
	return map[string]interface{}{
		"name": name,
		"type": proxy.Proto().String(),
		"addr": proxy.Addr(),
	}, nil
}
