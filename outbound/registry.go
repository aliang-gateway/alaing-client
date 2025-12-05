package outbound

import (
	"fmt"
	"sync"

	"nursor.org/nursorgate/common/logger"
	"nursor.org/nursorgate/outbound/proxy"
	"nursor.org/nursorgate/outbound/proxy/direct"
	"nursor.org/nursorgate/outbound/proxy/nonelane"
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

// RegisterDefault 注册默认的 direct 代理
// 如果已经存在，则不覆盖
func (r *Registry) RegisterDefault() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// ��果已经存在 direct 代理，直接返回
	if _, exists := r.proxies["direct"]; exists {
		return nil
	}

	// 创建并注册 direct 代理
	directProxy := direct.NewDirect()
	r.proxies["direct"] = directProxy
	logger.Info("Default direct proxy registered")
	return nil
}

// RegisterNonelane 注册默认的 nonelane 代理
// 如果已经存在，则不覆盖
// serverAddr: nonelane 服务器地址，如果为空则使用默认值
func (r *Registry) RegisterNonelane(serverAddr string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// 如果已经存在 nonelane 代理，直接返回
	if _, exists := r.proxies["nonelane"]; exists {
		return nil
	}

	// 如果没有提供地址，使用默认值
	if serverAddr == "" {
		serverAddr = "ai-gateway.nursor.org:443"
		logger.Debug("Using default nonelane server address")
	}

	// 创建 nonelane 配置
	config := nonelane.DefaultConfig(serverAddr)

	// 创建并注册 nonelane 代理
	nonelaneProxy, err := nonelane.NewNonelane(config)
	if err != nil {
		return fmt.Errorf("failed to create nonelane proxy: %w", err)
	}

	r.proxies["nonelane"] = nonelaneProxy
	logger.Info(fmt.Sprintf("Default nonelane proxy registered (server: %s)", serverAddr))
	return nil
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

// GetDefaultName 获取默认代理的名称
func (r *Registry) GetDefaultName() string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.defaultName
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

// GetNonelane 获取 nonelane 代理
// 如果 nonelane 代理未注册，返回错误
func (r *Registry) GetNonelane() (proxy.Proxy, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	p, exists := r.proxies["nonelane"]
	if !exists {
		return nil, fmt.Errorf("nonelane proxy not found, please register it first")
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
			IsNonelane:  name == "nonelane",
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
	IsNonelane  bool   `json:"is_nonelane"`
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

// RegisterFromConfig 根据配置注册代理（支持自定义名称）
// 使用factory模式创建代理实例，并将配置存储在ConfigStore中
func (r *Registry) RegisterFromConfig(name string, cfg *proxyConfig.ProxyConfig) error {
	if name == "" {
		return fmt.Errorf("proxy name cannot be empty")
	}
	if cfg == nil {
		return fmt.Errorf("config cannot be nil")
	}

	// 验证配置
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("invalid config: %w", err)
	}

	// 使用factory创建代理实例
	p, err := proxyConfig.CreateProxyFromConfig(cfg)
	if err != nil {
		return fmt.Errorf("failed to create proxy: %w", err)
	}

	// 在Registry中注册实例
	if err := r.Register(name, p); err != nil {
		return err
	}

	// 将配置存储在ConfigStore中
	if err := proxyConfig.GetConfigStore().Set(name, cfg); err != nil {
		logger.Warn(fmt.Sprintf("Failed to store config for '%s': %v", name, err))
		// 不因配置存储失败而中止注册
	}

	// 根据配置设置默认代理和门代理
	if cfg.IsDefault {
		if err := r.SetDefault(name); err != nil {
			return fmt.Errorf("failed to set default proxy: %w", err)
		}
	}
	if cfg.IsDoorProxy {
		if err := r.SetDoor(name); err != nil {
			return fmt.Errorf("failed to set door proxy: %w", err)
		}
	}

	return nil
}
