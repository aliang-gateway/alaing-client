package cmd

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"time"

	"nursor.org/nursorgate/common/logger"
	proxyConfig "nursor.org/nursorgate/processor/config"
	proxyRegistry "nursor.org/nursorgate/processor/proxy"
	runnerUtils "nursor.org/nursorgate/runner/utils"
)

// Config 完整配置结构
type Config struct {
	Engine       *EngineConfig          `json:"engine"`
	CurrentProxy string                 `json:"currentProxy"`
	CoreServer   string                 `json:"coreServer"`
	Proxies      map[string]interface{} `json:"proxies"`
}

// EngineConfig 引擎配置（对应 processor/config.EngineConf），以前的tun2socks配置key
type EngineConfig struct {
	MTU                      int    `json:"mtu"`
	Mark                     int    `json:"fwmark"`
	RestAPI                  string `json:"restapi"`
	Device                   string `json:"device"`
	LogLevel                 string `json:"loglevel"`
	Interface                string `json:"interface"`
	TCPModerateReceiveBuffer bool   `json:"tcp-moderate-receive-buffer"`
	TCPSendBufferSize        string `json:"tcp-send-buffer-size"`
	TCPReceiveBufferSize     string `json:"tcp-receive-buffer-size"`
	MulticastGroups          string `json:"multicast-groups"`
	TUNPreUp                 string `json:"tun-pre-up"`
	TUNPostUp                string `json:"tun-post-up"`
	UDPTimeout               string `json:"udp-timeout"` // 字符串格式，需要解析为 time.Duration
}

// LoadConfig 从文件加载配置
func LoadConfig(configPath string) (*Config, error) {
	data, err := ioutil.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return &config, nil
}

// ApplyConfig 应用配置到系统
func ApplyConfig(config *Config) error {
	// 1. 应用引擎配置
	if config.Engine != nil {
		if err := applyEngineConfig(config.Engine); err != nil {
			return fmt.Errorf("failed to apply engine config: %w", err)
		}
	}

	// 2. 应用核心服务器配置
	if config.CoreServer != "" {
		runnerUtils.SetServerHost(config.CoreServer)
		logger.Info(fmt.Sprintf("Core server set to: %s", config.CoreServer))
	}

	// 3. 应用代理配置
	if err := applyProxyConfigs(config.Proxies); err != nil {
		return fmt.Errorf("failed to apply proxy configs: %w", err)
	}

	// 4. 设置当前代理
	if config.CurrentProxy != "" {
		if err := proxyRegistry.GetRegistry().SetDefault(config.CurrentProxy); err != nil {
			logger.Warn(fmt.Sprintf("Failed to set current proxy '%s': %v", config.CurrentProxy, err))
		} else {
			logger.Info(fmt.Sprintf("Current proxy set to: %s", config.CurrentProxy))
		}
	}

	return nil
}

// applyEngineConfig 应用引擎配置
func applyEngineConfig(engineCfg *EngineConfig) error {
	// 解析 UDP 超时时间
	udpTimeout, err := time.ParseDuration(engineCfg.UDPTimeout)
	if err != nil {
		// 如果解析失败，使用默认值
		udpTimeout = 60 * time.Second
		logger.Warn(fmt.Sprintf("Failed to parse udp-timeout '%s', using default 60s", engineCfg.UDPTimeout))
	}

	// 转换为 processor/config.EngineConf
	engineConf := &proxyConfig.EngineConf{
		MTU:                      engineCfg.MTU,
		Mark:                     engineCfg.Mark,
		RestAPI:                  engineCfg.RestAPI,
		Device:                   engineCfg.Device,
		LogLevel:                 engineCfg.LogLevel,
		Interface:                engineCfg.Interface,
		TCPModerateReceiveBuffer: engineCfg.TCPModerateReceiveBuffer,
		TCPSendBufferSize:        engineCfg.TCPSendBufferSize,
		TCPReceiveBufferSize:     engineCfg.TCPReceiveBufferSize,
		MulticastGroups:          engineCfg.MulticastGroups,
		TUNPreUp:                 engineCfg.TUNPreUp,
		TUNPostUp:                engineCfg.TUNPostUp,
		UDPTimeout:               udpTimeout,
	}

	// 插入到配置系统
	proxyConfig.Insert(engineConf)
	logger.Info("Engine config applied successfully")
	return nil
}

// applyProxyConfigs 应用代理配置
func applyProxyConfigs(proxies map[string]interface{}) error {
	if len(proxies) == 0 {
		logger.Warn("No proxies configured")
		return nil
	}

	registry := proxyRegistry.GetRegistry()

	for name, proxyData := range proxies {
		// 将 interface{} 转换为 map[string]interface{}
		proxyMap, ok := proxyData.(map[string]interface{})
		if !ok {
			logger.Warn(fmt.Sprintf("Invalid proxy config for '%s': expected map, got %T", name, proxyData))
			continue
		}

		// 解析代理配置
		proxyCfg, err := parseProxyConfig(proxyMap)
		if err != nil {
			logger.Error(fmt.Sprintf("Failed to parse proxy config for '%s': %v", name, err))
			continue
		}

		// 注册到注册中心
		if err := registry.RegisterFromConfig(name, proxyCfg); err != nil {
			logger.Error(fmt.Sprintf("Failed to register proxy '%s': %v", name, err))
			continue
		}

		logger.Info(fmt.Sprintf("Proxy '%s' registered successfully", name))
	}

	return nil
}

// parseProxyConfig 从 map[string]interface{} 解析代理配置
func parseProxyConfig(proxyMap map[string]interface{}) (*proxyConfig.ProxyConfig, error) {
	cfg := &proxyConfig.ProxyConfig{}

	// 解析 type
	if typeVal, ok := proxyMap["type"].(string); ok {
		cfg.Type = typeVal
	} else {
		return nil, fmt.Errorf("missing or invalid 'type' field")
	}

	// 解析 is_default
	if isDefaultVal, ok := proxyMap["is_default"].(bool); ok {
		cfg.IsDefault = isDefaultVal
	}

	// 解析 is_door_proxy
	if isDoorProxyVal, ok := proxyMap["is_door_proxy"].(bool); ok {
		cfg.IsDoorProxy = isDoorProxyVal
	}

	// 根据类型解析具体配置
	switch cfg.Type {
	case "vless":
		if vlessData, ok := proxyMap["vless"].(map[string]interface{}); ok {
			vlessCfg, err := parseVLESSConfig(vlessData)
			if err != nil {
				return nil, fmt.Errorf("failed to parse VLESS config: %w", err)
			}
			cfg.VLESS = vlessCfg
		} else {
			return nil, fmt.Errorf("missing 'vless' config for vless type")
		}

	case "shadowsocks":
		if ssData, ok := proxyMap["shadowsocks"].(map[string]interface{}); ok {
			ssCfg, err := parseShadowsocksConfig(ssData)
			if err != nil {
				return nil, fmt.Errorf("failed to parse Shadowsocks config: %w", err)
			}
			cfg.Shadowsocks = ssCfg
		} else {
			return nil, fmt.Errorf("missing 'shadowsocks' config for shadowsocks type")
		}

	default:
		return nil, fmt.Errorf("unsupported proxy type: %s", cfg.Type)
	}

	return cfg, nil
}

// parseVLESSConfig 解析 VLESS 配置
func parseVLESSConfig(vlessMap map[string]interface{}) (*proxyConfig.VLESSConfig, error) {
	cfg := &proxyConfig.VLESSConfig{}

	// 解析基础字段
	if server, ok := vlessMap["server"].(string); ok {
		cfg.Server = server
	}
	if uuid, ok := vlessMap["uuid"].(string); ok {
		cfg.UUID = uuid
	}
	if flow, ok := vlessMap["flow"].(string); ok {
		cfg.Flow = flow
	}

	// 解析 TLS 配置
	if tlsEnabled, ok := vlessMap["tls_enabled"].(bool); ok {
		cfg.TLSEnabled = tlsEnabled
	}
	if sni, ok := vlessMap["sni"].(string); ok {
		cfg.SNI = sni
	}

	// 解析 REALITY 配置
	if realityEnabled, ok := vlessMap["reality_enabled"].(bool); ok {
		cfg.RealityEnabled = realityEnabled
	}
	if publicKey, ok := vlessMap["public_key"].(string); ok {
		cfg.PublicKey = publicKey
	}
	if shortID, ok := vlessMap["short_id"].(string); ok {
		cfg.ShortID = shortID
	}
	if shortIDList, ok := vlessMap["short_id_list"].(string); ok {
		cfg.ShortIDList = shortIDList
	}

	return cfg, nil
}

// parseShadowsocksConfig 解析 Shadowsocks 配置
func parseShadowsocksConfig(ssMap map[string]interface{}) (*proxyConfig.ShadowsocksConfig, error) {
	cfg := &proxyConfig.ShadowsocksConfig{}

	if server, ok := ssMap["server"].(string); ok {
		cfg.Server = server
	}
	if method, ok := ssMap["method"].(string); ok {
		cfg.Method = method
	}
	if password, ok := ssMap["password"].(string); ok {
		cfg.Password = password
	}
	if obfsMode, ok := ssMap["obfs_mode"].(string); ok {
		cfg.ObfsMode = obfsMode
	}
	if obfsHost, ok := ssMap["obfs_host"].(string); ok {
		cfg.ObfsHost = obfsHost
	}

	return cfg, nil
}

// LoadAndApplyConfig 加载并应用配置文件
func LoadAndApplyConfig(configPath string) error {
	// 检查文件是否存在
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		logger.Warn(fmt.Sprintf("Config file not found: %s, using defaults", configPath))
		return nil
	}

	// 加载配置
	config, err := LoadConfig(configPath)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// 应用配置
	if err := ApplyConfig(config); err != nil {
		return fmt.Errorf("failed to apply config: %w", err)
	}

	logger.Info(fmt.Sprintf("Config loaded and applied successfully from: %s", configPath))
	return nil
}

// FetchConfigFromRemote 从远程服务器获取配置
func FetchConfigFromRemote(token string, serverURL string) (*Config, error) {
	if serverURL == "" {
		// 使用默认服务器地址
		serverURL = runnerUtils.GetServerHost()
		if serverURL == "" {
			serverURL = "https://api2.nursor.org:12235" // 默认服务器
		}
	}

	// 确保 URL 格式正确
	if !strings.HasPrefix(serverURL, "http://") && !strings.HasPrefix(serverURL, "https://") {
		serverURL = "https://" + serverURL
	}

	// 构建请求 URL
	url := fmt.Sprintf("%s/api/config?token=%s", serverURL, token)

	// 创建 HTTP 请求
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// 设置请求头
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	req.Header.Set("Content-Type", "application/json")

	// 发送请求
	client := &http.Client{
		Timeout: 30 * time.Second,
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch config from remote: %w", err)
	}
	defer resp.Body.Close()

	// 检查响应状态
	if resp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
		return nil, fmt.Errorf("remote server returned status %d: %s", resp.StatusCode, string(body))
	}

	// 读取响应体
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// 解析 JSON
	var config Config
	if err := json.Unmarshal(body, &config); err != nil {
		return nil, fmt.Errorf("failed to parse remote config: %w", err)
	}

	return &config, nil
}

// FetchAndApplyConfigFromRemote 从远程获取配置并应用
func FetchAndApplyConfigFromRemote(token string, serverURL string) error {
	// 从远程获取配置
	config, err := FetchConfigFromRemote(token, serverURL)
	if err != nil {
		return fmt.Errorf("failed to fetch config from remote: %w", err)
	}

	// 应用配置
	if err := ApplyConfig(config); err != nil {
		return fmt.Errorf("failed to apply config: %w", err)
	}

	logger.Info("Config fetched and applied successfully from remote server")
	return nil
}

// LoadConfigFromBytes 从字节数组加载配置（用于从远程获取的配置）
func LoadConfigFromBytes(data []byte) (*Config, error) {
	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}
	return &config, nil
}

// SaveConfigToFile 保存配置到文件
func SaveConfigToFile(config *Config, filePath string) error {
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := ioutil.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}
