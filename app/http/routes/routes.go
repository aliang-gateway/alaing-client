package routes

import (
	"fmt"
	"net/http"

	"nursor.org/nursorgate/app/http/handlers"
	"nursor.org/nursorgate/app/http/repositories"
	"nursor.org/nursorgate/app/http/services"
	"nursor.org/nursorgate/common/logger"
	"nursor.org/nursorgate/processor/statistic"
)

// Handlers holds all HTTP handler instances
type Handlers struct {
	Logger        *handlers.LogHandler
	Proxy         *handlers.ProxyHandler
	ProxyRegistry *handlers.ProxyRegistryHandler
	SoftwareCfg   *handlers.SoftwareConfigHandler
	Token         *handlers.TokenHandler
	Run           *handlers.RunHandler
	LogStream     *handlers.LogStreamHandler
	Rules         *handlers.RulesHandler
	DNSCache      *handlers.DNSCacheHandler
	Cert          *handlers.CertHandler
	Auth          *handlers.AuthHandler
	Startup       *handlers.StartupHandler
	Config        *handlers.ConfigHandler
	TrafficStats  *handlers.TrafficStatsHandler
	HTTPStats     *handlers.HTTPStatsHandler
	Chat          *handlers.ChatHandler

	statsCollector     *statistic.StatsCollector
	httpStatsCollector *statistic.HTTPStatsCollector
}

// NewHandlers creates and initializes all handlers with their dependencies
func NewHandlers() *Handlers {
	logService := services.NewLogService()
	logConfigService := services.NewLogConfigService()
	tokenService := services.NewTokenService()
	runService := services.NewRunService()
	softwareCfgService := services.NewSoftwareConfigService()
	certService := services.NewCertService()
	proxyRepository := repositories.NewProxyRepository()
	statsCollector := statistic.NewStatsCollector()
	httpStatsCollector := statistic.GetDefaultHTTPStatsCollector()

	return &Handlers{
		Logger:             handlers.NewLogHandler(logService, logConfigService),
		Proxy:              handlers.NewProxyHandler(),
		ProxyRegistry:      handlers.NewProxyRegistryHandler(proxyRepository),
		SoftwareCfg:        handlers.NewSoftwareConfigHandler(softwareCfgService),
		Token:              handlers.NewTokenHandler(tokenService),
		Run:                handlers.NewRunHandler(runService),
		LogStream:          handlers.NewLogStreamHandler(),
		Rules:              handlers.NewRulesHandler(),
		DNSCache:           handlers.NewDNSCacheHandler(),
		Cert:               handlers.NewCertHandler(certService),
		Auth:               handlers.NewAuthHandler(),
		Startup:            handlers.NewStartupHandler(),
		Config:             handlers.NewConfigHandler(),
		TrafficStats:       handlers.NewTrafficStatsHandler(statsCollector),
		HTTPStats:          handlers.NewHTTPStatsHandler(httpStatsCollector),
		Chat:               handlers.NewChatHandler(),
		statsCollector:     statsCollector,
		httpStatsCollector: httpStatsCollector,
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

	mux.HandleFunc("/api/software-config/save", h.SoftwareCfg.HandleSave)
	mux.HandleFunc("/api/software-config/activate", h.SoftwareCfg.HandleActivate)
	mux.HandleFunc("/api/software-config/list", h.SoftwareCfg.HandleList)
	mux.HandleFunc("/api/software-config/cloud/push", h.SoftwareCfg.HandlePushToCloud)
	mux.HandleFunc("/api/software-config/cloud/pull", h.SoftwareCfg.HandlePullFromCloud)

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

	// Startup Status API (/api/startup/*)
	mux.HandleFunc("/api/startup/status", h.Startup.HandleStartupStatus)
	mux.HandleFunc("/api/startup/detail", h.Startup.HandleStartupDetail)

	// Routing Config compatibility API (/api/config/routing/*)
	mux.HandleFunc("/api/config/routing", h.Config.HandleRoutingConfig)
	mux.HandleFunc("/api/config/routing/rules/", h.Config.HandleToggleRuleStatus)
	mux.HandleFunc("/api/config/routing/auto-update", h.Config.HandleAutoUpdateStatus)

	// Traffic Statistics API (/api/stats/traffic/*)
	mux.HandleFunc("/api/stats/traffic/", h.TrafficStats.HandleGetStats)
	mux.HandleFunc("/api/stats/traffic/current", h.TrafficStats.HandleGetCurrentStats)
	mux.HandleFunc("/api/stats/traffic/cache/info", h.TrafficStats.HandleGetCacheInfo)
	mux.HandleFunc("/api/stats/traffic/cache/clear", h.TrafficStats.HandleClearCache)

	// HTTP Statistics API (/api/stats/http/*)
	mux.HandleFunc("/api/stats/http/requests", h.HTTPStats.HandleGetRequests)
	mux.HandleFunc("/api/stats/http/domains", h.HTTPStats.HandleGetDomainStats)
	mux.HandleFunc("/api/stats/http/chart", h.HTTPStats.HandleGetChartData)
	mux.HandleFunc("/api/stats/http/info", h.HTTPStats.HandleGetStats)
	mux.HandleFunc("/api/stats/http/clear", h.HTTPStats.HandleClear)
	mux.HandleFunc("/api/stats/http/preset-domains", h.HTTPStats.HandleGetPresetDomains)

	mux.HandleFunc("/api/chat/completions", h.Chat.HandleCompletions)
}

// StartStatsCollector starts the traffic statistics collector background task
func StartStatsCollector(h *Handlers) error {
	if h.statsCollector == nil {
		return fmt.Errorf("stats collector not initialized")
	}

	if err := h.statsCollector.Start(); err != nil {
		logger.Error(fmt.Sprintf("Failed to start stats collector: %v", err))
		return err
	}

	logger.Info("Traffic stats collector started successfully")
	return nil
}

// StopStatsCollector stops the traffic statistics collector
func StopStatsCollector(h *Handlers) {
	if h.statsCollector != nil {
		h.statsCollector.Stop()
		logger.Info("Traffic stats collector stopped")
	}
}

// StartHTTPStatsCollector starts the HTTP statistics collector background task
func StartHTTPStatsCollector(h *Handlers) error {
	if h.httpStatsCollector == nil {
		return fmt.Errorf("http stats collector not initialized")
	}

	if err := h.httpStatsCollector.Start(); err != nil {
		logger.Error(fmt.Sprintf("Failed to start http stats collector: %v", err))
		return err
	}

	logger.Info("HTTP stats collector started successfully")
	return nil
}

// StopHTTPStatsCollector stops the HTTP statistics collector
func StopHTTPStatsCollector(h *Handlers) {
	if h.httpStatsCollector != nil {
		h.httpStatsCollector.Stop()
		logger.Info("HTTP stats collector stopped")
	}
}
