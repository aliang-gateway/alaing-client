package handlers

import (
	"fmt"
	"net/http"

	"nursor.org/nursorgate/app/http/common"
	"nursor.org/nursorgate/common/logger"
	tun "nursor.org/nursorgate/inbound/tun/engine"
	user "nursor.org/nursorgate/processor/auth"
	"nursor.org/nursorgate/runner"
)

// handleRun 处理 /run/start
func handleRun(w http.ResponseWriter, r *http.Request) {
	var req struct {
		// UserToken  string `json:"user_token"`
		InnerToken string `json:"inner_token"`
	}
	if err := common.DecodeRequest(r, &req); err != nil {
		common.SendError(w, "Invalid request body", http.StatusBadRequest, nil)
		return
	}
	// user.SetUserToken(req.UserToken)
	user.SetInnerToken(req.InnerToken)
	go runner.Start()
	res := <-runner.RunStatusChan
	common.SendResponse(w, res)
}

// handleStop 处理 /run/stop
func handleStop(w http.ResponseWriter, r *http.Request) {
	tun.Stop()
	common.SendResponse(w, map[string]string{"status": "success"})
}

// handleRunUserInfo 处理 /run/userInfo
func handleRunUserInfo(w http.ResponseWriter, r *http.Request) {
	var req struct {
		UserUUID   string `json:"user_uuid"`
		InnerToken string `json:"inner_token"`
		Username   string `json:"username"`
		Password   string `json:"password"`
	}
	if err := common.DecodeRequest(r, &req); err != nil {
		common.SendError(w, "Invalid request body", http.StatusBadRequest, nil)
		return
	}
	logger.SetUserInfo(req.InnerToken)
	user.SetUsername(req.Username)
	user.SetPassword(req.Password)
	user.SetUserUUID(req.UserUUID)
	logger.Info("set user info tag")
	common.SendResponse(w, map[string]string{
		"status":  "success",
		"user_id": fmt.Sprintf("%d", user.GetUserId()),
	})
}

// RegisterRunRoutes 注册Run相关路由
func RegisterRunRoutes() {
	http.HandleFunc("/run/start", handleRun)
	http.HandleFunc("/run/stop", handleStop)
	http.HandleFunc("/run/userInfo", handleRunUserInfo)
}
