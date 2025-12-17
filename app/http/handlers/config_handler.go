package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"nursor.org/nursorgate/app/http/common"
	"nursor.org/nursorgate/common/logger"
	"nursor.org/nursorgate/common/model"

	"github.com/nacos-group/nacos-sdk-go/clients/config_client"
	"github.com/nacos-group/nacos-sdk-go/vo"
)

const (
	NacosRoutingRulesDataID = "routing-rules"
	NacosDefaultGroup       = "DEFAULT_GROUP"
)

// ConfigHandler 配置管理处理器
type ConfigHandler struct {
	nacosClient interface{}
}

// NewConfigHandler 创建新的配置处理器实例
func NewConfigHandler(nacosClient interface{}) *ConfigHandler {
	return &ConfigHandler{
		nacosClient: nacosClient,
	}
}

// HandleGetRoutingConfig 获取路由规则配置
// GET /api/config/routing
func (h *ConfigHandler) HandleGetRoutingConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		common.Error(w, http.StatusMethodNotAllowed, "Method not allowed", nil)
		return
	}

	// 类型断言获取Nacos客户端
	nacosClient, ok := h.nacosClient.(config_client.IConfigClient)
	if !ok || nacosClient == nil {
		common.ErrorServiceUnavailable(w, "Configuration service is not available")
		return
	}

	// 从Nacos读取配置
	configContent, err := nacosClient.GetConfig(vo.ConfigParam{
		DataId: NacosRoutingRulesDataID,
		Group:  NacosDefaultGroup,
	})

	if err != nil {
		logger.Error(fmt.Sprintf("Failed to get routing config from Nacos: %v", err))
		common.ErrorInternalServer(w, "Failed to load configuration from Nacos", map[string]interface{}{
			"error": err.Error(),
		})
		return
	}

	// 如果配置为空，返回默认配置
	if configContent == "" {
		logger.Warn("Routing config is empty in Nacos, returning default config")
		defaultConfig := model.NewDefaultRoutingRulesConfig()
		common.Success(w, defaultConfig)
		return
	}

	// 反序列化为RoutingRulesConfig对象
	config, err := model.NewRoutingRulesConfigFromJSON([]byte(configContent))
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to parse routing config: %v", err))
		common.ErrorInternalServer(w, "Invalid configuration format in Nacos", map[string]interface{}{
			"error": err.Error(),
		})
		return
	}

	common.Success(w, config)
}

// HandleUpdateRoutingConfig 更新路由规则配置
// POST /api/config/routing
func (h *ConfigHandler) HandleUpdateRoutingConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		common.Error(w, http.StatusMethodNotAllowed, "Method not allowed", nil)
		return
	}

	// 类型断言获取Nacos客户端
	nacosClient, ok := h.nacosClient.(config_client.IConfigClient)
	if !ok || nacosClient == nil {
		common.ErrorServiceUnavailable(w, "Configuration service is not available")
		return
	}

	// 解析请求体
	var config model.RoutingRulesConfig
	if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
		common.ErrorBadRequest(w, "Invalid JSON format", map[string]interface{}{
			"error": err.Error(),
		})
		return
	}

	// 验证配置
	if err := config.Validate(); err != nil {
		common.ErrorBadRequest(w, "Configuration validation failed", map[string]interface{}{
			"error": err.Error(),
		})
		return
	}

	// 序列化为JSON
	configJSON, err := config.ToJSON()
	if err != nil {
		common.ErrorInternalServer(w, "Failed to serialize configuration", map[string]interface{}{
			"error": err.Error(),
		})
		return
	}

	// 写入Nacos
	success, err := nacosClient.PublishConfig(vo.ConfigParam{
		DataId:  NacosRoutingRulesDataID,
		Group:   NacosDefaultGroup,
		Content: string(configJSON),
	})

	if err != nil || !success {
		logger.Error(fmt.Sprintf("Failed to publish routing config to Nacos: %v", err))
		common.ErrorInternalServer(w, "Failed to save configuration to Nacos", map[string]interface{}{
			"error": err,
		})
		return
	}

	logger.Info("Routing configuration updated successfully")
	common.Success(w, map[string]interface{}{
		"message":    "Configuration updated successfully",
		"applied_at": fmt.Sprintf("%d", r.Context().Value("timestamp")),
	})
}

// HandleToggleRuleStatus 切换规则启用/禁用状态
// PUT /api/config/routing/rules/{ruleId}/toggle
func (h *ConfigHandler) HandleToggleRuleStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		common.Error(w, http.StatusMethodNotAllowed, "Method not allowed", nil)
		return
	}

	// 类型断言获取Nacos客户端
	nacosClient, ok := h.nacosClient.(config_client.IConfigClient)
	if !ok || nacosClient == nil {
		common.ErrorServiceUnavailable(w, "Configuration service is not available")
		return
	}

	// 从URL路径中提取ruleId
	pathParts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/config/routing/rules/"), "/")
	if len(pathParts) < 2 {
		common.ErrorBadRequest(w, "Invalid URL format", nil)
		return
	}
	ruleID := pathParts[0]

	// 解析请求体
	var req struct {
		Enabled bool `json:"enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		common.ErrorBadRequest(w, "Invalid JSON format", map[string]interface{}{
			"error": err.Error(),
		})
		return
	}

	// 从Nacos读取当前配置
	configContent, err := nacosClient.GetConfig(vo.ConfigParam{
		DataId: NacosRoutingRulesDataID,
		Group:  NacosDefaultGroup,
	})

	if err != nil {
		logger.Error(fmt.Sprintf("Failed to get routing config from Nacos: %v", err))
		common.ErrorInternalServer(w, "Failed to load configuration", map[string]interface{}{
			"error": err.Error(),
		})
		return
	}

	// 反序列化
	config, err := model.NewRoutingRulesConfigFromJSON([]byte(configContent))
	if err != nil {
		common.ErrorInternalServer(w, "Invalid configuration format", map[string]interface{}{
			"error": err.Error(),
		})
		return
	}

	// 查找并更新规则
	ruleFound := false
	allRuleSets := []struct {
		name string
		set  *model.RoutingRuleSet
	}{
		{"to_door", &config.ToDoor},
		{"black_list", &config.BlackList},
		{"none_lane", &config.NoneLane},
	}

	for _, rs := range allRuleSets {
		for i := range rs.set.Rules {
			if rs.set.Rules[i].ID == ruleID {
				rs.set.Rules[i].Enabled = req.Enabled
				ruleFound = true
				logger.Info(fmt.Sprintf("Rule %s in %s toggled to enabled=%v", ruleID, rs.name, req.Enabled))
				break
			}
		}
		if ruleFound {
			break
		}
	}

	if !ruleFound {
		common.ErrorNotFound(w, fmt.Sprintf("Rule with id '%s' not found", ruleID))
		return
	}

	// 序列化并写回Nacos
	configJSON, err := config.ToJSON()
	if err != nil {
		common.ErrorInternalServer(w, "Failed to serialize configuration", map[string]interface{}{
			"error": err.Error(),
		})
		return
	}

	success, err := nacosClient.PublishConfig(vo.ConfigParam{
		DataId:  NacosRoutingRulesDataID,
		Group:   NacosDefaultGroup,
		Content: string(configJSON),
	})

	if err != nil || !success {
		logger.Error(fmt.Sprintf("Failed to publish updated config to Nacos: %v", err))
		common.ErrorInternalServer(w, "Failed to save configuration", map[string]interface{}{
			"error": err,
		})
		return
	}

	common.Success(w, map[string]interface{}{
		"message": "Rule toggled successfully",
		"rule_id": ruleID,
		"enabled": req.Enabled,
	})
}
