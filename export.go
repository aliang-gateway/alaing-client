package main

/*
#include <stdlib.h>
#include <stdbool.h>
*/
import "C"

import (
	"encoding/json"
	"time"

	"nursor.org/nursorgate/app/http/handlers"
	"nursor.org/nursorgate/common/logger"
	"nursor.org/nursorgate/common/model"
	"nursor.org/nursorgate/inbound/http"
	httpServer "nursor.org/nursorgate/inbound/http"
	tun "nursor.org/nursorgate/inbound/tun/engine"
	runner2 "nursor.org/nursorgate/inbound/tun/runner"
	"nursor.org/nursorgate/inbound/tun/runner/utils"
	"nursor.org/nursorgate/outbound"
	user "nursor.org/nursorgate/processor/auth"
	"nursor.org/nursorgate/processor/cert/client"
	proxyConfig "nursor.org/nursorgate/processor/config"
	"nursor.org/nursorgate/processor/http2"
	proxyRegistry "nursor.org/nursorgate/processor/proxy"
)

//export startClient
func startClient() {
	// 初始化允许代理域名
	model.NewAllowProxyDomain()
	http.StartMitmHttp()
}

//export setOutboundToken
func setOutboundToken(token *C.char) {
	outbound.SetOutboundToken(C.GoString(token))
}

//export setServerHost
func setServerHost(host *C.char) {
	utils.SetServerHost(C.GoString(host))
}

//export exportCaCertToFile
func exportCaCertToFile(certPath *C.char) {
	err := client.ExportRootCaCertToFile(C.GoString(certPath))
	if err != nil {
		logger.Error(err.Error())
		return
	}
}

//export getToCursorDomain
func getToCursorDomain() *C.char {
	jsonStr, _ := json.Marshal(model.NewAllowProxyDomain())
	return C.CString(string(jsonStr))
}

//export runGate
func runGate(innerToken *C.char) *C.char {
	innerTokenStr := C.GoString(innerToken)
	user.SetInnerToken(innerTokenStr)
	logger.SetUserInfo(innerTokenStr)
	model.NewAllowProxyDomain()
	utils.SetServerHost("api2.nursor.org:12235")
	go runner2.Start()
	res := <-runner2.RunStatusChan
	logger.Info(res)
	resStr, _ := json.Marshal(res)
	return C.CString(string(resStr))
}

//export setUserInfo
func setUserInfo(innerToken *C.char, username *C.char, password *C.char, userUUID *C.char) {
	innerTokenStr := C.GoString(innerToken)
	usernameStr := C.GoString(username)
	passwordStr := C.GoString(password)
	userUUIDStr := C.GoString(userUUID)
	user.SetUsername(usernameStr)
	user.SetPassword(passwordStr)
	user.SetInnerToken(innerTokenStr)
	user.SetUserUUID(userUUIDStr)
	logger.SetUserInfo(innerTokenStr)
}

//export setLogWatchMode
func setLogWatchMode(enableWatch *C.bool, level *C.int) {
	watchMode := *enableWatch != C.bool(false)
	http2.IsWatcherAllowed = watchMode
	logLevel := int(*level)
	logger.SetHttpLogLevel(logger.LogLevel(logLevel))
	logger.SetLogLevel(logger.LogLevel(logLevel))
}

//export setCursorGateMode
func setCursorGateMode(enableCursorGate *C.bool) {
	cursorMode := *enableCursorGate != C.bool(false)
	http2.IsCursorProxyEnabled = cursorMode
}

//export stopGate
func stopGate() {
	runner2.Stop()
}

//export registerProxy
func registerProxy(name *C.char, configJSON *C.char) *C.char {
	nameStr := C.GoString(name)
	jsonStr := C.GoString(configJSON)

	// Parse JSON config
	var cfg proxyConfig.ProxyConfig
	if err := json.Unmarshal([]byte(jsonStr), &cfg); err != nil {
		errMsg := "Failed to parse config JSON: " + err.Error()
		logger.Error(errMsg)
		return C.CString("{\"error\": \"" + errMsg + "\"}")
	}

	// Validate config
	if err := cfg.Validate(); err != nil {
		errMsg := "Invalid config: " + err.Error()
		logger.Error(errMsg)
		return C.CString("{\"error\": \"" + errMsg + "\"}")
	}

	// Register proxy
	if err := proxyRegistry.GetRegistry().RegisterFromConfig(nameStr, &cfg); err != nil {
		errMsg := "Failed to register proxy: " + err.Error()
		logger.Error(errMsg)
		return C.CString("{\"error\": \"" + errMsg + "\"}")
	}

	return C.CString("{\"status\": \"success\"}")
}

//export switchProxy
func switchProxy(name *C.char) *C.char {
	nameStr := C.GoString(name)

	// Set as default proxy
	if err := proxyRegistry.GetRegistry().SetDefault(nameStr); err != nil {
		errMsg := "Failed to switch proxy: " + err.Error()
		logger.Error(errMsg)
		return C.CString("{\"error\": \"" + errMsg + "\"}")
	}

	return C.CString("{\"status\": \"success\"}")
}

//export listProxies
func listProxies() *C.char {
	info := proxyRegistry.GetRegistry().ListWithInfo()
	jsonStr, _ := json.Marshal(info)
	return C.CString(string(jsonStr))
}

//export getCurrentProxy
func getCurrentProxy() *C.char {
	registry := proxyRegistry.GetRegistry()
	currentName := registry.GetDefaultName()
	proxy, err := registry.GetDefault()

	if err != nil {
		return C.CString("{\"error\": \"No proxy set\"}")
	}

	result := map[string]interface{}{
		"name": currentName,
		"type": proxy.Proto().String(),
		"addr": proxy.Addr(),
	}
	jsonStr, _ := json.Marshal(result)
	return C.CString(string(jsonStr))
}

//export setCurrentProxy
func setCurrentProxy(name *C.char) *C.char {
	nameStr := C.GoString(name)

	if nameStr == "" {
		return C.CString("{\"error\": \"name is required\"}")
	}

	registry := proxyRegistry.GetRegistry()
	if err := registry.SetDefault(nameStr); err != nil {
		errMsg := "Failed to set proxy: " + err.Error()
		return C.CString("{\"error\": \"" + errMsg + "\"}")
	}

	proxy, _ := registry.GetDefault()
	result := map[string]interface{}{
		"name": nameStr,
		"type": proxy.Proto().String(),
		"addr": proxy.Addr(),
	}
	jsonStr, _ := json.Marshal(result)
	return C.CString(string(jsonStr))
}

//export getLogsJSON
func getLogsJSON(limit C.int, levelStr *C.char) *C.char {
	limitInt := int(limit)
	levelString := C.GoString(levelStr)

	// Convert level string to LogLevelType
	var level logger.LogLevelType
	switch levelString {
	case "TRACE":
		level = logger.TRACE
	case "DEBUG":
		level = logger.DEBUG
	case "INFO":
		level = logger.INFO
	case "WARN":
		level = logger.WARN
	case "ERROR":
		level = logger.ERROR
	case "FATAL":
		level = logger.FATAL
	case "PANIC":
		level = logger.PANIC
	default:
		level = 0 // All levels
	}

	// Get logs from buffer
	entries := logger.GetBufferEntries(limitInt, level, "")

	// Convert to response format
	var logEntries []map[string]interface{}
	for _, entry := range entries {
		logEntries = append(logEntries, map[string]interface{}{
			"level":     levelToStringFFI(entry.Level),
			"timestamp": entry.Timestamp.Format("2006-01-02 15:04:05.000"),
			"message":   entry.Message,
			"source":    entry.Source,
		})
	}

	response := map[string]interface{}{
		"code": 0,
		"msg":  "success",
		"data": map[string]interface{}{
			"entries": logEntries,
			"count":   len(logEntries),
		},
	}

	jsonStr, _ := json.Marshal(response)
	return C.CString(string(jsonStr))
}

//export clearLogs
func clearLogs() {
	logger.ClearBuffer()
}

//export setLogLevel
func setLogLevel(levelStr *C.char) *C.char {
	levelString := C.GoString(levelStr)

	// Convert level string to LogLevelType
	var level logger.LogLevelType
	switch levelString {
	case "TRACE":
		level = logger.TRACE
	case "DEBUG":
		level = logger.DEBUG
	case "INFO":
		level = logger.INFO
	case "WARN":
		level = logger.WARN
	case "ERROR":
		level = logger.ERROR
	case "FATAL":
		level = logger.FATAL
	case "PANIC":
		level = logger.PANIC
	default:
		return C.CString("{\"error\": \"Invalid log level\"}")
	}

	// Update configuration
	logger.UpdateLogLevel(level)

	response := map[string]interface{}{
		"code": 0,
		"msg":  "success",
		"data": map[string]string{"level": levelString},
	}

	jsonStr, _ := json.Marshal(response)
	return C.CString(string(jsonStr))
}

// Helper function to convert LogLevelType to string for FFI
func levelToStringFFI(level logger.LogLevelType) string {
	switch level {
	case logger.TRACE:
		return "TRACE"
	case logger.DEBUG:
		return "DEBUG"
	case logger.INFO:
		return "INFO"
	case logger.WARN:
		return "WARN"
	case logger.ERROR:
		return "ERROR"
	case logger.FATAL:
		return "FATAL"
	case logger.PANIC:
		return "PANIC"
	default:
		return "UNKNOWN"
	}
}

//export getLogConfigJSON
func getLogConfigJSON() *C.char {
	config := logger.GetLogConfig()

	configData := map[string]interface{}{
		"level":              levelToStringFFI(config.Level),
		"errorWindow":        config.ErrorWindow.String(),
		"maxErrorCount":      config.MaxErrorCount,
		"cleanupInterval":    config.CleanupInterval.String(),
		"fileLogPath":        config.FileLogPath,
		"enableFileRotation": config.EnableFileRotation,
		"maxLogSize":         config.MaxLogSize,
		"maxLogBackups":      config.MaxLogBackups,
		"sentryDSN":          maskSensitiveDataFFI(config.SentryDSN),
		"enableSentry":       config.EnableSentry,
	}

	response := map[string]interface{}{
		"code": 0,
		"msg":  "success",
		"data": configData,
	}

	jsonStr, _ := json.Marshal(response)
	return C.CString(string(jsonStr))
}

//export setLogConfigJSON
func setLogConfigJSON(configJSON *C.char) *C.char {
	jsonStr := C.GoString(configJSON)

	var req struct {
		Level              string `json:"level,omitempty"`
		ErrorWindow        string `json:"errorWindow,omitempty"`
		MaxErrorCount      int    `json:"maxErrorCount,omitempty"`
		CleanupInterval    string `json:"cleanupInterval,omitempty"`
		FileLogPath        string `json:"fileLogPath,omitempty"`
		EnableFileRotation bool   `json:"enableFileRotation,omitempty"`
		MaxLogSize         int64  `json:"maxLogSize,omitempty"`
		MaxLogBackups      int    `json:"maxLogBackups,omitempty"`
		SentryDSN          string `json:"sentryDSN,omitempty"`
		EnableSentry       bool   `json:"enableSentry,omitempty"`
	}

	if err := json.Unmarshal([]byte(jsonStr), &req); err != nil {
		errResp := map[string]interface{}{
			"code": 1,
			"msg":  "Failed to parse config JSON: " + err.Error(),
		}
		jsonResp, _ := json.Marshal(errResp)
		return C.CString(string(jsonResp))
	}

	config := logger.GetLogConfig()
	updated := false

	// Update level if provided
	if req.Level != "" {
		var level logger.LogLevelType
		switch req.Level {
		case "TRACE":
			level = logger.TRACE
		case "DEBUG":
			level = logger.DEBUG
		case "INFO":
			level = logger.INFO
		case "WARN":
			level = logger.WARN
		case "ERROR":
			level = logger.ERROR
		case "FATAL":
			level = logger.FATAL
		case "PANIC":
			level = logger.PANIC
		default:
			errResp := map[string]interface{}{
				"code": 1,
				"msg":  "Invalid log level: " + req.Level,
			}
			jsonResp, _ := json.Marshal(errResp)
			return C.CString(string(jsonResp))
		}
		config.Level = level
		updated = true
	}

	// Update error window if provided
	if req.ErrorWindow != "" {
		d, err := time.ParseDuration(req.ErrorWindow)
		if err != nil {
			errResp := map[string]interface{}{
				"code": 1,
				"msg":  "Invalid errorWindow duration format: " + err.Error(),
			}
			jsonResp, _ := json.Marshal(errResp)
			return C.CString(string(jsonResp))
		}
		config.ErrorWindow = d
		updated = true
	}

	// Update max error count if provided
	if req.MaxErrorCount > 0 {
		config.MaxErrorCount = req.MaxErrorCount
		updated = true
	}

	// Update cleanup interval if provided
	if req.CleanupInterval != "" {
		d, err := time.ParseDuration(req.CleanupInterval)
		if err != nil {
			errResp := map[string]interface{}{
				"code": 1,
				"msg":  "Invalid cleanupInterval duration format: " + err.Error(),
			}
			jsonResp, _ := json.Marshal(errResp)
			return C.CString(string(jsonResp))
		}
		config.CleanupInterval = d
		updated = true
	}

	// Update file log path if provided
	if req.FileLogPath != "" {
		config.FileLogPath = req.FileLogPath
		updated = true
	}

	// Update enable file rotation if provided
	if req.EnableFileRotation {
		config.EnableFileRotation = req.EnableFileRotation
		updated = true
	}

	// Update max log size if provided
	if req.MaxLogSize > 0 {
		config.MaxLogSize = req.MaxLogSize
		updated = true
	}

	// Update max log backups if provided
	if req.MaxLogBackups > 0 {
		config.MaxLogBackups = req.MaxLogBackups
		updated = true
	}

	// Update Sentry DSN if provided
	if req.SentryDSN != "" {
		config.SentryDSN = req.SentryDSN
		config.EnableSentry = true
		updated = true
	}

	// Update enable Sentry if explicitly set
	if req.EnableSentry {
		config.EnableSentry = req.EnableSentry
		updated = true
	}

	if !updated {
		errResp := map[string]interface{}{
			"code": 1,
			"msg":  "No valid configuration fields provided",
		}
		jsonResp, _ := json.Marshal(errResp)
		return C.CString(string(jsonResp))
	}

	// Save updated config
	logger.SetLogConfig(config)

	response := map[string]interface{}{
		"code": 0,
		"msg":  "success",
		"data": map[string]string{"status": "config updated"},
	}

	respBytes, _ := json.Marshal(response)
	return C.CString(string(respBytes))
}

// Helper function to mask sensitive data for FFI
func maskSensitiveDataFFI(data string) string {
	if data == "" {
		return ""
	}
	if len(data) <= 8 {
		return "***"
	}
	return data[:8] + "***"
}

// ============================================================================
// Run Control FFI Exports - Core functionality exposed directly
// ============================================================================

//export runStart
func runStart(innerToken *C.char) *C.char {
	innerTokenStr := C.GoString(innerToken)
	user.SetInnerToken(innerTokenStr)

	// 根据当前模式启动对应的服务
	// 这里实现核心逻辑，不通过HTTP
	result := make(map[string]interface{})

	switch handlers.GetCurrentMode() {
	case "http":
		result["status"] = "success"
		result["message"] = "HTTP proxy server is already running"
		result["details"] = "HTTP proxy was started when you switched to HTTP mode"
		result["port"] = "127.0.0.1:56432"

	case "tun":
		// 启动TUN服务
		go runner2.Start()
		res := <-runner2.RunStatusChan
		result = map[string]interface{}{
			"status":  res["status"],
			"message": res["message"],
			"details": res["details"],
		}

	default:
		result["status"] = "error"
		result["message"] = "No mode selected"
	}

	resultJSON, _ := json.Marshal(result)
	return C.CString(string(resultJSON))
}

//export runStop
func runStop() *C.char {
	// 停止当前运行的服务
	result := make(map[string]interface{})

	currentMode := handlers.GetCurrentMode()
	if !handlers.IsTunRunning() {
		result["status"] = "error"
		result["message"] = "No service is currently running"
	} else {
		handlers.SetTunRunning(false)

		switch currentMode {
		case "http":
			logger.Info("Stopping HTTP proxy server...")
			httpServer.StopHttpProxy()
			result["status"] = "success"
			result["message"] = "http service stopped successfully"
			result["stopped_mode"] = "http"
			result["details"] = "HTTP proxy server on 127.0.0.1:56432 has been stopped"

		case "tun":
			logger.Info("Stopping TUN service...")
			tun.Stop()
			result["status"] = "success"
			result["message"] = "tun service stopped successfully"
			result["stopped_mode"] = "tun"
			result["details"] = "TUN interface service has been stopped"

		default:
			result["status"] = "error"
			result["message"] = "Unknown mode"
		}
	}

	resultJSON, _ := json.Marshal(result)
	return C.CString(string(resultJSON))
}

//export runStatus
func runStatus() *C.char {
	response := make(map[string]interface{})
	currentMode := handlers.GetCurrentMode()
	tunRunning := handlers.IsTunRunning()

	response["current_mode"] = currentMode
	response["tun_running"] = tunRunning
	response["available_modes"] = []string{"http", "tun"}

	// Add detailed status based on current mode
	switch currentMode {
	case "tun":
		if tunRunning {
			response["status"] = "TUN service is running"
			response["description"] = "Transparent proxy mode via TUN interface"
		} else {
			response["status"] = "TUN mode selected, service not running"
			response["description"] = "TUN mode is ready, call runStart to activate"
		}
	case "http":
		if tunRunning {
			response["status"] = "HTTP proxy server is running"
			response["description"] = "HTTP CONNECT proxy mode on port 56432"
		} else {
			response["status"] = "HTTP mode selected, service not running"
			response["description"] = "HTTP mode is ready, service will start automatically"
		}
	}

	resultJSON, _ := json.Marshal(response)
	return C.CString(string(resultJSON))
}

//export runSwift
func runSwift(targetMode *C.char) *C.char {
	targetModeStr := C.GoString(targetMode)

	// Validate target mode
	if targetModeStr != "http" && targetModeStr != "tun" {
		errResp := map[string]interface{}{
			"status":  "error",
			"message": "Invalid target mode: " + targetModeStr + ". Must be 'http' or 'tun'",
		}
		errJSON, _ := json.Marshal(errResp)
		return C.CString(string(errJSON))
	}

	response := make(map[string]interface{})
	currentMode := handlers.GetCurrentMode()

	// Check if already in target mode and running
	if currentMode == targetModeStr && handlers.IsTunRunning() {
		response["status"] = "already_running"
		response["current_mode"] = currentMode
		response["message"] = "Already running in " + currentMode + " mode"
	} else {
		// If switching from a different mode, stop the current service first
		if currentMode != targetModeStr && handlers.IsTunRunning() {
			logger.Info("Stopping " + currentMode + " service before switching to " + targetModeStr + " mode...")
			handlers.SetTunRunning(false)

			// Stop the previous service
			stopServiceCore(currentMode)
		}

		// Set new mode
		handlers.SetCurrentMode(targetModeStr)

		logger.Info("Switching to " + targetModeStr + " mode")

		response["status"] = "switched"
		response["target_mode"] = targetModeStr

		// Start the appropriate service based on target mode
		if targetModeStr == "http" {
			// Start HTTP proxy server in a goroutine
			go func() {
				logger.Info("Starting HTTP proxy server...")
				handlers.SetTunRunning(true)

				httpServer.StartMitmHttp()

				// If HTTP server exits, reset state
				handlers.SetTunRunning(false)
			}()
			response["message"] = "Switched to HTTP proxy mode. Server is starting on 127.0.0.1:56432"
			response["usage"] = "curl -x http://127.0.0.1:56432 https://example.com"
			response["details"] = "HTTP proxy server will be ready in a moment"
			response["next_action"] = "HTTP service starts automatically, you can begin using it after 1 second"

		} else if targetModeStr == "tun" {
			// For TUN mode, we don't start automatically - user must call runStart
			response["message"] = "Switched to TUN mode. Use runStart to activate the TUN service"
			response["usage"] = "Call runStart with innerToken"
			response["next_step"] = "Call runStart to initialize and start the TUN interface"
		}
	}

	resultJSON, _ := json.Marshal(response)
	return C.CString(string(resultJSON))
}

//export runSetUserInfo
func runSetUserInfo(innerToken *C.char, username *C.char, password *C.char, userUUID *C.char) *C.char {
	innerTokenStr := C.GoString(innerToken)
	usernameStr := C.GoString(username)
	passwordStr := C.GoString(password)
	userUUIDStr := C.GoString(userUUID)

	user.SetUsername(usernameStr)
	user.SetPassword(passwordStr)
	user.SetInnerToken(innerTokenStr)
	user.SetUserUUID(userUUIDStr)
	logger.SetUserInfo(innerTokenStr)

	result := map[string]interface{}{
		"status":  "success",
		"user_id": user.GetUserId(),
	}

	resultJSON, _ := json.Marshal(result)
	return C.CString(string(resultJSON))
}

// Helper function to stop a service
func stopServiceCore(mode string) {
	switch mode {
	case "http":
		logger.Info("Stopping HTTP proxy server...")
		httpServer.StopHttpProxy()
		logger.Info("HTTP proxy server stopped")

	case "tun":
		logger.Info("Stopping TUN service...")
		tun.Stop()
		logger.Info("TUN service stopped")
	}
}

// main 函数仅用于测试，实际使用时应该通过 FFI 调用导出的函数
// 如果要编译命令行工具，请使用: go build ./cmd/nursor
func main() {
	panic("test")
}
