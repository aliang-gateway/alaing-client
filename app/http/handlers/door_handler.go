package handlers

import (
	"net/http"

	"nursor.org/nursorgate/app/http/common"
	"nursor.org/nursorgate/outbound"
)

// DoorHandler handles HTTP requests for door proxy operations
type DoorHandler struct{}

// NewDoorHandler creates a new door handler instance
func NewDoorHandler() *DoorHandler {
	return &DoorHandler{}
}

// HandleDoorMemberList handles GET /api/proxy/door/members
func (dh *DoorHandler) HandleDoorMemberList(w http.ResponseWriter, r *http.Request) {
	registry := outbound.GetRegistry()
	members, err := registry.ListDoorMembers()
	if err != nil {
		common.ErrorNotFound(w, err.Error())
		return
	}

	// 转换为响应格式
	type MemberResponse struct {
		ShowName   string `json:"showname"`
		Type       string `json:"type"`
		Addr       string `json:"addr"`
		Latency    int64  `json:"latency"`
		LastUpdate int64  `json:"last_update"`
	}

	membersResp := make([]MemberResponse, 0, len(members))
	for _, member := range members {
		membersResp = append(membersResp, MemberResponse{
			ShowName:   member.ShowName,
			Type:       member.Proxy.Proto().String(),
			Addr:       member.Proxy.Addr(),
			Latency:    member.Latency,
			LastUpdate: member.LastUpdate,
		})
	}

	common.Success(w, map[string]interface{}{
		"members": membersResp,
		"count":   len(membersResp),
	})
}

// HandleDoorMemberSwitch handles POST /api/proxy/door/switch
func (dh *DoorHandler) HandleDoorMemberSwitch(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ShowName string `json:"showname"`
	}

	if err := common.DecodeRequest(r, &req); err != nil {
		common.ErrorBadRequest(w, "Invalid request body", nil)
		return
	}

	if req.ShowName == "" {
		common.ErrorBadRequest(w, "showname is required", nil)
		return
	}

	registry := outbound.GetRegistry()
	if err := registry.SetDoorMember(req.ShowName); err != nil {
		common.ErrorBadRequest(w, err.Error(), nil)
		return
	}

	common.Success(w, map[string]interface{}{
		"status": "success",
		"member": req.ShowName,
	})
}

// HandleDoorAutoSelect handles POST /api/proxy/door/auto
func (dh *DoorHandler) HandleDoorAutoSelect(w http.ResponseWriter, r *http.Request) {
	registry := outbound.GetRegistry()
	if err := registry.EnableDoorAutoSelect(); err != nil {
		common.ErrorBadRequest(w, err.Error(), nil)
		return
	}

	common.Success(w, map[string]interface{}{
		"status":  "success",
		"message": "auto select enabled",
	})
}
