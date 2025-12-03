package proxy

import (
	"fmt"
	"sync"

	"nursor.org/nursorgate/common/logger"
	"nursor.org/nursorgate/outbound/proxy"
	proxyConfig "nursor.org/nursorgate/processor/config"
)

// Registry 代理注册中心，线程安全
type Registry struct {
	mu          sync.RWMutex
	proxies     map[string]proxy.Proxy // 代理实例映射，key 为代理名称
	defaultName string                 // 默认代理名称
	doorName    string                 // 门代理名称
}

var (
	globalRegistry *Registry
	once           sync.Once
)

// GetRegistry 获取全局代理注册中心（单例）
func GetRegistry() *Registry {
	once.Do(func() {
		globalRegistry = &Registry{
			proxies: make(map[string]proxy.Proxy),
		}
	})
	return globalRegistry
}

// Register 注册一个代理实例
// name: 代理名称，用于后续查找和切换
// p: 代理实例
func (r *Registry) Register(name string, p proxy.Proxy) error {
	if name == "" {
		return fmt.Errorf("proxy name cannot be empty")
	}
	if p == nil {
		return fmt.Errorf("proxy instance cannot be nil")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	// 如果已存在同名代理，记录警告但允许覆盖
	if _, exists := r.proxies[name]; exists {
		logger.Warn(fmt.Sprintf("Proxy '%s' already exists, will be replaced", name))
	}

	r.proxies[name] = p
	logger.Info(fmt.Sprintf("Proxy '%s' registered successfully (type: %s, addr: %s)",
		name, p.Proto().String(), p.Addr()))
	return nil
}

// Unregister 注销一个代理
func (r *Registry) Unregister(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.proxies[name]; !exists {
		return fmt.Errorf("proxy '%s' not found", name)
	}

	delete(r.proxies, name)

	// 如果注销的是默认代理或门代理，清除标记
	if r.defaultName == name {
		r.defaultName = ""
		logger.Warn(fmt.Sprintf("Default proxy '%s' was unregistered", name))
	}
	if r.doorName == name {
		r.doorName = ""
		logger.Warn(fmt.Sprintf("Door proxy '%s' was unregistered", name))
	}

	logger.Info(fmt.Sprintf("Proxy '%s' unregistered", name))
	return nil
}

// Get 根据名称获取代理实例
func (r *Registry) Get(name string) (proxy.Proxy, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	p, exists := r.proxies[name]
	if !exists {
		return nil, fmt.Errorf("proxy '%s' not found", name)
	}
	return p, nil
}

// GetDefault 获取默认代理
func (r *Registry) GetDefault() (proxy.Proxy, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.defaultName == "" {
		return nil, fmt.Errorf("no default proxy set")
	}

	p, exists := r.proxies[r.defaultName]
	if !exists {
		return nil, fmt.Errorf("default proxy '%s' not found", r.defaultName)
	}
	return p, nil
}

// GetDoor 获取门代理
func (r *Registry) GetDoor() (proxy.Proxy, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.doorName == "" {
		return nil, fmt.Errorf("no door proxy set")
	}

	p, exists := r.proxies[r.doorName]
	if !exists {
		return nil, fmt.Errorf("door proxy '%s' not found", r.doorName)
	}
	return p, nil
}

// SetDefault 设置默认代理
func (r *Registry) SetDefault(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.proxies[name]; !exists {
		return fmt.Errorf("proxy '%s' not found, cannot set as default", name)
	}

	oldName := r.defaultName
	r.defaultName = name
	logger.Info(fmt.Sprintf("Default proxy changed from '%s' to '%s'", oldName, name))
	return nil
}

// SetDoor 设置门代理
func (r *Registry) SetDoor(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.proxies[name]; !exists {
		return fmt.Errorf("proxy '%s' not found, cannot set as door proxy", name)
	}

	oldName := r.doorName
	r.doorName = name
	logger.Info(fmt.Sprintf("Door proxy changed from '%s' to '%s'", oldName, name))
	return nil
}

// List 列出所有已注册的代理名称
func (r *Registry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.proxies))
	for name := range r.proxies {
		names = append(names, name)
	}
	return names
}

// ListWithInfo 列出所有代理及其信息
func (r *Registry) ListWithInfo() map[string]ProxyInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	info := make(map[string]ProxyInfo)
	for name, p := range r.proxies {
		info[name] = ProxyInfo{
			Name:        name,
			Type:        p.Proto().String(),
			Addr:        p.Addr(),
			IsDefault:   name == r.defaultName,
			IsDoorProxy: name == r.doorName,
		}
	}
	return info
}

// ProxyInfo 代理信息
type ProxyInfo struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Addr        string `json:"addr"`
	IsDefault   bool   `json:"is_default"`
	IsDoorProxy bool   `json:"is_door_proxy"`
}

// Count 返回已注册的代理数量
func (r *Registry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.proxies)
}

// Clear 清空所有代理（谨慎使用）
func (r *Registry) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.proxies = make(map[string]proxy.Proxy)
	r.defaultName = ""
	r.doorName = ""
	logger.Warn("All proxies cleared")
}

// InitializeFromConfig 根据全局配置初始化代理
// 从 processor/config 包中读取配置并创建代理实例
func (r *Registry) InitializeFromConfig() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// 初始化 VLESS 代理
	if vlessCfg := proxyConfig.GetVLESSConfig(); vlessCfg != nil {
		p, err := proxyConfig.CreateVLESSProxyFromConfig(vlessCfg)
		if err != nil {
			logger.Error(fmt.Sprintf("Failed to create VLESS proxy from config: %v", err))
		} else {
			r.proxies["vless-default"] = p
			if r.defaultName == "" {
				r.defaultName = "vless-default"
			}
			if r.doorName == "" {
				r.doorName = "vless-default"
			}
			logger.Info("VLESS proxy initialized from config: vless-default")
		}
	}

	// 初始化 Shadowsocks 代理
	if ssCfg := proxyConfig.GetShadowsocksConfig(); ssCfg != nil {
		p, err := proxyConfig.CreateShadowsocksProxyFromConfig(ssCfg)
		if err != nil {
			logger.Error(fmt.Sprintf("Failed to create Shadowsocks proxy from config: %v", err))
		} else {
			r.proxies["shadowsocks-default"] = p
			if r.defaultName == "" {
				r.defaultName = "shadowsocks-default"
			}
			logger.Info("Shadowsocks proxy initialized from config: shadowsocks-default")
		}
	}

	if len(r.proxies) == 0 {
		return fmt.Errorf("no proxy configuration found, cannot initialize")
	}

	return nil
}

// RegisterFromConfig 根据配置注册代理（支持自定义名称）
func (r *Registry) RegisterFromConfig(name string, cfg *proxyConfig.ProxyConfig) error {
	if name == "" {
		return fmt.Errorf("proxy name cannot be empty")
	}

	var p proxy.Proxy
	var err error

	switch cfg.Type {
	case "vless":
		if cfg.VLESS == nil {
			return fmt.Errorf("VLESS config is required")
		}
		p, err = proxyConfig.CreateVLESSProxyFromConfig(cfg.VLESS)
	case "shadowsocks":
		if cfg.Shadowsocks == nil {
			return fmt.Errorf("Shadowsocks config is required")
		}
		p, err = proxyConfig.CreateShadowsocksProxyFromConfig(cfg.Shadowsocks)
	default:
		return fmt.Errorf("unsupported proxy type: %s", cfg.Type)
	}

	if err != nil {
		return fmt.Errorf("failed to create proxy: %w", err)
	}

	if err := r.Register(name, p); err != nil {
		return err
	}

	// 根据配置设置默认代理和门代理
	if cfg.IsDefault {
		if err := r.SetDefault(name); err != nil {
			return err
		}
	}
	if cfg.IsDoorProxy {
		if err := r.SetDoor(name); err != nil {
			return err
		}
	}

	return nil
}
