package routes

import (
	"net/http"

	"nursor.org/nursorgate/app/http/handlers"
	"nursor.org/nursorgate/app/http/repositories"
	"nursor.org/nursorgate/app/http/services"
	"nursor.org/nursorgate/processor/statistic"
)

// Handlers holds all HTTP handler instances
type Handlers struct {
	Logger        *handlers.LogHandler
	Proxy         *handlers.ProxyHandler
	ProxyRegistry *handlers.ProxyRegistryHandler
	Token         *handlers.TokenHandler
	Run           *handlers.RunHandler
	LogStream     *handlers.LogStreamHandler
	Door          *handlers.DoorHandler
	Rules         *handlers.RulesHandler
	DNSCache      *handlers.DNSCacheHandler
	Stats         *handlers.StatsHandler
	Cert          *handlers.CertHandler
	Auth          *handlers.AuthHandler
}

// NewHandlers creates and initializes all handlers with their dependencies
func NewHandlers() *Handlers {
	// Initialize services
	logService := services.NewLogService()
	logConfigService := services.NewLogConfigService()
	proxyService := services.NewProxyService()
	tokenService := services.NewTokenService()
	runService := services.NewRunService()
	certService := services.NewCertService()

	// Initialize repositories
	proxyRepository := repositories.NewProxyRepository()

	// Create handlers with dependency injection
	return &Handlers{
		Logger:        handlers.NewLogHandler(logService, logConfigService),
		Proxy:         handlers.NewProxyHandler(proxyService),
		ProxyRegistry: handlers.NewProxyRegistryHandler(proxyRepository),
		Token:         handlers.NewTokenHandler(tokenService),
		Run:           handlers.NewRunHandler(runService),
		LogStream:     handlers.NewLogStreamHandler(),
		Door:          handlers.NewDoorHandler(),
		Rules:         handlers.NewRulesHandler(),
		DNSCache:      handlers.NewDNSCacheHandler(),
		Stats:         handlers.NewStatsHandler(statistic.DefaultManager),
		Cert:          handlers.NewCertHandler(certService),
		Auth:          handlers.NewAuthHandler(),
	}
}

// RegisterRoutes registers all feature-grouped HTTP routes
func RegisterRoutes(h *Handlers, mux *http.ServeMux) {
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
	mux.HandleFunc("/api/proxy/list", h.ProxyRegistry.HandleProxyRegistryList)
	mux.HandleFunc("/api/proxy/get", h.ProxyRegistry.HandleProxyRegistryGet)

	// Door proxy routes (/api/proxy/door/*)
	mux.HandleFunc("/api/proxy/door/members", h.Door.HandleDoorMemberList)
	mux.HandleFunc("/api/proxy/door/auto", h.Door.HandleDoorAutoSelect)

	// Token routes (/api/token/*)
	mux.HandleFunc("/api/token/get", h.Token.HandleTokenGet)
	mux.HandleFunc("/api/token/set", h.Token.HandleTokenSet)

	// Authentication routes (/api/auth/*)
	mux.HandleFunc("/api/auth/activate", h.Auth.HandleActivateToken)
	mux.HandleFunc("/api/auth/userinfo", h.Auth.HandleGetUserInfo)
	mux.HandleFunc("/api/auth/refresh-status", h.Auth.HandleGetRefreshStatus)
	mux.HandleFunc("/api/auth/logout", h.Auth.HandleLogout)

	// Run mode routes (/api/run/*)
	mux.HandleFunc("/api/run/start", h.Run.HandleRunStart)
	mux.HandleFunc("/api/run/stop", h.Run.HandleRunStop)
	mux.HandleFunc("/api/run/userInfo", h.Run.HandleRunUserInfo)
	mux.HandleFunc("/api/run/status", h.Run.HandleRunStatus)
	mux.HandleFunc("/api/run/swift", h.Run.HandleRunSwift)
	mux.HandleFunc("/api/run/mode", h.Run.HandleRunSwift) // Alias for /api/run/swift

	// Routing Rules API (/api/rules/*)
	mux.HandleFunc("/api/rules/geoip/status", h.Rules.HandleGetGeoIPStatus)
	mux.HandleFunc("/api/rules/geoip/lookup", h.Rules.HandleGeoIPLookup)
	mux.HandleFunc("/api/rules/cache/stats", h.Rules.HandleGetCacheStats)
	mux.HandleFunc("/api/rules/cache/clear", h.Rules.HandleClearCache)
	mux.HandleFunc("/api/rules/engine/status", h.Rules.HandleGetRuleEngineStatus)
	mux.HandleFunc("/api/rules/engine/enable", h.Rules.HandleEnableRuleEngine)
	mux.HandleFunc("/api/rules/engine/disable", h.Rules.HandleDisableRuleEngine)

	// Statistics API (/api/stats/*)
	mux.HandleFunc("/api/stats", h.Stats.HandleGetStats)

	// Certificate Management API (/api/cert/*)
	mux.HandleFunc("/api/cert/status", h.Cert.HandleGetStatus)
	mux.HandleFunc("/api/cert/export", h.Cert.HandleExport)
	mux.HandleFunc("/api/cert/download", h.Cert.HandleDownload)
	mux.HandleFunc("/api/cert/install", h.Cert.HandleInstall)
	mux.HandleFunc("/api/cert/remove", h.Cert.HandleRemove)
	mux.HandleFunc("/api/cert/generate", h.Cert.HandleGenerateCert)
	mux.HandleFunc("/api/cert/info", h.Cert.HandleGetInfo)

	// DNS Cache API (/api/dns/*)
	// 注意：更具体的路由必须放在更通用的路由之前，避免路径冲突
	mux.HandleFunc("/api/dns/cache/query", h.DNSCache.QueryDomain)
	mux.HandleFunc("/api/dns/cache/reverse", h.DNSCache.ReverseQuery)
	mux.HandleFunc("/api/dns/cache/clear", h.DNSCache.ClearAll)
	mux.HandleFunc("/api/dns/cache/delete/{domain}", h.DNSCache.DeleteEntry)
	mux.HandleFunc("/api/dns/cache", h.DNSCache.GetCacheEntries)
	mux.HandleFunc("/api/dns/stats", h.DNSCache.GetStatistics)
	mux.HandleFunc("/api/dns/hotspots", h.DNSCache.GetHotspots)
}
