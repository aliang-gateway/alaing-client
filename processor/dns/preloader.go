package dns

import (
	"context"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"nursor.org/nursorgate/common/logger"
	"nursor.org/nursorgate/processor/config"
)

// Preloader DNS预加载器，负责在代理注册前解析域名
type Preloader struct {
	resolver DNSResolverInterface
	config   *config.DNSPreResolutionConfig
	mu       sync.RWMutex

	// 预解析结果缓存
	preloadCache map[string]*PreloadResult
	cacheMu      sync.RWMutex

	// 统计信息
	stats struct {
		TotalAttempts int64
		Successful    int64
		Failed        int64
		CacheHits     int64
		TotalDuration time.Duration
		lastUpdate    time.Time
		mu            sync.RWMutex
	}
}

// NewPreloader 创建DNS预加载器
func NewPreloader(resolver DNSResolverInterface, cfg *config.DNSPreResolutionConfig) *Preloader {
	if cfg == nil {
		cfg = &config.DNSPreResolutionConfig{
			Enabled:         true,
			Timeout:         "10s",
			ConcurrentLimit: 10,
			RetryOnFailure:  true,
			CacheResults:    true,
			PreferIPv4:      true,
			ForceResolve:    false,
		}
	}

	p := &Preloader{
		resolver:     resolver,
		config:       cfg,
		preloadCache: make(map[string]*PreloadResult),
	}

	// 启动缓存清理协程
	go p.startCacheCleanup()

	return p
}

// PreloadDomain 预解析单个域名
func (p *Preloader) PreloadDomain(ctx context.Context, domain string) *PreloadResult {
	domain = normalizeDomain(domain)
	if domain == "" {
		return &PreloadResult{
			Domain:    domain,
			Success:   false,
			Error:     fmt.Errorf("empty domain"),
			Timestamp: time.Now(),
		}
	}

	// 检查是否是IP地址
	if ip := net.ParseIP(domain); ip != nil {
		return &PreloadResult{
			Domain:    domain,
			IP:        ip,
			Success:   true,
			Timestamp: time.Now(),
		}
	}

	// 检查预解析缓存
	if p.config.CacheResults {
		if cached := p.getPreloadCached(domain); cached != nil {
			p.updateStats(true, time.Duration(0), true) // 缓存命中
			return cached
		}
	}

	start := time.Now()
	result := &PreloadResult{
		Domain:    domain,
		Timestamp: start,
	}

	// 执行DNS解析
	resolveResult, err := p.resolver.ResolveIP(ctx, domain)
	result.Duration = time.Since(start)

	if err == nil && resolveResult.Success && len(resolveResult.IPs) > 0 {
		result.Success = true
		// 根据配置选择IP
		result.IP = p.selectIP(resolveResult.IPs)

		logger.Debug(fmt.Sprintf("[DNS Preloader] ✓ %s -> %s (%v)", domain, result.IP, result.Duration))

		// 缓存结果
		if p.config.CacheResults {
			p.setPreloadCached(domain, result)
		}
	} else {
		result.Error = fmt.Errorf("preload failed: %v", err)
		logger.Warn(fmt.Sprintf("[DNS Preloader] ✗ %s: %v", domain, err))

		// 如果配置了重试，进行重试
		if p.config.RetryOnFailure {
			result = p.retryPreload(ctx, domain, result)
		}
	}

	p.updateStats(result.Success, result.Duration, false)
	return result
}

// PreloadDomains 批量预解析域名
func (p *Preloader) PreloadDomains(ctx context.Context, domains []string) map[string]*PreloadResult {
	logger.Info(fmt.Sprintf("[DNS Preloader] Starting batch preload of %d domains", len(domains)))

	results := make(map[string]*PreloadResult)

	// 去重和预处理
	uniqueDomains := make(map[string]bool)
	for _, domain := range domains {
		domain = normalizeDomain(domain)
		if domain != "" && !p.isIPAddress(domain) {
			uniqueDomains[domain] = true
		}
	}

	// 并发解析，限制并发数
	sem := make(chan struct{}, p.config.ConcurrentLimit)
	var wg sync.WaitGroup
	var mu sync.Mutex

	for domain := range uniqueDomains {
		wg.Add(1)
		go func(d string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			result := p.PreloadDomain(ctx, d)
			mu.Lock()
			results[d] = result
			mu.Unlock()
		}(domain)
	}

	wg.Wait()

	// 统计结果
	successCount := 0
	failedCount := 0
	for _, result := range results {
		if result.Success {
			successCount++
		} else {
			failedCount++
		}
	}

	logger.Info(fmt.Sprintf("[DNS Preloader] Batch preload completed: %d successful, %d failed", successCount, failedCount))
	return results
}

// PreloadServerHost 预解析服务器主机地址（主要用于代理配置）
func (p *Preloader) PreloadServerHost(ctx context.Context, serverHost string) string {
	if serverHost == "" {
		return serverHost
	}

	// 如果已经是IP地址，直接返回
	if ip := net.ParseIP(serverHost); ip != nil {
		return serverHost
	}

	result := p.PreloadDomain(ctx, serverHost)
	if result.Success && result.IP != nil {
		return result.IP.String()
	}

	// 预解析失败，返回原域名
	logger.Warn(fmt.Sprintf("[DNS Preloader] Failed to resolve %s, keeping original domain", serverHost))
	return serverHost
}

// GetCachedResult 获取缓存的预解析结果
func (p *Preloader) GetCachedResult(domain string) *PreloadResult {
	domain = normalizeDomain(domain)
	if domain == "" {
		return nil
	}
	return p.getPreloadCached(domain)
}

// ClearCache 清理预解析缓存
func (p *Preloader) ClearCache() {
	p.cacheMu.Lock()
	defer p.cacheMu.Unlock()
	p.preloadCache = make(map[string]*PreloadResult)
	logger.Info("[DNS Preloader] Cache cleared")
}

// GetStats 获取预加载统计信息
func (p *Preloader) GetStats() *PreloaderStats {
	p.stats.mu.RLock()
	defer p.stats.mu.RUnlock()

	p.cacheMu.RLock()
	cacheSize := len(p.preloadCache)
	p.cacheMu.RUnlock()

	successRate := float64(0)
	if p.stats.TotalAttempts > 0 {
		successRate = float64(p.stats.Successful) / float64(p.stats.TotalAttempts)
	}

	cacheHitRate := float64(0)
	if p.stats.TotalAttempts > 0 {
		cacheHitRate = float64(p.stats.CacheHits) / float64(p.stats.TotalAttempts)
	}

	avgDuration := time.Duration(0)
	if p.stats.Successful > 0 {
		avgDuration = p.stats.TotalDuration / time.Duration(p.stats.Successful)
	}

	return &PreloaderStats{
		TotalAttempts: p.stats.TotalAttempts,
		Successful:    p.stats.Successful,
		Failed:        p.stats.Failed,
		CacheHits:     p.stats.CacheHits,
		SuccessRate:   successRate,
		CacheHitRate:  cacheHitRate,
		AvgDuration:   avgDuration,
		CacheSize:     cacheSize,
		LastUpdate:    p.stats.lastUpdate,
	}
}

// PreloaderStats 预加载器统计信息
type PreloaderStats struct {
	TotalAttempts int64         `json:"total_attempts"`
	Successful    int64         `json:"successful"`
	Failed        int64         `json:"failed"`
	CacheHits     int64         `json:"cache_hits"`
	SuccessRate   float64       `json:"success_rate"`
	CacheHitRate  float64       `json:"cache_hit_rate"`
	AvgDuration   time.Duration `json:"avg_duration"`
	CacheSize     int           `json:"cache_size"`
	LastUpdate    time.Time     `json:"last_update"`
}

// SetConfig 更新配置
func (p *Preloader) SetConfig(cfg *config.DNSPreResolutionConfig) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.config = cfg
}

// GetConfig 获取当前配置
func (p *Preloader) GetConfig() *config.DNSPreResolutionConfig {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.config
}

// 内部方法

// retryPreload 重试预解析
func (p *Preloader) retryPreload(ctx context.Context, domain string, originalResult *PreloadResult) *PreloadResult {
	if !p.config.RetryOnFailure {
		return originalResult
	}

	maxRetries := 2 // 默认重试2次
	for i := 0; i < maxRetries; i++ {
		time.Sleep(100 * time.Millisecond) // 短暂延迟后重试

		resolveResult, err := p.resolver.ResolveIP(ctx, domain)
		if err == nil && resolveResult.Success && len(resolveResult.IPs) > 0 {
			result := &PreloadResult{
				Domain:    domain,
				IP:        p.selectIP(resolveResult.IPs),
				Success:   true,
				Duration:  time.Since(originalResult.Timestamp),
				Timestamp: originalResult.Timestamp,
			}

			logger.Debug(fmt.Sprintf("[DNS Preloader] ✓ Retry success for %s -> %s", domain, result.IP))
			return result
		}
	}

	// 重试都失败，返回原始结果
	return originalResult
}

// selectIP 根据配置选择最合适的IP地址
func (p *Preloader) selectIP(ips []net.IP) net.IP {
	if len(ips) == 0 {
		return nil
	}

	if p.config.PreferIPv4 {
		// 优先选择IPv4
		for _, ip := range ips {
			if ip.To4() != nil {
				return ip
			}
		}
		// 没有IPv4，返回第一个IPv6
		return ips[0]
	} else {
		// 优先选择IPv6
		for _, ip := range ips {
			if ip.To4() == nil {
				return ip
			}
		}
		// 没有IPv6，返回第一个IPv4
		return ips[0]
	}
}

// getPreloadCached 获取预解析缓存
func (p *Preloader) getPreloadCached(domain string) *PreloadResult {
	p.cacheMu.RLock()
	defer p.cacheMu.RUnlock()

	result, exists := p.preloadCache[domain]
	if !exists {
		return nil
	}

	// 检查是否过期（TTL为1小时）
	if time.Since(result.Timestamp) > time.Hour {
		return nil
	}

	return result
}

// setPreloadCached 设置预解析缓存
func (p *Preloader) setPreloadCached(domain string, result *PreloadResult) {
	p.cacheMu.Lock()
	defer p.cacheMu.Unlock()
	p.preloadCache[domain] = result
}

// updateStats 更新统计信息
func (p *Preloader) updateStats(success bool, duration time.Duration, cacheHit bool) {
	p.stats.mu.Lock()
	defer p.stats.mu.Unlock()

	p.stats.TotalAttempts++
	if success {
		p.stats.Successful++
		p.stats.TotalDuration += duration
	} else {
		p.stats.Failed++
	}

	if cacheHit {
		p.stats.CacheHits++
	}

	p.stats.lastUpdate = time.Now()
}

// startCacheCleanup 启动缓存清理协程
func (p *Preloader) startCacheCleanup() {
	ticker := time.NewTicker(30 * time.Minute) // 每30分钟清理一次
	defer ticker.Stop()

	for range ticker.C {
		p.cleanExpiredCache()
	}
}

// cleanExpiredCache 清理过期的预解析缓存
func (p *Preloader) cleanExpiredCache() {
	p.cacheMu.Lock()
	defer p.cacheMu.Unlock()

	now := time.Now()
	expiredCount := 0
	for domain, result := range p.preloadCache {
		if now.Sub(result.Timestamp) > time.Hour { // 1小时过期
			delete(p.preloadCache, domain)
			expiredCount++
		}
	}

	if expiredCount > 0 {
		logger.Debug(fmt.Sprintf("[DNS Preloader] Cleaned %d expired cache entries", expiredCount))
	}
}

// 工具函数

// normalizeDomain 规范化域名
func normalizeDomain(domain string) string {
	if domain == "" {
		return ""
	}
	domain = strings.ToLower(strings.TrimSpace(domain))
	// 移除端口号
	if idx := strings.Index(domain, ":"); idx != -1 {
		domain = domain[:idx]
	}
	return domain
}

// isIPAddress 检查字符串是否是IP地址
func (p *Preloader) isIPAddress(s string) bool {
	return net.ParseIP(s) != nil
}
