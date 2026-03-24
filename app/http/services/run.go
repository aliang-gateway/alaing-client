package services

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"nursor.org/nursorgate/app/http/models"
	"nursor.org/nursorgate/common/logger"
	httpServer "nursor.org/nursorgate/inbound/http"
	tun "nursor.org/nursorgate/inbound/tun/engine"
	runner2 "nursor.org/nursorgate/inbound/tun/runner"
	"nursor.org/nursorgate/processor/config"
	"nursor.org/nursorgate/processor/routing"
	"nursor.org/nursorgate/processor/runtime"
)

var (
	activeIngressModeResolver = activeIngressModeFromSnapshot
	applyIngressModeUpdater   = applyIngressModeToSnapshot
	tunStartRunner            = defaultStartTUN
	httpStartRunner           = httpServer.StartMitmHttp
	httpStopRunner            = httpServer.StopHttpProxy
	tunStopRunner             = tun.Stop
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
	if mode, ok := activeIngressModeResolver(); ok {
		return string(mode)
	}
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
	startupState := runtime.GetStartupState()
	if !canStartProxyWithStatus(startupState.GetStatus()) {
		return map[string]interface{}{
			"error":  "activation_required",
			"status": "failed",
			"msg":    "系统尚未准备好启动代理，请先完成登录或配置恢复。",
		}
	}

	rs.modeChangeMutex.Lock()

	// Check if already running
	if rs.isRunning {
		rs.modeChangeMutex.Unlock()
		return map[string]interface{}{
			"status":  "already_running",
			"message": "Service is already running",
		}
	}

	startMode := rs.resolveAuthoritativeModeLocked()
	rs.isRunning = true // 先设置运行状态，避免并发启动
	rs.modeChangeMutex.Unlock()

	logger.Info("Starting " + string(startMode) + " service...")

	switch startMode {
	case models.ModeTUN:
		return rs.startTUN()
	case models.ModeHTTP:
		// HTTP 服务内部也有检查，但我们需要先设置状态
		go func() {
			httpStartRunner()
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

func canStartProxyWithStatus(status runtime.StartupStatus) bool {
	return status == runtime.READY || status == runtime.CONFIGURED
}

// startTUN handles TUN mode startup
func (rs *RunService) startTUN() map[string]interface{} {
	res := tunStartRunner()

	rs.modeChangeMutex.Lock()
	result := rs.handleTUNStartResultLocked(res)
	rs.modeChangeMutex.Unlock()

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
		httpStopRunner()
		response["details"] = "HTTP proxy server on 127.0.0.1:56432 has been stopped"

	case models.ModeTUN:
		logger.Info("Stopping TUN service...")
		tunStopRunner()
		response["details"] = "TUN interface service has been stopped"
	}

	return response
}

// GetStatus returns the current service status
func (rs *RunService) GetStatus() map[string]interface{} {
	rs.modeChangeMutex.RLock()
	defer rs.modeChangeMutex.RUnlock()
	mode := rs.currentMode
	if authoritativeMode, ok := activeIngressModeResolver(); ok {
		mode = authoritativeMode
	}

	response := map[string]interface{}{
		"current_mode": string(mode),
		"is_running":   rs.isRunning,
		"available_modes": []string{
			string(models.ModeHTTP),
			string(models.ModeTUN),
		},
	}

	switch mode {
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
			response["description"] = "HTTP mode is ready, call start to activate"
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

	authoritativeMode := rs.resolveAuthoritativeModeLocked()

	// Check if already in target mode and running
	if authoritativeMode == targetModeEnum && rs.isRunning {
		return map[string]interface{}{
			"status":       "already_running",
			"current_mode": string(authoritativeMode),
			"message":      "Already running in " + string(authoritativeMode) + " mode",
		}
	}

	previousMode := authoritativeMode
	wasRunning := rs.isRunning
	if previousMode != targetModeEnum && wasRunning {
		logger.Info("Stopping " + string(previousMode) + " service before switching to " + string(targetModeEnum) + " mode...")
		rs.isRunning = false
		rs.stopServiceSync(previousMode)
	}

	if err := applyIngressModeUpdater(targetModeEnum); err != nil {
		if previousMode != targetModeEnum {
			rs.rollbackToPreviousModeLocked(previousMode, wasRunning)
		}
		return map[string]interface{}{
			"error":          "switch_failed",
			"status":         "failed",
			"msg":            fmt.Sprintf("failed to activate ingress mode %s: %v", targetModeEnum, err),
			"current_mode":   string(rs.resolveAuthoritativeModeLocked()),
			"rollback_state": map[string]interface{}{"mode": string(previousMode), "running": rs.isRunning},
		}
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

		if wasRunning || targetModeEnum == models.ModeHTTP {
			rs.isRunning = true
			go func() {
				logger.Info("Starting HTTP proxy server...")
				httpStartRunner()
			}()
		}

	case models.ModeTUN:
		if wasRunning {
			tunResult := rs.handleTUNStartResultLocked(tunStartRunner())
			if status, ok := tunResult["status"].(string); ok && status == "failed" {
				rs.rollbackFromActivationFailureLocked(previousMode, tunResult)
				return map[string]interface{}{
					"error":          "switch_failed",
					"status":         "failed",
					"msg":            fmt.Sprintf("failed to start %s mode during hot switch", targetModeEnum),
					"target_result":  tunResult,
					"current_mode":   string(rs.resolveAuthoritativeModeLocked()),
					"rollback_state": map[string]interface{}{"mode": string(previousMode), "running": rs.isRunning},
				}
			}
			response["message"] = "Hot switched to TUN mode and started TUN service"
			response["next_step"] = "TUN service is active"
			break
		}

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
		httpStopRunner()
		logger.Info("HTTP proxy server stopped")

	case models.ModeTUN:
		logger.Info("Stopping TUN service...")
		tunStopRunner()
		logger.Info("TUN service stopped")
	}
}

func (rs *RunService) resolveAuthoritativeModeLocked() models.RunMode {
	if mode, ok := activeIngressModeResolver(); ok {
		rs.currentMode = mode
		return mode
	}
	return rs.currentMode
}

func (rs *RunService) rollbackToPreviousModeLocked(previousMode models.RunMode, shouldRun bool) {
	if err := applyIngressModeUpdater(previousMode); err != nil {
		logger.Error(fmt.Sprintf("Failed to roll back ingress mode to %s: %v", previousMode, err))
	}
	rs.currentMode = previousMode
	if shouldRun {
		rs.isRunning = true
		rs.startModeAsync(previousMode)
	}
}

func (rs *RunService) rollbackFromActivationFailureLocked(previousMode models.RunMode, targetResult map[string]interface{}) {
	rs.isRunning = false
	if err := applyIngressModeUpdater(previousMode); err != nil {
		logger.Error(fmt.Sprintf("Failed to roll back ingress mode after activation failure: %v", err))
	}
	rs.currentMode = previousMode
	rs.isRunning = true
	rs.startModeAsync(previousMode)
	logger.Error(fmt.Sprintf("Hot switch activation failed, rolled back to %s mode: %+v", previousMode, targetResult))
}

func (rs *RunService) startModeAsync(mode models.RunMode) {
	switch mode {
	case models.ModeHTTP:
		go func() {
			logger.Info("Starting HTTP proxy server...")
			httpStartRunner()
		}()
	case models.ModeTUN:
		go func() {
			res := rs.startTUN()
			if status, ok := res["status"].(string); ok && status == "failed" {
				logger.Error(fmt.Sprintf("Rollback TUN restart failed: %+v", res))
			}
		}()
	}
}

func activeIngressModeFromSnapshot() (models.RunMode, bool) {
	active := config.GetRoutingApplyStore().ActiveSnapshot()
	snapshot, ok := active.(*routing.RuntimeSnapshot)
	if !ok || snapshot == nil {
		return "", false
	}
	mode := models.RunMode(strings.ToLower(strings.TrimSpace(snapshot.IngressMode())))
	if mode != models.ModeHTTP && mode != models.ModeTUN {
		return "", false
	}
	return mode, true
}

func applyIngressModeToSnapshot(mode models.RunMode) error {
	store := config.GetRoutingApplyStore()
	canonical := store.ActiveCanonicalSchema()
	if canonical == nil {
		return fmt.Errorf("active routing snapshot is not initialized")
	}
	canonical.Ingress.Mode = string(mode)

	raw, err := json.Marshal(canonical)
	if err != nil {
		return fmt.Errorf("marshal canonical routing schema failed: %w", err)
	}

	_, err = store.Apply(raw, func(cfg *config.CanonicalRoutingSchema) (any, error) {
		return routing.CompileRuntimeSnapshot(cfg)
	})
	if err != nil {
		return fmt.Errorf("apply ingress mode snapshot failed: %w", err)
	}
	return nil
}

func defaultStartTUN() map[string]string {
	go runner2.Start()
	return <-runner2.RunStatusChan
}

func (rs *RunService) handleTUNStartResultLocked(res map[string]string) map[string]interface{} {
	if status, ok := res["status"]; ok {
		switch status {
		case "failed":
			rs.isRunning = false
		case "success":
			rs.isRunning = true
		}
	}

	result := make(map[string]interface{}, len(res))
	for k, v := range res {
		result[k] = v
	}
	return result
}
