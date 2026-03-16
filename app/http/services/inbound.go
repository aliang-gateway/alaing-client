package services

import (
	"nursor.org/nursorgate/processor/config"
)

// InboundService provides inbound proxy management services
type InboundService struct{}

// NewInboundService creates a new InboundService instance
func NewInboundService() *InboundService {
	return &InboundService{}
}

// GetInboundStatus returns current door proxy status from global configuration
func (s *InboundService) GetInboundStatus() map[string]interface{} {
	// 从全局配置获取门代理的当前成员数量（配置为单一真实来源）
	memberCount := config.GetDoorProxyMemberCount()

	return map[string]interface{}{
		"status":        "success",
		"proxy_count":   memberCount,
		"source":        "configuration", // 说明数据来自配置而非运行时
		"cache_enabled": false,           // 架构简化后不使用缓存
	}
}
