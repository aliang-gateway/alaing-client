package handlers

import (
	"fmt"
	"net/http"
	"sync"

	"nursor.org/nursorgate/app/http/common"
	"nursor.org/nursorgate/common/logger"
	httpServer "nursor.org/nursorgate/inbound/http"
	tun "nursor.org/nursorgate/inbound/tun/engine"
	runner2 "nursor.org/nursorgate/inbound/tun/runner"
	user "nursor.org/nursorgate/processor/auth"
)

// RunMode represents the current operation mode
type RunMode string

const (
	ModeHTTP RunMode = "http"
	ModeTUN  RunMode = "tun"
)

// Global state for managing run modes
var (
	currentMode     RunMode = ModeHTTP // 默认为HTTP模式
	tunRunning      bool
	modeChangeMutex sync.RWMutex
)

// ============================================================================
// FFI State Management Functions - Exposed for direct API access
// ============================================================================

// GetCurrentMode returns the current operating mode
func GetCurrentMode() string {
	modeChangeMutex.RLock()
	defer modeChangeMutex.RUnlock()
	return string(currentMode)
}

// SetCurrentMode sets the operating mode
func SetCurrentMode(mode string) {
	modeChangeMutex.Lock()
	defer modeChangeMutex.Unlock()
	currentMode = RunMode(mode)
}

// IsTunRunning returns whether a service is currently running
func IsTunRunning() bool {
	modeChangeMutex.RLock()
	defer modeChangeMutex.RUnlock()
	return tunRunning
}

// SetTunRunning sets the running state
func SetTunRunning(running bool) {
	modeChangeMutex.Lock()
	defer modeChangeMutex.Unlock()
	tunRunning = running
}

// HandleRunStart 处理 /run/start
// 根据当前模式启动相应的服务
func HandleRunStart(w http.ResponseWriter, r *http.Request) {
	var req struct {
		InnerToken string `json:"inner_token"`
	}
	if err := common.DecodeRequest(r, &req); err != nil {
		common.SendError(w, "Invalid request body", http.StatusBadRequest, nil)
		return
	}
	// user.SetUserToken(req.UserToken)
	user.SetInnerToken(req.InnerToken)

	modeChangeMutex.Lock()
	// 不能在已经有服务运行时再启动
	if tunRunning {
		modeChangeMutex.Unlock()
		common.SendError(w, fmt.Sprintf("%s service is already running", currentMode), http.StatusConflict, nil)
		return
	}

	startMode := currentMode
	modeChangeMutex.Unlock()

	logger.Info(fmt.Sprintf("Starting %s service...", startMode))

	// 根据运行模式启动对应的服务
	switch startMode {
	case ModeTUN:
		HandleRunTUN(w, req.InnerToken)

	case ModeHTTP:
		// HTTP模式不需要额外启动（已在swift中启动）
		common.SendResponse(w, map[string]interface{}{
			"status":  "success",
			"message": "HTTP proxy server is already running",
			"details": "HTTP proxy was started when you switched to HTTP mode",
			"port":    "127.0.0.1:56432",
		})

	default:
		common.SendError(w, fmt.Sprintf("Unknown mode: %s", startMode), http.StatusInternalServerError, nil)
	}
}

// HandleRunTUN 处理TUN模式的启动
func HandleRunTUN(w http.ResponseWriter, innerToken string) {
	modeChangeMutex.Lock()
	tunRunning = true
	modeChangeMutex.Unlock()

	go runner2.Start()
	res := <-runner2.RunStatusChan

	// Update mode based on result
	modeChangeMutex.Lock()
	if status, ok := res["status"]; ok && status == "failed" {
		tunRunning = false
	}
	modeChangeMutex.Unlock()

	common.SendResponse(w, res)
}

// HandleRunStop 处理 /run/stop
func HandleRunStop(w http.ResponseWriter, r *http.Request) {
	modeChangeMutex.Lock()
	if !tunRunning {
		modeChangeMutex.Unlock()
		common.SendError(w, "No service is currently running", http.StatusBadRequest, nil)
		return
	}

	stoppedMode := currentMode
	tunRunning = false
	modeChangeMutex.Unlock()

	logger.Info(fmt.Sprintf("Stopping %s service...", stoppedMode))

	response := map[string]interface{}{
		"status":       "success",
		"message":      fmt.Sprintf("%s service stopped successfully", stoppedMode),
		"stopped_mode": stoppedMode,
	}

	// Stop the appropriate service based on what was running
	switch stoppedMode {
	case ModeHTTP:
		logger.Info("Stopping HTTP proxy server...")
		httpServer.StopHttpProxy()
		response["details"] = "HTTP proxy server on 127.0.0.1:56432 has been stopped"

	case ModeTUN:
		logger.Info("Stopping TUN service...")
		tun.Stop()
		response["details"] = "TUN interface service has been stopped"
	}

	common.SendResponse(w, response)
}

// HandleRunUserInfo 处理 /run/userInfo
func HandleRunUserInfo(w http.ResponseWriter, r *http.Request) {
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

// HandleRunStatus 处理 /run/status - 查询当前运行状态
func HandleRunStatus(w http.ResponseWriter, r *http.Request) {
	modeChangeMutex.RLock()
	defer modeChangeMutex.RUnlock()

	response := map[string]interface{}{
		"current_mode": currentMode,
		"tun_running":  tunRunning,
		"available_modes": []string{
			string(ModeHTTP),
			string(ModeTUN),
		},
	}

	// Add detailed status based on current mode
	switch currentMode {
	case ModeTUN:
		if tunRunning {
			response["status"] = "TUN service is running"
			response["description"] = "Transparent proxy mode via TUN interface"
		} else {
			response["status"] = "TUN mode selected, service not running"
			response["description"] = "TUN mode is ready, call /run/start to activate"
		}
	case ModeHTTP:
		if tunRunning {
			response["status"] = "HTTP proxy server is running"
			response["description"] = "HTTP CONNECT proxy mode on port 56432"
		} else {
			response["status"] = "HTTP mode selected, service not running"
			response["description"] = "HTTP mode is ready, service will start automatically"
		}
	}

	common.SendResponse(w, response)
}

// HandleRunSwift 处理 /run/swift - 切换运行模式
// 请求体: { "target_mode": "http" 或 "tun" }
// 逻辑: 先停止当前服务（如果有），再启动新模式的服务
func HandleRunSwift(w http.ResponseWriter, r *http.Request) {
	var req struct {
		TargetMode string `json:"target_mode"`
	}

	if err := common.DecodeRequest(r, &req); err != nil {
		common.SendError(w, "Invalid request body", http.StatusBadRequest, nil)
		return
	}

	targetMode := RunMode(req.TargetMode)

	// Validate target mode
	if targetMode != ModeHTTP && targetMode != ModeTUN {
		common.SendError(w, fmt.Sprintf("Invalid target mode: %s. Must be 'http' or 'tun'", req.TargetMode), http.StatusBadRequest, nil)
		return
	}

	modeChangeMutex.Lock()

	// Check if already in target mode and running
	if currentMode == targetMode && tunRunning {
		modeChangeMutex.Unlock()
		common.SendResponse(w, map[string]interface{}{
			"status":       "already_running",
			"current_mode": currentMode,
			"message":      fmt.Sprintf("Already running in %s mode", currentMode),
		})
		return
	}

	// If switching from a different mode, stop the current service first
	previousMode := currentMode

	if currentMode != targetMode && tunRunning {
		logger.Info(fmt.Sprintf("Stopping %s service before switching to %s mode...", previousMode, targetMode))
		tunRunning = false
		modeChangeMutex.Unlock()

		// Stop the previous service
		stopService(previousMode)

		// Wait a moment for cleanup
		// time.Sleep(500 * time.Millisecond)

		modeChangeMutex.Lock()
	}

	// Set new mode
	currentMode = targetMode
	// Don't set tunRunning to true yet - wait for actual service start
	modeChangeMutex.Unlock()

	logger.Info(fmt.Sprintf("Switching to %s mode", targetMode))

	response := map[string]interface{}{
		"status":      "switched",
		"target_mode": targetMode,
	}

	// Start the appropriate service based on target mode
	switch targetMode {
	case ModeHTTP:
		// Start HTTP proxy server in a goroutine
		go func() {
			logger.Info("Starting HTTP proxy server...")
			modeChangeMutex.Lock()
			tunRunning = true
			modeChangeMutex.Unlock()

			httpServer.StartMitmHttp()

			// If HTTP server exits, reset state
			modeChangeMutex.Lock()
			tunRunning = false
			modeChangeMutex.Unlock()
		}()
		response["message"] = "Switched to HTTP proxy mode. Server is starting on 127.0.0.1:56432"
		response["usage"] = "curl -x http://127.0.0.1:56432 https://example.com"
		response["details"] = "HTTP proxy server will be ready in a moment"
		response["next_action"] = "HTTP service starts automatically, you can begin using it after 1 second"

	case ModeTUN:
		// For TUN mode, we don't start automatically - user must call /run/start
		response["message"] = "Switched to TUN mode. Use /run/start to activate the TUN service"
		response["usage"] = "POST /run/start with InnerToken"
		response["next_step"] = "Call /run/start to initialize and start the TUN interface"
	}

	common.SendResponse(w, response)
}

// stopService 停止指定的服务
func stopService(mode RunMode) {
	switch mode {
	case ModeHTTP:
		logger.Info("Stopping HTTP proxy server...")
		httpServer.StopHttpProxy()
		logger.Info("HTTP proxy server stopped")

	case ModeTUN:
		logger.Info("Stopping TUN service...")
		tun.Stop()
		logger.Info("TUN service stopped")
	}
}

// RegisterRunRoutes 注册Run相关路由
func RegisterRunRoutes() {
	http.HandleFunc("/run/start", HandleRunStart)
	http.HandleFunc("/run/stop", HandleRunStop)
	http.HandleFunc("/run/userInfo", HandleRunUserInfo)
	http.HandleFunc("/run/status", HandleRunStatus)
	http.HandleFunc("/run/swift", HandleRunSwift)
}
