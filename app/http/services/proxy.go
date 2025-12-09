package services

import (
	"fmt"

	proxyRegistry "nursor.org/nursorgate/outbound"
)

// ProxyService handles proxy operations
type ProxyService struct{}

// NewProxyService creates a new proxy service instance
func NewProxyService() *ProxyService {
	return &ProxyService{}
}

// GetCurrentProxy gets the current door proxy member information
// Only returns door proxy's current member, not direct proxy
func (ps *ProxyService) GetCurrentProxy() (map[string]interface{}, error) {
	registry := proxyRegistry.GetRegistry()

	// Get door proxy group
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

// SetCurrentProxy sets the current door proxy member
// Only accepts door:memberName format and sets door proxy current member
func (ps *ProxyService) SetCurrentProxy(name string) (map[string]interface{}, error) {
	registry := proxyRegistry.GetRegistry()

	// Check if it's a door member format (door:memberName)
	if len(name) <= 5 || name[:5] != "door:" {
		return nil, fmt.Errorf("only door members can be set as current, format: door:memberName")
	}

	memberName := name[5:]
	if memberName == "" {
		return nil, fmt.Errorf("member name cannot be empty")
	}

	// Set the door member
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
