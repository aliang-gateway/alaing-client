package handlers

import (
	"net/http"

	"nursor.org/nursorgate/app/http/common"
	"nursor.org/nursorgate/outbound"
)

// handleTokenSet 处理 /token/set
func handleTokenSet(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Token string `json:"token"`
	}
	if err := common.DecodeRequest(r, &req); err != nil {
		common.SendError(w, "Invalid request body", http.StatusBadRequest, nil)
		return
	}
	outbound.SetOutboundToken(req.Token)
	common.SendResponse(w, map[string]string{"token": req.Token})
}

// handleTokenGet 处理 /token/get
func handleTokenGet(w http.ResponseWriter, r *http.Request) {
	common.SendResponse(w, map[string]string{"token": outbound.GetOutboundToken()})
}

// RegisterTokenRoutes 注册Token相关路由
func RegisterTokenRoutes() {
	http.HandleFunc("/token/set", handleTokenSet)
	http.HandleFunc("/token/get", handleTokenGet)
}
