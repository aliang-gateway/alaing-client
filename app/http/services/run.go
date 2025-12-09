package services

import (
	"sync"

	"nursor.org/nursorgate/app/http/models"
	"nursor.org/nursorgate/common/logger"
	httpServer "nursor.org/nursorgate/inbound/http"
	tun "nursor.org/nursorgate/inbound/tun/engine"
	runner2 "nursor.org/nursorgate/inbound/tun/runner"
	user "nursor.org/nursorgate/processor/auth"
)

// RunService handles run/mode operations
type RunService struct {
	modeChangeMutex sync.RWMutex
	currentMode     models.RunMode
	tunRunning      bool
}

// NewRunService creates a new run service instance
func NewRunService() *RunService {
	return &RunService{
		currentMode: models.ModeHTTP,
		tunRunning:  false,
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

// IsTunRunning returns whether a service is currently running
func (rs *RunService) IsTunRunning() bool {
	rs.modeChangeMutex.RLock()
	defer rs.modeChangeMutex.RUnlock()
	return rs.tunRunning
}

// SetTunRunning sets the running state
func (rs *RunService) SetTunRunning(running bool) {
	rs.modeChangeMutex.Lock()
	defer rs.modeChangeMutex.Unlock()
	rs.tunRunning = running
}

// StartService starts the service for the current mode
func (rs *RunService) StartService() map[string]interface{} {
	rs.modeChangeMutex.Lock()

	// Check if already running
	if rs.tunRunning {
		rs.modeChangeMutex.Unlock()
		return map[string]interface{}{
			"error":  "already_running",
			"status": "failed",
			"msg":    "Service is already running",
		}
	}

	startMode := rs.currentMode
	rs.modeChangeMutex.Unlock()

	logger.Info("Starting " + string(startMode) + " service...")

	switch startMode {
	case models.ModeTUN:
		return rs.startTUN()
	case models.ModeHTTP:
		return map[string]interface{}{
			"status":  "success",
			"message": "HTTP proxy server is already running",
			"details": "HTTP proxy was started when you switched to HTTP mode",
			"port":    "127.0.0.1:56432",
		}
	default:
		return map[string]interface{}{
			"error":  "unknown_mode",
			"status": "failed",
			"msg":    "Unknown mode: " + string(startMode),
		}
	}
}

// startTUN handles TUN mode startup
func (rs *RunService) startTUN() map[string]interface{} {
	rs.modeChangeMutex.Lock()
	rs.tunRunning = true
	rs.modeChangeMutex.Unlock()

	go runner2.Start()
	res := <-runner2.RunStatusChan

	// Update mode based on result
	rs.modeChangeMutex.Lock()
	if status, ok := res["status"]; ok && status == "failed" {
		rs.tunRunning = false
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

	if !rs.tunRunning {
		rs.modeChangeMutex.Unlock()
		return map[string]interface{}{
			"error":  "not_running",
			"status": "failed",
			"msg":    "No service is currently running",
		}
	}

	stoppedMode := rs.currentMode
	rs.tunRunning = false
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

// SetUserInfo sets user information
func (rs *RunService) SetUserInfo(userUUID, innerToken, username, password string) map[string]interface{} {
	logger.SetUserInfo(innerToken)
	user.SetUsername(username)
	user.SetPassword(password)
	user.SetUserUUID(userUUID)
	logger.Info("set user info tag")

	return map[string]interface{}{
		"status":  "success",
		"user_id": user.GetUserId(),
	}
}

// GetStatus returns the current service status
func (rs *RunService) GetStatus() map[string]interface{} {
	rs.modeChangeMutex.RLock()
	defer rs.modeChangeMutex.RUnlock()

	response := map[string]interface{}{
		"current_mode": string(rs.currentMode),
		"tun_running":  rs.tunRunning,
		"available_modes": []string{
			string(models.ModeHTTP),
			string(models.ModeTUN),
		},
	}

	switch rs.currentMode {
	case models.ModeTUN:
		if rs.tunRunning {
			response["status"] = "TUN service is running"
			response["description"] = "Transparent proxy mode via TUN interface"
		} else {
			response["status"] = "TUN mode selected, service not running"
			response["description"] = "TUN mode is ready, call start to activate"
		}
	case models.ModeHTTP:
		if rs.tunRunning {
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
	if rs.currentMode == targetModeEnum && rs.tunRunning {
		return map[string]interface{}{
			"status":       "already_running",
			"current_mode": string(rs.currentMode),
			"message":      "Already running in " + string(rs.currentMode) + " mode",
		}
	}

	// Perform the mode transition
	previousMode := rs.currentMode
	if previousMode != targetModeEnum && rs.tunRunning {
		logger.Info("Stopping " + string(previousMode) + " service before switching to " + string(targetModeEnum) + " mode...")
		rs.tunRunning = false
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
		response["message"] = "Switched to HTTP proxy mode. Server is starting on 127.0.0.1:56432"
		response["usage"] = "curl -x http://127.0.0.1:56432 https://example.com"
		response["details"] = "HTTP proxy server will be ready in a moment"
		response["next_action"] = "HTTP service starts automatically, you can begin using it after 1 second"

		// Start HTTP service in background
		go func() {
			logger.Info("Starting HTTP proxy server...")
			rs.modeChangeMutex.Lock()
			rs.tunRunning = true
			rs.modeChangeMutex.Unlock()

			httpServer.StartMitmHttp()

			rs.modeChangeMutex.Lock()
			rs.tunRunning = false
			rs.modeChangeMutex.Unlock()
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
