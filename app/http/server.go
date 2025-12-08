package http

import (
	"fmt"
	"log"
	"net/http"

	"nursor.org/nursorgate/app/http/middleware"
	"nursor.org/nursorgate/app/http/routes"
	"nursor.org/nursorgate/common/logger"
)

var (
	// mux is the custom request multiplexer for applying middleware
	mux *http.ServeMux
)

// StartHttpServer 启动HTTP服务器，注册所有路由
func StartHttpServer() {
	// 定义 HTTP 服务端口
	port := "127.0.0.1:56431"

	// Initialize custom mux
	mux = http.NewServeMux()

	// 注册所有路由
	registerAllRoutes()

	// 启动 HTTP 服务（非阻塞）
	go func() {
		logger.Info(fmt.Sprintf("Starting HTTP server on %s...\n", port))

		// Wrap mux with middleware stack
		middlewares := middleware.GetDefaultMiddleware()
		wrappedMux := middleware.Chain(mux, middlewares...)

		err := http.ListenAndServe(port, wrappedMux)
		if err != nil {
			log.Fatalf("HTTP server failed: %v", err)
		}
	}()

	// 保持主线程运行
	select {}
}

// registerAllRoutes 注册所有HTTP路由
func registerAllRoutes() {
	// Create all handlers with dependency injection
	handlers := routes.NewHandlers()

	// Register all feature-grouped routes (using custom mux)
	registerRoutesWithMux(handlers)
}

// registerRoutesWithMux registers routes with custom mux instead of default http mux
func registerRoutesWithMux(h *routes.Handlers) {
	// Logger routes (/api/logs/*)
	mux.HandleFunc("/api/logs", h.Logger.HandleGetLogs)
	mux.HandleFunc("/api/logs/clear", h.Logger.HandleClearLogs)
	mux.HandleFunc("/api/logs/level", h.Logger.HandleSetLogLevel)
	mux.HandleFunc("/api/logs/config", h.Logger.HandleLogConfig)
	mux.HandleFunc("/api/logs/stream", h.LogStream.HandleLogStream)

	// Proxy routes (/api/proxy/*)
	mux.HandleFunc("/api/proxy/current/get", h.Proxy.HandleGetCurrentProxy)
	mux.HandleFunc("/api/proxy/current/set", h.Proxy.HandleSetCurrentProxy)

	// Proxy registry routes (/api/proxy/registry/*)
	mux.HandleFunc("/api/proxy/registry/list", h.ProxyRegistry.HandleProxyRegistryList)
	mux.HandleFunc("/api/proxy/registry/get", h.ProxyRegistry.HandleProxyRegistryGet)

	// Door proxy routes (/api/proxy/door/*)
	mux.HandleFunc("/api/proxy/door/members", h.Door.HandleDoorMemberList)
	mux.HandleFunc("/api/proxy/door/switch", h.Door.HandleDoorMemberSwitch)
	mux.HandleFunc("/api/proxy/door/auto", h.Door.HandleDoorAutoSelect)

	// Token routes (/api/token/*)
	mux.HandleFunc("/api/token/get", h.Token.HandleTokenGet)
	mux.HandleFunc("/api/token/set", h.Token.HandleTokenSet)

	// Run mode routes (/api/run/*)
	mux.HandleFunc("/api/run/start", h.Run.HandleRunStart)
	mux.HandleFunc("/api/run/stop", h.Run.HandleRunStop)
	mux.HandleFunc("/api/run/userInfo", h.Run.HandleRunUserInfo)
	mux.HandleFunc("/api/run/status", h.Run.HandleRunStatus)
	mux.HandleFunc("/api/run/swift", h.Run.HandleRunSwift)

	// Routing Rules API (/api/rules/*)
	mux.HandleFunc("/api/rules/geoip/status", h.Rules.HandleGetGeoIPStatus)
	mux.HandleFunc("/api/rules/geoip/lookup", h.Rules.HandleGeoIPLookup)
	mux.HandleFunc("/api/rules/cache/stats", h.Rules.HandleGetCacheStats)
	mux.HandleFunc("/api/rules/cache/clear", h.Rules.HandleClearCache)
	mux.HandleFunc("/api/rules/engine/status", h.Rules.HandleGetRuleEngineStatus)
	mux.HandleFunc("/api/rules/engine/enable", h.Rules.HandleEnableRuleEngine)
	mux.HandleFunc("/api/rules/engine/disable", h.Rules.HandleDisableRuleEngine)

}
