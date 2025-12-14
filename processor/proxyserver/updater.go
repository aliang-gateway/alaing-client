package proxyserver

import (
	"fmt"

	"nursor.org/nursorgate/common/logger"
	"nursor.org/nursorgate/outbound"
	"nursor.org/nursorgate/processor/config"
)

// UpdateDoorProxies 获取并更新Door代理成员（简化版：无缓存层，直接API获取）
// 新策略：直接从API获取DoorProxyMember，跳过所有中间层
func UpdateDoorProxies(accessToken string) error {
	// 直接获取DoorProxyMember，跳过所有缓存层
	members, err := FetchInbounds(accessToken)
	if err != nil {
		return fmt.Errorf("failed to fetch door proxy members: %w", err)
	}

	if len(members) == 0 {
		logger.Warn("API returned empty door proxy member list")
		return nil
	}

	logger.Info(fmt.Sprintf("Successfully fetched %d door proxy members from API", len(members)))

	// 直接注册到Door代理组（集成DNS处理）
	return registerInboundsToDoor(members)
}

// registerInboundsToDoor 处理DoorProxyMember并注册到Door代理组（集成DNS处理和配置更新）
func registerInboundsToDoor(members []config.DoorProxyMember) error {
	if len(members) == 0 {
		return fmt.Errorf("no members to register")
	}

	// DNS预解析和延迟测试（从converter.go迁移过来）
	processedMembers, err := processMembersWithDNS(members)
	if err != nil {
		logger.Warn(fmt.Sprintf("DNS processing failed, using original members: %v", err))
		processedMembers = members
	}

	// 创建Door配置
	doorConfig := &config.DoorProxyConfig{
		Type:    "door",
		Members: processedMembers,
	}

	// 1. 更新全局配置（配置成为唯一真实来源）
	if err := config.UpdateGlobalDoorProxyConfig(doorConfig); err != nil {
		logger.Warn(fmt.Sprintf("Failed to update global config: %v", err))
		// 不返回错误，继续执行
	}

	// 2. 注册到代理组（运行时实例）
	registry := outbound.GetRegistry()
	if registry == nil {
		return fmt.Errorf("proxy registry is not available")
	}

	logger.Info(fmt.Sprintf("Registering %d proxy members to Door proxy", len(processedMembers)))
	if err := registry.RegisterDoorFromConfig(doorConfig); err != nil {
		return err
	}

	// 3. 同步DNS解析器
	UpdateGlobalResolverWithDoorConfig(doorConfig)

	return nil
}
