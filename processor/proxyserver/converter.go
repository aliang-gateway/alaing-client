package proxyserver

import (
	"nursor.org/nursorgate/common/logger"
	"nursor.org/nursorgate/outbound/proxy"
	"nursor.org/nursorgate/processor/config"
	"nursor.org/nursorgate/processor/dns"
)

// Global DNS resolver instance
var globalResolver dns.DNSResolverInterface

// InitGlobalResolver initializes the global DNS resolver
// Should be called in ApplyConfig after proxies are registered
func InitGlobalResolver(primaryProxy, fallbackProxy proxy.Proxy, cfg *config.Config) error {
	if cfg == nil || cfg.DNSPreResolution == nil || !cfg.DNSPreResolution.Enabled {
		logger.Info("[DNS] DNS resolution disabled in config")
		return nil
	}

	// Create hybrid DNS resolver using primary and fallback proxies
	hybridResolver := dns.NewHybridResolver(
		&dns.DNSConfig{
			Type:             dns.ResolverTypeHybrid,
			PrimaryDNS:       cfg.DNSPreResolution.GetPrimaryDNS(),
			FallbackDNS:      cfg.DNSPreResolution.GetFallbackDNS(),
			SystemDNSEnabled: cfg.DNSPreResolution.SystemDNSFallback,
			Timeout:          cfg.DNSPreResolution.GetTimeout(),
			MaxTTL:           cfg.DNSPreResolution.GetMaxCacheTTL(),
			CacheEnabled:     cfg.DNSPreResolution.CacheResults,
		},
		primaryProxy,  // primary dialer (implements proxy.Dialer)
		fallbackProxy, // fallback dialer (implements proxy.Dialer)
	)

	globalResolver = hybridResolver
	logger.Info("[DNS] Global DNS resolver initialized successfully")
	return nil
}

// GetGlobalResolver returns the global DNS resolver instance
func GetGlobalResolver() dns.DNSResolverInterface {
	return globalResolver
}
