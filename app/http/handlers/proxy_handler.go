package handlers

import (
	"fmt"
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
// Returns the current door proxy member information
func (ph *ProxyHandler) HandleGetCurrentProxy(w http.ResponseWriter, r *http.Request) {
	registry := proxyRegistry.GetRegistry()
	doorGroup := registry.GetDoorGroup()

	if doorGroup == nil || doorGroup.Count() == 0 {
		common.ErrorNotFound(w, "No door proxy configured")
		return
	}

	// Get current member name
	currentMemberName := doorGroup.GetCurrentMemberName()
	if currentMemberName == "" {
		// Auto-select is enabled, find best member
		members := doorGroup.ListMembers()
		if len(members) == 0 {
			common.ErrorNotFound(w, "No door members available")
			return
		}
		// Find member with lowest latency
		var bestMember *proxyRegistry.DoorProxyMemberInfo
		var lowestLatency int64 = -1
		for i := range members {
			member := &members[i]
			if lowestLatency == -1 || member.Latency < lowestLatency {
				lowestLatency = member.Latency
				bestMember = member
			}
		}
		if bestMember == nil {
			common.ErrorNotFound(w, "No current door member selected")
			return
		}
		currentMemberName = bestMember.ShowName
	}

	// Get current member details
	member, err := doorGroup.GetMember(currentMemberName)
	if err != nil {
		common.ErrorInternalServer(w, fmt.Sprintf("Failed to get current door member: %v", err), nil)
		return
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

	proxyInfo := map[string]interface{}{
		"name":      "door:" + currentMemberName,
		"type":      member.Proto().String(),
		"addr":      member.Addr(),
		"show_name": currentMemberName,
		"latency":   latency,
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

	// Validate format: must be "door:memberName"
	if len(req.Name) <= 5 || req.Name[:5] != "door:" {
		common.ErrorBadRequest(w, "Invalid format, expected: door:memberName", nil)
		return
	}

	memberName := req.Name[5:]
	if memberName == "" {
		common.ErrorBadRequest(w, "member name cannot be empty", nil)
		return
	}

	// Set the door member
	registry := proxyRegistry.GetRegistry()
	if err := registry.SetDoorMember(memberName); err != nil {
		common.ErrorBadRequest(w, fmt.Sprintf("Failed to set door member '%s': %v", memberName, err), nil)
		return
	}

	// Get the door proxy for response
	doorProxy, err := registry.GetDoor(memberName)
	if err != nil {
		common.ErrorInternalServer(w, fmt.Sprintf("Failed to get door proxy: %v", err), nil)
		return
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

	proxyInfo := map[string]interface{}{
		"name":      "door:" + memberName,
		"type":      doorProxy.Proto().String(),
		"addr":      doorProxy.Addr(),
		"show_name": memberName,
		"latency":   latency,
		"success":   true,
	}

	common.Success(w, proxyInfo)
}
