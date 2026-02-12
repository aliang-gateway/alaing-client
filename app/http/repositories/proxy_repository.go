package repositories

import (
	"errors"
	"fmt"

	proxyRegistry "nursor.org/nursorgate/outbound"
)

// ProxyRepositoryImpl provides access to proxy functionality
type ProxyRepositoryImpl struct{}

// NewProxyRepository creates a new proxy repository instance
func NewProxyRepository() *ProxyRepositoryImpl {
	return &ProxyRepositoryImpl{}
}

// GetCurrentProxy gets the current default proxy (backward compatibility)
func (pr *ProxyRepositoryImpl) GetCurrentProxy() (map[string]interface{}, error) {
	registry := proxyRegistry.GetRegistry()
	doorGroup := registry.GetDoorGroup()

	if doorGroup == nil || doorGroup.Count() == 0 {
		return map[string]interface{}{
			"error": "No door proxy configured",
		}, nil
	}

	// Get current member name
	currentMemberName := doorGroup.GetCurrentMemberName()
	if currentMemberName == "" {
		return map[string]interface{}{
			"error": "No current door member selected",
		}, nil
	}

	// Get current member details
	member, err := doorGroup.GetMember(currentMemberName)
	if err != nil {
		return map[string]interface{}{
			"error": fmt.Sprintf("Failed to get current door member: %v", err),
		}, nil
	}

	// Get member list for latency info
	members := doorGroup.ListMembers()
	var latency int64
	for _, m := range members {
		if m.ShowName == currentMemberName {
			latency = m.Latency
			break
		}
	}

	return map[string]interface{}{
		"name":      "door:" + currentMemberName,
		"type":      member.Proto().String(),
		"addr":      member.Addr(),
		"show_name": currentMemberName,
		"latency":   latency,
	}, nil
}

// SetCurrentProxy sets the current default proxy (backward compatibility)
func (pr *ProxyRepositoryImpl) SetCurrentProxy(name string) (map[string]interface{}, error) {
	// Validate format: must be "door:memberName"
	if len(name) <= 5 || name[:5] != "door:" {
		return nil, errors.New("only door members can be set as current, format: door:memberName")
	}

	memberName := name[5:]
	if memberName == "" {
		return nil, errors.New("member name cannot be empty")
	}

	// Set the door member
	registry := proxyRegistry.GetRegistry()
	if err := registry.SetDoorMember(memberName); err != nil {
		return nil, fmt.Errorf("failed to set door member '%s': %w", memberName, err)
	}

	// Get the door proxy for response
	doorProxy, err := registry.GetDoor(memberName)
	if err != nil {
		return nil, fmt.Errorf("failed to get door proxy: %w", err)
	}

	// Get member list for latency info
	doorGroup := registry.GetDoorGroup()
	var latency int64
	if doorGroup != nil {
		members := doorGroup.ListMembers()
		for _, m := range members {
			if m.ShowName == memberName {
				latency = m.Latency
				break
			}
		}
	}

	return map[string]interface{}{
		"name":      "door:" + memberName,
		"type":      doorProxy.Proto().String(),
		"addr":      doorProxy.Addr(),
		"show_name": memberName,
		"latency":   latency,
		"success":   true,
	}, nil
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
