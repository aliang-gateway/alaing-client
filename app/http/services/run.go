package services

import (
	"sync"

	"nursor.org/nursorgate/app/http/models"
	"nursor.org/nursorgate/common/logger"
	httpServer "nursor.org/nursorgate/inbound/http"
	tun "nursor.org/nursorgate/inbound/tun/engine"
	runner2 "nursor.org/nursorgate/inbound/tun/runner"
	"nursor.org/nursorgate/processor/config"
)

// RunService handles run/mode operations
type RunService struct {
	modeChangeMutex sync.RWMutex
	currentMode     models.RunMode
	isRunning       bool // 统一使用 isRunning 字段
}

// NewRunService creates a new run service instance
func NewRunService() *RunService {
	return &RunService{
		currentMode: models.ModeHTTP,
		isRunning:   false,
	}
}

// GetCurrentMode returns the current operating mode
func (rs *RunService) GetCurrentMode() string {
	rs.modeChangeMutex.RLock()
	defer rs.modeChangeMutex.RUnlock()
	return string(rs.currentMode)
}

// SetCurrentMode sets the operating mode
func (rs *RunService) SetCurrentMode(mode string) {
	rs.modeChangeMutex.Lock()
	defer rs.modeChangeMutex.Unlock()
	rs.currentMode = models.RunMode(mode)
}

// IsRunning returns whether a service is currently running
func (rs *RunService) IsRunning() bool {
	rs.modeChangeMutex.RLock()
	defer rs.modeChangeMutex.RUnlock()
	return rs.isRunning
}

// SetRunning sets the running state
func (rs *RunService) SetRunning(running bool) {
	rs.modeChangeMutex.Lock()
	defer rs.modeChangeMutex.Unlock()
	rs.isRunning = running
}

// StartService starts the service for the current mode
func (rs *RunService) StartService() map[string]interface{} {
	// Check if using default configuration AND no local user info - if so, require activation
	if config.IsUsingDefaultConfig() && !config.HasLocalUserInfo() {
		return map[string]interface{}{
			"error":  "activation_required",
			"status": "failed",
			"msg":    "需要激活配置。请提供 --config 或 --token 参数。",
		}
	}

	rs.modeChangeMutex.Lock()

	// Check if already running
	if rs.isRunning {
		rs.modeChangeMutex.Unlock()
		return map[string]interface{}{
			"error":  "already_running",
			"status": "failed",
			"msg":    "Service is already running",
		}
	}

	startMode := rs.currentMode
	rs.isRunning = true // 先设置运行状态，避免并发启动
	rs.modeChangeMutex.Unlock()

	logger.Info("Starting " + string(startMode) + " service...")

	switch startMode {
	case models.ModeTUN:
		return rs.startTUN()
	case models.ModeHTTP:
		// HTTP 服务内部也有检查，但我们需要先设置状态
		go func() {
			httpServer.StartMitmHttp()
			// 如果启动失败，HTTP 服务内部会处理，但我们需要确保状态同步
			// 注意：StartMitmHttp 是阻塞的，只有在停止时才会返回
		}()
		return map[string]interface{}{
			"status":  "success",
			"message": "HTTP proxy server is starting",
			"details": "HTTP proxy server is starting on port 56432",
			"port":    "56432",
		}
	default:
		// 未知模式，回滚状态
		rs.modeChangeMutex.Lock()
		rs.isRunning = false
		rs.modeChangeMutex.Unlock()
		return map[string]interface{}{
			"error":  "unknown_mode",
			"status": "failed",
			"msg":    "Unknown mode: " + string(startMode),
		}
	}
}

// startTUN handles TUN mode startup
func (rs *RunService) startTUN() map[string]interface{} {
	// 状态已在 StartService 中设置
	go runner2.Start()
	res := <-runner2.RunStatusChan

	// Update mode based on result
	rs.modeChangeMutex.Lock()
	if status, ok := res["status"]; ok && status == "failed" {
		rs.isRunning = false // 启动失败，回滚状态
	}
	rs.modeChangeMutex.Unlock()

	// Convert map[string]string to map[string]interface{}
	result := make(map[string]interface{})
	for k, v := range res {
		result[k] = v
	}
	return result
}

// StopService stops the current running service
func (rs *RunService) StopService() map[string]interface{} {
	rs.modeChangeMutex.Lock()

	if !rs.isRunning {
		rs.modeChangeMutex.Unlock()
		return map[string]interface{}{
			"error":  "not_running",
			"status": "failed",
			"msg":    "No service is currently running",
		}
	}

	stoppedMode := rs.currentMode
	rs.isRunning = false
	rs.modeChangeMutex.Unlock()

	logger.Info("Stopping " + string(stoppedMode) + " service...")

	response := map[string]interface{}{
		"status":       "success",
		"message":      string(stoppedMode) + " service stopped successfully",
		"stopped_mode": stoppedMode,
	}

	switch stoppedMode {
	case models.ModeHTTP:
		logger.Info("Stopping HTTP proxy server...")
		httpServer.StopHttpProxy()
		response["details"] = "HTTP proxy server on 127.0.0.1:56432 has been stopped"

	case models.ModeTUN:
		logger.Info("Stopping TUN service...")
		tun.Stop()
		response["details"] = "TUN interface service has been stopped"
	}

	return response
}

// GetStatus returns the current service status
func (rs *RunService) GetStatus() map[string]interface{} {
	rs.modeChangeMutex.RLock()
	defer rs.modeChangeMutex.RUnlock()

	response := map[string]interface{}{
		"current_mode": string(rs.currentMode),
		"is_running":   rs.isRunning,
		"available_modes": []string{
			string(models.ModeHTTP),
			string(models.ModeTUN),
		},
	}

	switch rs.currentMode {
	case models.ModeTUN:
		if rs.isRunning {
			response["status"] = "TUN service is running"
			response["description"] = "Transparent proxy mode via TUN interface"
		} else {
			response["status"] = "TUN mode selected, service not running"
			response["description"] = "TUN mode is ready, call start to activate"
		}
	case models.ModeHTTP:
		if rs.isRunning {
			response["status"] = "HTTP proxy server is running"
			response["description"] = "HTTP CONNECT proxy mode on port 56432"
		} else {
			response["status"] = "HTTP mode selected, service not running"
			response["description"] = "HTTP mode is ready, service will start automatically"
		}
	}

	return response
}

// SwitchMode switches the operating mode
func (rs *RunService) SwitchMode(targetMode string) map[string]interface{} {
	rs.modeChangeMutex.Lock()
	defer rs.modeChangeMutex.Unlock()

	targetModeEnum := models.RunMode(targetMode)

	// Validate target mode
	if targetModeEnum != models.ModeHTTP && targetModeEnum != models.ModeTUN {
		return map[string]interface{}{
			"error":  "invalid_mode",
			"status": "failed",
			"msg":    "Invalid target mode: " + targetMode + ". Must be 'http' or 'tun'",
		}
	}

	// Check if already in target mode and running
	if rs.currentMode == targetModeEnum && rs.isRunning {
		return map[string]interface{}{
			"status":       "already_running",
			"current_mode": string(rs.currentMode),
			"message":      "Already running in " + string(rs.currentMode) + " mode",
		}
	}

	// Perform the mode transition
	previousMode := rs.currentMode
	if previousMode != targetModeEnum && rs.isRunning {
		logger.Info("Stopping " + string(previousMode) + " service before switching to " + string(targetModeEnum) + " mode...")
		rs.isRunning = false
		rs.stopServiceSync(previousMode)
	}

	rs.currentMode = targetModeEnum

	logger.Info("Switching to " + string(targetModeEnum) + " mode")

	response := map[string]interface{}{
		"status":      "switched",
		"target_mode": targetModeEnum,
	}

	switch targetModeEnum {
	case models.ModeHTTP:
		// 检查是否已经在运行 HTTP 服务
		if rs.isRunning && previousMode == models.ModeHTTP {
			response["message"] = "Already running in HTTP proxy mode"
			response["status"] = "already_running"
			return response
		}

		response["message"] = "Switched to HTTP proxy mode. Server is starting on 127.0.0.1:56432"
		response["usage"] = "curl -x http://127.0.0.1:56432 https://example.com"
		response["details"] = "HTTP proxy server will be ready in a moment"
		response["next_action"] = "HTTP service starts automatically, you can begin using it after 1 second"

		// Start HTTP service in background
		rs.isRunning = true // 设置运行状态
		go func() {
			logger.Info("Starting HTTP proxy server...")
			// HTTP 服务内部会检查是否已经在运行，避免重复启动
			httpServer.StartMitmHttp()
			// 服务停止时，状态会在 StopService 中更新
		}()

	case models.ModeTUN:
		response["message"] = "Switched to TUN mode. Use start to activate the TUN service"
		response["usage"] = "POST /api/run/start with InnerToken"
		response["next_step"] = "Call start to initialize and start the TUN interface"
	}

	return response
}

// stopServiceSync synchronously stops a service
func (rs *RunService) stopServiceSync(mode models.RunMode) {
	switch mode {
	case models.ModeHTTP:
		logger.Info("Stopping HTTP proxy server...")
		httpServer.StopHttpProxy()
		logger.Info("HTTP proxy server stopped")

	case models.ModeTUN:
		logger.Info("Stopping TUN service...")
		tun.Stop()
		logger.Info("TUN service stopped")
	}
}
