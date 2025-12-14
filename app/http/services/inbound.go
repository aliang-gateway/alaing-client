package services

import (
	"nursor.org/nursorgate/outbound"
)

// InboundService provides proxyserver proxy management services
type InboundService struct{}

// NewInboundService creates a new InboundService instance
func NewInboundService() *InboundService {
	return &InboundService{}
}

// GetInboundStatus returns current door proxy status (simplified after removing cache layer)
func (s *InboundService) GetInboundStatus() map[string]interface{} {
	registry := outbound.GetRegistry()
	if registry == nil {
		return map[string]interface{}{
			"status": "error",
			"error":  "proxy registry not available",
		}
	}

	// 获取门代理的当前成员数量
	doorGroup := registry.GetDoorGroup()
	memberCount := 0
	if doorGroup != nil {
		memberCount = doorGroup.Count()
	}

	return map[string]interface{}{
		"status":        "success",
		"proxy_count":   memberCount,
		"architecture":  "direct_api_fetch", // 说明使用直接API获取，无缓存
		"cache_enabled": false,              // 架构简化后不使用缓存
	}
}
