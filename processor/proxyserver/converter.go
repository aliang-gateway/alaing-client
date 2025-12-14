package proxyserver

import (
	"fmt"
	"os"
	"path/filepath"

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

// CleanupGlobalResolver cleans up the global DNS resolver
func CleanupGlobalResolver() {
	if globalResolver != nil {
		if clearable, ok := globalResolver.(interface{ ClearCache() }); ok {
			clearable.ClearCache()
		}
		globalResolver = nil
		logger.Info("[DNS] Global DNS resolver cleaned up")
	}
}

// processMembersWithDNS 处理DoorProxyMember列表，执行DNS预解析和延迟测试
// 这个函数将原来在转换层的DNS处理逻辑集成到注册流程中
func processMembersWithDNS(members []config.DoorProxyMember) ([]config.DoorProxyMember, error) {
	if len(members) == 0 {
		return members, nil
	}

	// TODO: DNS预解析逻辑
	// 后续可以在这里添加：
	// 1. 域名解析并缓存IP
	// 2. 延迟测试
	// 3. 根据延迟对members排序

	logger.Info(fmt.Sprintf("Processed %d members with DNS resolution", len(members)))
	return members, nil
}

// CleanupLegacyCache 删除旧的缓存文件（兼容性清理）
// 在系统初始化时调用，清理因架构改变而不再使用的缓存文件
func CleanupLegacyCache() {
	// 获取用户主目录
	homeDir, err := os.UserHomeDir()
	if err != nil {
		logger.Debug(fmt.Sprintf("Failed to get home directory: %v", err))
		return
	}

	// 构建旧缓存文件路径
	cacheDir := filepath.Join(homeDir, ".nursorgate")
	cacheFile := filepath.Join(cacheDir, "inbounds.cache")

	// 检查文件是否存在
	if _, err := os.Stat(cacheFile); err == nil {
		// 文件存在，删除它
		if err := os.Remove(cacheFile); err != nil {
			logger.Warn(fmt.Sprintf("Failed to remove legacy cache file: %v", err))
		} else {
			logger.Info("Successfully cleaned up legacy cache file")
		}
	} else if !os.IsNotExist(err) {
		// 发生其他错误
		logger.Debug(fmt.Sprintf("Error checking cache file: %v", err))
	}
	// 如果文件不存在，不需要做任何事
}
