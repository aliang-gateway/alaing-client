package handlers

import (
	"net/http"

	"nursor.org/nursorgate/app/http/common"
	authuser "nursor.org/nursorgate/processor/auth"
	"nursor.org/nursorgate/processor/runtime"
)

// StartupHandler handles system startup status queries
type StartupHandler struct{}

// NewStartupHandler creates a new startup handler instance
func NewStartupHandler() *StartupHandler {
	return &StartupHandler{}
}

// HandleStartupStatus handles GET /api/startup/status
// Returns the current system startup status and related information
func (h *StartupHandler) HandleStartupStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		common.Error(w, common.CodeBadRequest, "Method not allowed", nil)
		return
	}

	startupState := runtime.GetStartupState()
	status := startupState.GetStatus()
	fetchSuccess := startupState.GetFetchSuccess()
	userInfo := startupState.GetUserInfo()

	// Build response data
	response := map[string]interface{}{
		"status":        string(status),
		"fetch_success": fetchSuccess,
		"timestamp":     startupState.GetTimestamp().Unix(),
	}

	// Add user info if available
	if userInfo != nil {
		response["user"] = map[string]interface{}{
			"username":      userInfo.Username,
			"plan_name":     userInfo.PlanName,
			"plan_type":     userInfo.PlanType,
			"traffic_used":  userInfo.TrafficUsed,
			"traffic_total": userInfo.TrafficTotal,
			"start_time":    userInfo.StartTime,
			"end_time":      userInfo.EndTime,
		}
	}

	// Add helpful information about status transitions
	response["transitions"] = getStatusTransitionInfo(status)

	common.Success(w, response)
}

// HandleStartupDetail handles GET /api/startup/detail
// Returns detailed startup diagnostic information
func (h *StartupHandler) HandleStartupDetail(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		common.Error(w, common.CodeBadRequest, "Method not allowed", nil)
		return
	}

	startupState := runtime.GetStartupState()
	status := startupState.GetStatus()
	fetchSuccess := startupState.GetFetchSuccess()
	userInfo := startupState.GetUserInfo()

	// Check if local user info exists
	hasLocalUserInfo := false
	userInfoPath, err := authuser.GetUserInfoPath()
	if err == nil {
		// Try to load to verify existence
		_, err := authuser.LoadUserInfo()
		hasLocalUserInfo = err == nil
	}

	// Build detailed response
	response := map[string]interface{}{
		"status": map[string]interface{}{
			"current":     string(status),
			"description": getStatusDescription(status),
		},
		"fetch": map[string]interface{}{
			"success": fetchSuccess,
			"message": getFetchStatusMessage(fetchSuccess),
		},
		"user": map[string]interface{}{
			"has_local_info": hasLocalUserInfo,
			"info_path":      userInfoPath,
		},
		"diagnostics": map[string]interface{}{
			"timestamp":              startupState.GetTimestamp().Unix(),
			"suggested_next_actions": getSuggestedActions(status),
		},
	}

	// Add user details if available
	if userInfo != nil {
		response["user"].(map[string]interface{})["details"] = map[string]interface{}{
			"username":      userInfo.Username,
			"plan_name":     userInfo.PlanName,
			"plan_type":     userInfo.PlanType,
			"traffic_used":  userInfo.TrafficUsed,
			"traffic_total": userInfo.TrafficTotal,
			"start_time":    userInfo.StartTime,
			"end_time":      userInfo.EndTime,
		}
	}

	common.Success(w, response)
}

// getStatusDescription returns a human-readable description of the status
func getStatusDescription(status runtime.StartupStatus) string {
	descriptions := map[runtime.StartupStatus]string{
		runtime.UNCONFIGURED: "System awaiting configuration - no token and no local user info found",
		runtime.CONFIGURING:  "System configuring - token provided, activation in progress",
		runtime.CONFIGURED:   "System configured - user info loaded but proxyserver fetch incomplete",
		runtime.READY:        "System ready - all components initialized and proxyserver configured",
	}

	if desc, ok := descriptions[status]; ok {
		return desc
	}
	return "Unknown status"
}

// getFetchStatusMessage returns a message about fetch status
func getFetchStatusMessage(success bool) string {
	if success {
		return "Proxyserver configuration successfully fetched and applied"
	}
	return "Proxyserver configuration fetch failed - system may have limited functionality"
}

// getSuggestedActions returns suggested actions based on current status
func getSuggestedActions(status runtime.StartupStatus) []string {
	actions := map[runtime.StartupStatus][]string{
		runtime.UNCONFIGURED: {
			"POST /api/auth/activate - Activate with token",
			"GET /api/startup/status - Check status again",
		},
		runtime.CONFIGURING: {
			"GET /api/startup/status - Check activation progress",
			"Wait for token activation and proxyserver fetch to complete",
		},
		runtime.CONFIGURED: {
			"System has user info but proxyserver fetch failed",
			"Proxy functionality may be limited",
			"POST /api/auth/activate - Retry with fresh token",
		},
		runtime.READY: {
			"System ready for proxy operations",
			"GET /api/proxy/list - List available proxies",
			"GET /api/auth/userinfo - Check user information",
		},
	}

	if actions, ok := actions[status]; ok {
		return actions
	}
	return []string{}
}

// getStatusTransitionInfo returns information about possible status transitions
func getStatusTransitionInfo(status runtime.StartupStatus) map[string]interface{} {
	transitions := map[runtime.StartupStatus]map[string]interface{}{
		runtime.UNCONFIGURED: {
			"description": "Initial state - no configuration",
			"possible_transitions": []string{
				"→ CONFIGURING (token provided)",
				"→ CONFIGURED (local user info found)",
			},
		},
		runtime.CONFIGURING: {
			"description": "Activation in progress",
			"possible_transitions": []string{
				"→ READY (token activation + fetch success)",
				"→ CONFIGURED (token activation success but fetch failed)",
				"→ UNCONFIGURED (token activation failed, no local fallback)",
			},
		},
		runtime.CONFIGURED: {
			"description": "User info available but proxyserver incomplete",
			"possible_transitions": []string{
				"→ READY (proxyserver fetch succeeds, e.g., after token refresh)",
				"→ UNCONFIGURED (user info deleted)",
			},
		},
		runtime.READY: {
			"description": "System ready for proxy operations",
			"possible_transitions": []string{
				"→ CONFIGURED (proxyserver fetch fails)",
				"→ UNCONFIGURED (user logout or info deleted)",
			},
		},
	}

	if info, ok := transitions[status]; ok {
		return info
	}
	return map[string]interface{}{}
}
