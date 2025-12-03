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
	runnerUtils "nursor.org/nursorgate/inbound/tun/runner/utils"
	proxyConfig "nursor.org/nursorgate/processor/config"
	proxyRegistry "nursor.org/nursorgate/processor/proxy"
)

// Config 完整配置结构
type Config struct {
	Engine       *EngineConfig                       `json:"engine"`
	CurrentProxy string                              `json:"currentProxy"`
	CoreServer   string                              `json:"coreServer"`
	Proxies      map[string]*proxyConfig.ProxyConfig `json:"proxies"`
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

	// 3. 注册默认的 direct 代理
	registry := proxyRegistry.GetRegistry()
	if err := registry.RegisterDefault(); err != nil {
		logger.Warn(fmt.Sprintf("Failed to register default direct proxy: %v", err))
	}

	// 4. 应用代理配置
	if err := applyProxyConfigs(config.Proxies); err != nil {
		return fmt.Errorf("failed to apply proxy configs: %w", err)
	}

	// 5. 设置当前代理（如果未设置，使用 direct）
	currentProxy := config.CurrentProxy
	if currentProxy == "" {
		currentProxy = "direct"
	}
	if err := registry.SetDefault(currentProxy); err != nil {
		logger.Warn(fmt.Sprintf("Failed to set current proxy '%s': %v", currentProxy, err))
		// 如果设置失败，回退到 direct
		if err := registry.SetDefault("direct"); err != nil {
			logger.Error(fmt.Sprintf("Failed to fallback to direct proxy: %v", err))
		}
	} else {
		logger.Info(fmt.Sprintf("Current proxy set to: %s", currentProxy))
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

// applyProxyConfigs 应用代理配置到注册中心
func applyProxyConfigs(proxies map[string]*proxyConfig.ProxyConfig) error {
	if len(proxies) == 0 {
		logger.Warn("No proxies configured")
		return nil
	}

	registry := proxyRegistry.GetRegistry()

	for name, cfg := range proxies {
		if cfg == nil {
			logger.Warn(fmt.Sprintf("Nil proxy config for '%s', skipping", name))
			continue
		}

		// 验证配置
		if err := cfg.Validate(); err != nil {
			logger.Error(fmt.Sprintf("Invalid config for proxy '%s': %v", name, err))
			continue
		}

		// 注册代理（创建实例 + 存储配置）
		if err := registry.RegisterFromConfig(name, cfg); err != nil {
			logger.Error(fmt.Sprintf("Failed to register proxy '%s': %v", name, err))
			continue
		}

		logger.Info(fmt.Sprintf("Proxy '%s' registered successfully", name))
	}

	return nil
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
