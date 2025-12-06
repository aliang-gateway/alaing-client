package routes

import (
	"net/http"

	"nursor.org/nursorgate/app/http/handlers"
	"nursor.org/nursorgate/app/http/repositories"
	"nursor.org/nursorgate/app/http/services"
)

// Handlers holds all HTTP handler instances
type Handlers struct {
	Logger         *handlers.LogHandler
	Proxy          *handlers.ProxyHandler
	ProxyRegistry  *handlers.ProxyRegistryHandler
	Config         *handlers.ConfigHandler
	Token          *handlers.TokenHandler
	Run            *handlers.RunHandler
	LogStream      *handlers.LogStreamHandler
}

// NewHandlers creates and initializes all handlers with their dependencies
func NewHandlers() *Handlers {
	// Initialize services
	logService := services.NewLogService()
	logConfigService := services.NewLogConfigService()
	proxyService := services.NewProxyService()
	tokenService := services.NewTokenService()
	runService := services.NewRunService()

	// Initialize repositories
	proxyRepository := repositories.NewProxyRepository()
	configRepository := repositories.NewConfigRepository()

	// Create handlers with dependency injection
	return &Handlers{
		Logger:        handlers.NewLogHandler(logService, logConfigService),
		Proxy:         handlers.NewProxyHandler(proxyService),
		ProxyRegistry: handlers.NewProxyRegistryHandler(proxyRepository),
		Config:        handlers.NewConfigHandler(configRepository),
		Token:         handlers.NewTokenHandler(tokenService),
		Run:           handlers.NewRunHandler(runService),
		LogStream:     handlers.NewLogStreamHandler(),
	}
}

// RegisterRoutes registers all feature-grouped HTTP routes
func RegisterRoutes(h *Handlers) {
	// Logger routes (/api/logs/*)
	http.HandleFunc("/api/logs", h.Logger.HandleGetLogs)
	http.HandleFunc("/api/logs/clear", h.Logger.HandleClearLogs)
	http.HandleFunc("/api/logs/level", h.Logger.HandleSetLogLevel)
	http.HandleFunc("/api/logs/config", h.Logger.HandleLogConfig)
	http.HandleFunc("/api/logs/stream", h.LogStream.HandleLogStream)

	// Proxy routes (/api/proxy/*)
	http.HandleFunc("/api/proxy/current/get", h.Proxy.HandleGetCurrentProxy)
	http.HandleFunc("/api/proxy/current/set", h.Proxy.HandleSetCurrentProxy)

	// Proxy registry routes (/api/proxy/registry/*)
	http.HandleFunc("/api/proxy/registry/list", h.ProxyRegistry.HandleProxyRegistryList)
	http.HandleFunc("/api/proxy/registry/get", h.ProxyRegistry.HandleProxyRegistryGet)
	http.HandleFunc("/api/proxy/registry/register", h.ProxyRegistry.HandleProxyRegistryRegister)
	http.HandleFunc("/api/proxy/registry/unregister", h.ProxyRegistry.HandleProxyRegistryUnregister)
	http.HandleFunc("/api/proxy/registry/set-default", h.ProxyRegistry.HandleProxyRegistrySetDefault)
	http.HandleFunc("/api/proxy/registry/set-door", h.ProxyRegistry.HandleProxyRegistrySetDoor)
	http.HandleFunc("/api/proxy/registry/switch", h.ProxyRegistry.HandleProxyRegistrySwitch)

	// Config routes (/api/config/*)
	http.HandleFunc("/api/config/get", h.Config.HandleConfigGet)
	http.HandleFunc("/api/config/list", h.Config.HandleConfigList)

	// Token routes (/api/token/*)
	http.HandleFunc("/api/token/get", h.Token.HandleTokenGet)
	http.HandleFunc("/api/token/set", h.Token.HandleTokenSet)

	// Run mode routes (/api/run/*)
	http.HandleFunc("/api/run/start", h.Run.HandleRunStart)
	http.HandleFunc("/api/run/stop", h.Run.HandleRunStop)
	http.HandleFunc("/api/run/userInfo", h.Run.HandleRunUserInfo)
	http.HandleFunc("/api/run/status", h.Run.HandleRunStatus)
	http.HandleFunc("/api/run/swift", h.Run.HandleRunSwift)

	// Legacy routes (for backward compatibility, before refactoring)
	// These will be removed after full migration
	http.HandleFunc("/proxy/current/get", h.Proxy.HandleGetCurrentProxy)
	http.HandleFunc("/proxy/current/set", h.Proxy.HandleSetCurrentProxy)
	http.HandleFunc("/proxy/registry/list", h.ProxyRegistry.HandleProxyRegistryList)
	http.HandleFunc("/proxy/registry/get", h.ProxyRegistry.HandleProxyRegistryGet)
	http.HandleFunc("/proxy/registry/register", h.ProxyRegistry.HandleProxyRegistryRegister)
	http.HandleFunc("/proxy/registry/unregister", h.ProxyRegistry.HandleProxyRegistryUnregister)
	http.HandleFunc("/proxy/registry/set-default", h.ProxyRegistry.HandleProxyRegistrySetDefault)
	http.HandleFunc("/proxy/registry/set-door", h.ProxyRegistry.HandleProxyRegistrySetDoor)
	http.HandleFunc("/proxy/registry/switch", h.ProxyRegistry.HandleProxyRegistrySwitch)
	http.HandleFunc("/config/get", h.Config.HandleConfigGet)
	http.HandleFunc("/config/list", h.Config.HandleConfigList)
	http.HandleFunc("/token/get", h.Token.HandleTokenGet)
	http.HandleFunc("/token/set", h.Token.HandleTokenSet)
	http.HandleFunc("/run/start", h.Run.HandleRunStart)
	http.HandleFunc("/run/stop", h.Run.HandleRunStop)
	http.HandleFunc("/run/userInfo", h.Run.HandleRunUserInfo)
	http.HandleFunc("/run/status", h.Run.HandleRunStatus)
	http.HandleFunc("/run/swift", h.Run.HandleRunSwift)
}
