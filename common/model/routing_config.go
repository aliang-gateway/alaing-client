package model

import (
	"encoding/json"
	"fmt"
	"net"
	"regexp"
	"strings"
	"time"
)

// RuleType 定义规则类型
type RuleType string

const (
	RuleTypeDomain RuleType = "domain" // 域名匹配
	RuleTypeIP     RuleType = "ip"     // IP段匹配 (CIDR)
	RuleTypeGeoIP  RuleType = "geoip"  // GeoIP国家代码匹配
)

// RoutingRule 单条路由规则
type RoutingRule struct {
	ID        string    `json:"id"`         // 规则唯一标识，格式: rule_{type}_{timestamp}
	Type      RuleType  `json:"type"`       // 规则类型: domain, ip, geoip
	Condition string    `json:"condition"`  // 匹配条件
	Enabled   bool      `json:"enabled"`    // 是否启用
	CreatedAt time.Time `json:"created_at"` // 创建时间
}

// RoutingRuleSet 路由规则集
type RoutingRuleSet struct {
	Rules []RoutingRule `json:"rules"` // 规则列表
}

// RulesSettings 规则全局设置
type RulesSettings struct {
	GeoIPEnabled    bool `json:"geoip_enabled"`     // 是否启用GeoIP判断
	NoneLaneEnabled bool `json:"none_lane_enabled"` // 是否启用NoneLane代理
}

// RoutingRulesConfig 统一的路由规则配置模型
type RoutingRulesConfig struct {
	ToDoor    RoutingRuleSet `json:"to_door"`    // To Door代理规则集
	BlackList RoutingRuleSet `json:"black_list"` // 黑名单规则集(内网IP段)
	NoneLane  RoutingRuleSet `json:"none_lane"`  // NoneLane代理规则集
	Settings  RulesSettings  `json:"settings"`   // 全局设置
}

// Validate 验证配置完整性和正确性
func (rc *RoutingRulesConfig) Validate() error {
	// 验证规则ID唯一性
	idSet := make(map[string]bool)

	// 验证所有规则集
	allRuleSets := []struct {
		name string
		set  RoutingRuleSet
	}{
		{"to_door", rc.ToDoor},
		{"black_list", rc.BlackList},
		{"none_lane", rc.NoneLane},
	}

	for _, rs := range allRuleSets {
		for i, rule := range rs.set.Rules {
			// 检查ID唯一性
			if idSet[rule.ID] {
				return fmt.Errorf("duplicate rule ID: %s in %s", rule.ID, rs.name)
			}
			idSet[rule.ID] = true

			// 检查ID格式
			if rule.ID == "" {
				return fmt.Errorf("empty rule ID at %s.rules[%d]", rs.name, i)
			}

			// 检查Type有效性
			if !isValidRuleType(rule.Type) {
				return fmt.Errorf("invalid rule type '%s' at %s.rules[%d]", rule.Type, rs.name, i)
			}

			// 检查Condition非空
			if rule.Condition == "" {
				return fmt.Errorf("empty condition at %s.rules[%d]", rs.name, i)
			}

			// 根据类型验证Condition格式
			if err := validateRuleCondition(rule); err != nil {
				return fmt.Errorf("invalid condition at %s.rules[%d]: %w", rs.name, i, err)
			}
		}
	}

	return nil
}

// ToJSON 序列化为JSON
func (rc *RoutingRulesConfig) ToJSON() ([]byte, error) {
	return json.MarshalIndent(rc, "", "  ")
}

// NewRoutingRulesConfigFromJSON 从JSON反序列化并验证
func NewRoutingRulesConfigFromJSON(data []byte) (*RoutingRulesConfig, error) {
	var rc RoutingRulesConfig
	if err := json.Unmarshal(data, &rc); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	if err := rc.Validate(); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	return &rc, nil
}

// isValidRuleType 检查规则类型是否有效
func isValidRuleType(t RuleType) bool {
	switch t {
	case RuleTypeDomain, RuleTypeIP, RuleTypeGeoIP:
		return true
	default:
		return false
	}
}

// validateRuleCondition 根据规则类型验证条件格式
func validateRuleCondition(rule RoutingRule) error {
	switch rule.Type {
	case RuleTypeDomain:
		return validateDomainPattern(rule.Condition)
	case RuleTypeIP:
		return validateCIDR(rule.Condition)
	case RuleTypeGeoIP:
		return validateCountryCode(rule.Condition)
	default:
		return fmt.Errorf("unknown rule type: %s", rule.Type)
	}
}

// validateDomainPattern 验证域名模式 (支持通配符 *.example.com 或 example.com)
func validateDomainPattern(pattern string) error {
	if pattern == "" {
		return fmt.Errorf("empty domain pattern")
	}

	// 允许通配符前缀 *.
	pattern = strings.TrimPrefix(pattern, "*.")

	// 基本域名验证 (允许字母、数字、连字符和点)
	domainRegex := regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(\.[a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$`)
	if !domainRegex.MatchString(pattern) {
		return fmt.Errorf("invalid domain pattern format")
	}

	return nil
}

// validateCIDR 验证CIDR格式 (如 192.168.0.0/16)
func validateCIDR(cidr string) error {
	_, _, err := net.ParseCIDR(cidr)
	if err != nil {
		return fmt.Errorf("invalid CIDR format: %w", err)
	}
	return nil
}

// validateCountryCode 验证ISO 3166-1 alpha-2国家代码 (2个字符)
func validateCountryCode(code string) error {
	if len(code) != 2 {
		return fmt.Errorf("country code must be 2 characters (ISO 3166-1 alpha-2)")
	}

	// 检查是否都是字母
	for _, c := range code {
		if (c < 'A' || c > 'Z') && (c < 'a' || c > 'z') {
			return fmt.Errorf("country code must contain only letters")
		}
	}

	return nil
}

// NewDefaultRoutingRulesConfig 创建默认配置
func NewDefaultRoutingRulesConfig() *RoutingRulesConfig {
	return &RoutingRulesConfig{
		ToDoor: RoutingRuleSet{
			Rules: []RoutingRule{},
		},
		BlackList: RoutingRuleSet{
			Rules: []RoutingRule{},
		},
		NoneLane: RoutingRuleSet{
			Rules: []RoutingRule{},
		},
		Settings: RulesSettings{
			GeoIPEnabled:    true,
			NoneLaneEnabled: false,
		},
	}
}

// GenerateRuleID 生成规则ID
func GenerateRuleID(ruleType RuleType) string {
	timestamp := time.Now().Unix()
	return fmt.Sprintf("rule_%s_%d", ruleType, timestamp)
}
