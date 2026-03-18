package middleware

import (
	"fmt"
	"net/http"
	"strings"

	"nursor.org/nursorgate/app/http/common"
	"nursor.org/nursorgate/processor/runtime"
)

// StartupStatusMiddleware checks the system's startup status and gates API access accordingly
// Different API endpoints are allowed based on current startup status:
// - Configuration APIs (token activation, logout, etc.) are always allowed
// - Proxy APIs are only allowed when status is READY
// - Status query API is always allowed
func StartupStatusMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path

		// White-list: Configuration-related APIs are always allowed
		if isConfigurationAPI(path) {
			next.ServeHTTP(w, r)
			return
		}

		// Status query API is always allowed
		if path == "/api/run/status" {
			next.ServeHTTP(w, r)
			return
		}

		// All other APIs require READY status
		startupState := runtime.GetStartupState()
		status := startupState.GetStatus()

		if status != runtime.READY {
			// System not ready for proxy operations
			if path == "/api/run/start" {
				respondSystemNotReady(w, status)
				return
			}
		}

		next.ServeHTTP(w, r)
	})
}

// isConfigurationAPI checks if the request path is a configuration-related API
// Configuration APIs are allowed regardless of startup status to enable system initialization
func isConfigurationAPI(path string) bool {
	// Configuration endpoints that should always be accessible
	configEndpoints := []string{
		"/api/auth/activate",       // Token activation
		"/api/auth/userinfo",       // User info retrieval
		"/api/auth/logout",         // Logout
		"/api/auth/refresh-status", // Refresh status (diagnostic)
		"/api/token/get",           // Get token
		"/api/token/set",           // Set token
		"/api/software-config/save",
		"/api/software-config/activate",
		"/api/software-config/cloud/push",
		"/api/software-config/cloud/pull",
		"/api/logs", // Logs access (for diagnosis)
		"/api/run/status",
	}

	for _, endpoint := range configEndpoints {
		if path == endpoint {
			return true
		}
	}

	// Partial match for paths with parameters
	if strings.HasPrefix(path, "/api/logs/") {
		return true
	}

	return false
}

// respondSystemNotReady sends a 503 Service Unavailable response when system is not ready
func respondSystemNotReady(w http.ResponseWriter, status runtime.StartupStatus) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusServiceUnavailable)

	statusStr := string(status)
	errorMsg := fmt.Sprintf("System not ready for proxy operations (status: %s)", statusStr)
	suggestedActions := []string{
		"POST /api/auth/activate - Activate with token",
		"GET /api/run/status - Check system status",
	}

	common.Error(w, common.CodeServiceUnavailable, errorMsg, map[string]interface{}{
		"status":            statusStr,
		"suggested_actions": suggestedActions,
	})
}
