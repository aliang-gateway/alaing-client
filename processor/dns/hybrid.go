package dns

import (
	"context"
	"fmt"
	"net"
	"strings"
	"time"

	"nursor.org/nursorgate/common/logger"
	"nursor.org/nursorgate/outbound/proxy"
)

// HybridResolver 混合DNS解析器
// 支持多级回退机制：primary → fallback → system
type HybridResolver struct {
	*BaseResolver
	primaryResolver  DNSResolverInterface
	fallbackResolver DNSResolverInterface
	systemResolver   DNSResolverInterface
}

// NewHybridResolver 创建混合DNS解析器
func NewHybridResolver(config *DNSConfig, primaryDialer, fallbackDialer proxy.Dialer) *HybridResolver {
	base := NewBaseResolver(config)

	// 创建主解析器（使用door代理）
	primaryConfig := &DNSConfig{
		Type:         ResolverTypePrimary,
		PrimaryDNS:   config.PrimaryDNS,
		Timeout:      config.Timeout,
		MaxTTL:       config.MaxTTL,
		CacheEnabled: config.CacheEnabled,
	}
	primaryResolver := NewPrimaryResolver(primaryConfig, primaryDialer)

	// 创建回退解析器（使用direct代理）
	var fallbackResolver DNSResolverInterface
	if fallbackDialer != nil {
		fallbackConfig := &DNSConfig{
			Type:         ResolverTypePrimary,
			PrimaryDNS:   config.FallbackDNS,
			Timeout:      config.Timeout,
			MaxTTL:       config.MaxTTL,
			CacheEnabled: config.CacheEnabled,
		}
		fallbackResolver = NewPrimaryResolver(fallbackConfig, fallbackDialer)
	} else {
		fallbackResolver = nil
	}

	// 创建系统解析器
	var systemResolver DNSResolverInterface
	if config.SystemDNSEnabled {
		systemConfig := &DNSConfig{
			Type:         ResolverTypeSystem,
			Timeout:      config.Timeout,
			MaxTTL:       config.MaxTTL,
			CacheEnabled: config.CacheEnabled,
		}
		systemResolver = NewBaseResolver(systemConfig)
	} else {
		systemResolver = nil
	}

	return &HybridResolver{
		BaseResolver:     base,
		primaryResolver:  primaryResolver,
		fallbackResolver: fallbackResolver,
		systemResolver:   systemResolver,
	}
}

// ResolveWithFallback 实现带回退机制的DNS解析
func (h *HybridResolver) ResolveWithFallback(ctx context.Context, domain string) (*ResolveResult, error) {
	domain = normalizeDomainForResolution(domain)
	if domain == "" {
		return &ResolveResult{
			Success: false,
			Error:   fmt.Errorf("empty domain"),
			Source:  "none",
		}, fmt.Errorf("empty domain")
	}

	// 检查是否是IP地址
	if ip := net.ParseIP(domain); ip != nil {
		return &ResolveResult{
			IPs:     []net.IP{ip},
			Success: true,
			Source:  "direct",
			TTL:     h.config.MaxTTL,
		}, nil
	}

	// 1. 尝试主解析器（通常是door代理）
	if h.primaryResolver != nil {
		result, err := h.primaryResolver.ResolveIP(ctx, domain)
		if err == nil && result.Success && len(result.IPs) > 0 {
			result.Source = "primary"
			logger.Debug(fmt.Sprintf("[DNS Hybrid] ✓ %s resolved via primary (%s)", domain, result.IPs[0]))
			return result, nil
		}
		logger.Debug(fmt.Sprintf("[DNS Hybrid] ✗ Primary resolver failed for %s: %v", domain, err))
	}

	// 2. 回退到回退解析器（通常是direct代理）
	if h.fallbackResolver != nil {
		result, err := h.fallbackResolver.ResolveIP(ctx, domain)
		if err == nil && result.Success && len(result.IPs) > 0 {
			result.Source = "fallback"
			logger.Debug(fmt.Sprintf("[DNS Hybrid] ✓ %s resolved via fallback (%s)", domain, result.IPs[0]))
			return result, nil
		}
		logger.Debug(fmt.Sprintf("[DNS Hybrid] ✗ Fallback resolver failed for %s: %v", domain, err))
	}

	// 3. 最后回退到系统DNS解析
	if h.systemResolver != nil {
		result, err := h.systemResolver.ResolveIP(ctx, domain)
		if err == nil && result.Success && len(result.IPs) > 0 {
			result.Source = "system"
			logger.Debug(fmt.Sprintf("[DNS Hybrid] ✓ %s resolved via system (%s)", domain, result.IPs[0]))
			return result, nil
		}
		logger.Debug(fmt.Sprintf("[DNS Hybrid] ✗ System resolver failed for %s: %v", domain, err))
	}

	// 所有解析器都失败了
	return &ResolveResult{
		Success: false,
		Error:   fmt.Errorf("all DNS resolvers failed for %s", domain),
		Source:  "none",
	}, fmt.Errorf("all DNS resolvers failed for %s", domain)
}

// performLookup 重写基础解析器的performLookup方法
func (h *HybridResolver) performLookup(ctx context.Context, domain string, preferIPv4 bool) ([]net.IP, time.Duration, error) {
	result, err := h.ResolveWithFallback(ctx, domain)
	if err != nil {
		return nil, 0, err
	}

	if !result.Success {
		return nil, 0, fmt.Errorf("no IP addresses found for %s", domain)
	}

	// 根据偏好过滤IP地址
	var filteredIPs []net.IP
	if preferIPv4 {
		for _, ip := range result.IPs {
			if ip.To4() != nil {
				filteredIPs = append(filteredIPs, ip)
			}
		}
	} else {
		for _, ip := range result.IPs {
			if ip.To4() == nil {
				filteredIPs = append(filteredIPs, ip)
			}
		}
	}

	// 如果没有找到偏好类型的IP，返回所有IP
	if len(filteredIPs) == 0 {
		filteredIPs = result.IPs
	}

	return filteredIPs, result.TTL, nil
}

// GetType 返回解析器类型
func (h *HybridResolver) GetType() DNSResolverType {
	return ResolverTypeHybrid
}

// GetPrimaryResolver 获取主解析器
func (h *HybridResolver) GetPrimaryResolver() DNSResolverInterface {
	return h.primaryResolver
}

// GetFallbackResolver 获取回退解析器
func (h *HybridResolver) GetFallbackResolver() DNSResolverInterface {
	return h.fallbackResolver
}

// GetSystemResolver 获取系统解析器
func (h *HybridResolver) GetSystemResolver() DNSResolverInterface {
	return h.systemResolver
}

// PrimaryResolver 主解析器（使用单一dialer）
type PrimaryResolver struct {
	*BaseResolver
	dialer proxy.Dialer
}

// NewPrimaryResolver 创建主解析器
func NewPrimaryResolver(config *DNSConfig, dialer proxy.Dialer) *PrimaryResolver {
	return &PrimaryResolver{
		BaseResolver: NewBaseResolver(config),
		dialer:       dialer,
	}
}

// GetType 返回解析器类型
func (p *PrimaryResolver) GetType() DNSResolverType {
	return ResolverTypePrimary
}

// performLookup 使用指定dialer执行DNS解析
func (p *PrimaryResolver) performLookup(ctx context.Context, domain string, preferIPv4 bool) ([]net.IP, time.Duration, error) {
	// 使用TUN模块的DNS解析逻辑，但使用我们的dialer
	if p.dialer == nil {
		// 如果没有dialer，回退到系统解析器
		return p.BaseResolver.performLookup(ctx, domain, preferIPv4)
	}

	// 创建带超时的上下文
	ctx, cancel := context.WithTimeout(ctx, p.config.Timeout)
	defer cancel()

	// 这里应该实现使用指定dialer的DNS解析逻辑
	// 暂时回退到系统解析器，稍后实现完整的dialer集成
	return p.BaseResolver.performLookup(ctx, domain, preferIPv4)
}

// SystemResolver 系统DNS解析器
type SystemResolver struct {
	*BaseResolver
}

// NewSystemResolver 创建系统解析器
func NewSystemResolver(config *DNSConfig) *SystemResolver {
	return &SystemResolver{
		BaseResolver: NewBaseResolver(config),
	}
}

// GetType 返回解析器类型
func (s *SystemResolver) GetType() DNSResolverType {
	return ResolverTypeSystem
}

// 工具函数

// normalizeDomainForResolution 规范化域名用于解析
func normalizeDomainForResolution(domain string) string {
	if domain == "" {
		return ""
	}
	domain = strings.ToLower(strings.TrimSpace(domain))
	// 确保域名以点结尾（DNS标准格式）
	if !strings.HasSuffix(domain, ".") {
		domain += "."
	}
	return domain
}

// CreateDefaultHybridResolver 创建默认配置的混合解析器
func CreateDefaultHybridResolver(primaryDialer, fallbackDialer proxy.Dialer) DNSResolverInterface {
	config := DefaultDNSConfig()
	return NewHybridResolver(config, primaryDialer, fallbackDialer)
}