package proxyserver

import (
	"fmt"

	"nursor.org/nursorgate/common/logger"
	"nursor.org/nursorgate/outbound/proxy"
	"nursor.org/nursorgate/processor/config"
	"nursor.org/nursorgate/processor/dns"
)

// Global DNS resolver instance
var globalResolver dns.DNSResolverInterface

// InitGlobalResolver initializes the global DNS resolver
// Should be called in ApplyConfig after door and direct proxies are registered
func InitGlobalResolver(doorProxy, directProxy proxy.Proxy, cfg *config.Config) error {
	if cfg == nil || cfg.DNSPreResolution == nil || !cfg.DNSPreResolution.Enabled {
		logger.Info("[DNS] DNS resolution disabled in config")
		return nil
	}

	// Create hybrid DNS resolver using door and direct proxies
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
		doorProxy,   // primary dialer (implements proxy.Dialer)
		directProxy, // fallback dialer (implements proxy.Dialer)
	)

	globalResolver = hybridResolver
	logger.Info("[DNS] Global DNS resolver initialized successfully")
	return nil
}

// GetGlobalResolver returns the global DNS resolver instance
func GetGlobalResolver() dns.DNSResolverInterface {
	return globalResolver
}

// UpdateGlobalResolverWithDoorConfig updates the global resolver after door config changes
func UpdateGlobalResolverWithDoorConfig(doorConfig *config.DoorProxyConfig) {
	if globalResolver == nil {
		return
	}

	if doorConfig == nil || len(doorConfig.Members) == 0 {
		logger.Debug("[DNS] No door members, skipping resolver sync")
		return
	}

	logger.Info(fmt.Sprintf("[DNS] Door config updated with %d members", len(doorConfig.Members)))
}
