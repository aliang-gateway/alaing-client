package rules

import (
	"fmt"
	"net"
	"sync"

	"nursor.org/nursorgate/common/logger"
	"nursor.org/nursorgate/common/model"
	M "nursor.org/nursorgate/inbound/tun/metadata"
	"nursor.org/nursorgate/processor/cache"
	"nursor.org/nursorgate/processor/geoip"
)

// RuleEngine evaluates routing rules in priority order
type RuleEngine struct {
	mu            sync.RWMutex
	geoipService  *geoip.Service
	ipDomainCache *cache.IPDomainCache
	nacosRouter   *model.AllowProxyDomain
	chinaDirect   bool // Whether Chinese IPs should route directly
	enabled       bool // Whether rule engine is enabled
}

var (
	defaultEngine *RuleEngine
	engineOnce    sync.Once
)

// GetEngine returns the singleton rule engine instance
func GetEngine() *RuleEngine {
	engineOnce.Do(func() {
		defaultEngine = &RuleEngine{
			enabled: false,
		}
	})
	return defaultEngine
}

// GetCache returns the IP-Domain cache from the singleton rule engine
func GetCache() *cache.IPDomainCache {
	engine := GetEngine()
	if engine != nil {
		engine.mu.RLock()
		defer engine.mu.RUnlock()
		return engine.ipDomainCache
	}
	return nil
}

// Initialize initializes the rule engine with configuration
// TODO(US2): Full implementation will be completed in Phase 4 - User Story 2
func (e *RuleEngine) Initialize(config *model.RoutingRulesConfig) error {
	if config == nil {
		logger.Info("Routing rules config is nil, rule engine disabled")
		return nil
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	// Initialize GeoIP service reference (will be fully configured in US2)
	e.geoipService = geoip.GetService()

	// Initialize Nacos router
	e.nacosRouter = model.NewAllowProxyDomain()

	e.enabled = true
	logger.Info("Rule engine initialized successfully (stub - full implementation in US2)")
	return nil
}

// EvaluateRoute evaluates routing rules for a connection
// Priority order:
// 1. Bypass rules (user-defined direct routes)
// 2. IP-Domain cache (previously evaluated routes)
// 3. Nacos rules (Cursor MITM and Door acceleration domains)
// 4. GeoIP rules (country-based routing)
// 5. Default (requires SNI if port 443)
func (e *RuleEngine) EvaluateRoute(ctx *EvaluationContext) (*RuleResult, error) {
	if !e.enabled {
		return &RuleResult{
			Route:       cache.RouteDirect,
			RequiresSNI: ctx.DstPort == 443, // TLS traffic may need SNI
			Reason:      "rule engine disabled",
		}, nil
	}

	// TODO(US2): Implement routing decision logic with priority: NoneLane > Door > GeoIP > Direct

	// Priority 2: Check cache (avoid repeated SNI extraction)
	if result := e.checkCache(ctx); result != nil {
		return result, nil
	}

	// Priority 3: Check Nacos rules if domain is known
	if ctx.Domain != "" {
		if result := e.checkNacosRules(ctx); result != nil {
			// Cache the result for future lookups
			e.cacheResult(ctx, result)
			return result, nil
		}
	}

	// Priority 4: Check GeoIP (country-based routing)
	if result := e.checkGeoIP(ctx); result != nil {
		e.cacheResult(ctx, result)
		return result, nil
	}

	// Priority 5: Default routing decision
	return e.defaultRoute(ctx), nil
}

// checkBypassRules removed - will be reimplemented as part of routing decision engine in US2
/*
func (e *RuleEngine) checkBypassRules(ctx *EvaluationContext) *RuleResult {
	// TODO(US2): Reimplement this with new model
	return nil
}
*/

// checkCache checks if the routing decision is cached
func (e *RuleEngine) checkCache(ctx *EvaluationContext) *RuleResult {
	if e.ipDomainCache == nil {
		return nil
	}

	e.mu.RLock()
	defer e.mu.RUnlock()

	// Try domain-based lookup first
	if ctx.Domain != "" {
		if entry, ok := e.ipDomainCache.Get(ctx.Domain); ok {
			return &RuleResult{
				Route:       entry.Route,
				MatchedRule: "cache_domain",
				RequiresSNI: false,
				Reason:      fmt.Sprintf("Cache hit for domain %s", ctx.Domain),
			}
		}
	}

	// Try IP-based lookup
	ipKey := ctx.DstIP.String()
	if entry, ok := e.ipDomainCache.Get(ipKey); ok {
		return &RuleResult{
			Route:       entry.Route,
			MatchedRule: "cache_ip",
			RequiresSNI: false,
			Reason:      fmt.Sprintf("Cache hit for IP %s", ctx.DstIP),
		}
	}

	return nil
}

// checkNacosRules checks Nacos domain acceleration rules
func (e *RuleEngine) checkNacosRules(ctx *EvaluationContext) *RuleResult {
	if e.nacosRouter == nil || ctx.Domain == "" {
		return nil
	}

	e.mu.RLock()
	defer e.mu.RUnlock()

	// Check if domain should route to Cursor MITM (highest priority for Cursor AI)
	if e.nacosRouter.IsAllowToCursor(ctx.Domain) {
		return &RuleResult{
			Route:       cache.RouteToCursor,
			MatchedRule: "nacos_cursor",
			RequiresSNI: false,
			Reason:      fmt.Sprintf("Domain %s matched Cursor MITM rules", ctx.Domain),
		}
	}

	// Check if domain should route to Door proxy
	if e.nacosRouter.IsAllowToAnyDoor(ctx.Domain) {
		return &RuleResult{
			Route:       cache.RouteToDoor,
			MatchedRule: "nacos_door",
			RequiresSNI: false,
			Reason:      fmt.Sprintf("Domain %s matched Door acceleration rules", ctx.Domain),
		}
	}

	return nil
}

// checkGeoIP checks country-based routing using GeoIP database
func (e *RuleEngine) checkGeoIP(ctx *EvaluationContext) *RuleResult {
	if e.geoipService == nil || !e.geoipService.IsEnabled() {
		return nil
	}

	e.mu.RLock()
	defer e.mu.RUnlock()

	// Convert netip.Addr to net.IP
	ip := net.IP(ctx.DstIP.AsSlice())

	// Check if IP is in China
	isChina := e.geoipService.IsChina(ip)

	if isChina {
		// Chinese IP handling
		if e.chinaDirect {
			return &RuleResult{
				Route:       cache.RouteDirect,
				MatchedRule: "geoip_china",
				RequiresSNI: false,
				Reason:      fmt.Sprintf("IP %s is in China (direct route)", ctx.DstIP),
			}
		} else {
			return &RuleResult{
				Route:       cache.RouteToDoor,
				MatchedRule: "geoip_china",
				RequiresSNI: false,
				Reason:      fmt.Sprintf("IP %s is in China (accelerated route)", ctx.DstIP),
			}
		}
	}

	// Foreign IP - accelerate via Door proxy
	return &RuleResult{
		Route:       cache.RouteToDoor,
		MatchedRule: "geoip_foreign",
		RequiresSNI: false,
		Reason:      fmt.Sprintf("IP %s is outside China (accelerated)", ctx.DstIP),
	}
}

// defaultRoute provides fallback routing decision
func (e *RuleEngine) defaultRoute(ctx *EvaluationContext) *RuleResult {
	// For TLS traffic (port 443) without domain, we need SNI extraction
	requiresSNI := ctx.DstPort == 443 && ctx.Domain == ""

	return &RuleResult{
		Route:       cache.RouteDirect,
		MatchedRule: "default",
		RequiresSNI: requiresSNI,
		Reason:      "No rules matched, using default direct route",
	}
}

// cacheResult stores a routing decision in the cache
func (e *RuleEngine) cacheResult(ctx *EvaluationContext, result *RuleResult) {
	if e.ipDomainCache == nil {
		return
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	entry := &cache.CacheEntry{
		Domain: ctx.Domain,
		IP:     ctx.DstIP,
		Route:  result.Route,
	}

	// Cache by domain if available
	if ctx.Domain != "" {
		e.ipDomainCache.Set(ctx.Domain, entry)
	}

	// Always cache by IP
	e.ipDomainCache.Set(ctx.DstIP.String(), entry)
}

// IsEnabled returns whether the rule engine is enabled
func (e *RuleEngine) IsEnabled() bool {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.enabled
}

// GetCacheStats returns cache statistics
func (e *RuleEngine) GetCacheStats() map[string]interface{} {
	if e.ipDomainCache == nil {
		return map[string]interface{}{"enabled": false}
	}

	e.mu.RLock()
	defer e.mu.RUnlock()

	return e.ipDomainCache.Stats()
}

// ClearCache clears all cached routing decisions
func (e *RuleEngine) ClearCache() {
	if e.ipDomainCache == nil {
		return
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	e.ipDomainCache.Clear()
	logger.Info("Rule engine cache cleared")
}

// StoreBinding stores DNS binding from connection metadata to cache
// This persists domain-IP relationships observed through SNI, HTTP Host, and CONNECT
func (e *RuleEngine) StoreBinding(metadata *M.Metadata) {
	if e.ipDomainCache == nil || metadata == nil {
		return
	}

	if metadata.DNSInfo == nil || !metadata.DNSInfo.ShouldCache {
		return
	}

	if metadata.HostName == "" || metadata.DstIP.IsUnspecified() {
		return
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	// Convert route string to RouteDecision
	var route cache.RouteDecision
	switch metadata.Route {
	case "RouteToCursor":
		route = cache.RouteToCursor
	case "RouteToDoor":
		route = cache.RouteToDoor
	case "RouteDirect":
		route = cache.RouteDirect
	default:
		route = cache.RouteDirect
	}

	// Create cache entry from DNS binding
	entry := &cache.CacheEntry{
		Domain:         metadata.HostName,
		IP:             metadata.DstIP,
		Route:          route,
		BindingSources: []M.BindingSource{metadata.DNSInfo.BindingSource},
		CreatedAt:      metadata.DNSInfo.BindingTime,
		ExpiresAt:      metadata.DNSInfo.BindingTime.Add(metadata.DNSInfo.CacheTTL),
	}

	// Store by domain
	if metadata.HostName != "" {
		e.ipDomainCache.SetWithTTL(metadata.HostName, entry, metadata.DNSInfo.CacheTTL)
	}

	logger.Debug(fmt.Sprintf("Stored DNS binding: %s (%s) via %s, route: %s",
		metadata.HostName, metadata.DstIP, metadata.DNSInfo.BindingSource, metadata.Route))
}

// Disable disables the rule engine
func (e *RuleEngine) Disable() {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.enabled = false
	logger.Info("Rule engine disabled")
}

// Enable enables the rule engine
func (e *RuleEngine) Enable() {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.enabled = true
	logger.Info("Rule engine enabled")
}
