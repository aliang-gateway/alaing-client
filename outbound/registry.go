package outbound

import (
	"fmt"
	"strings"
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
	doorGroup   *DoorProxyGroup        // 门代理集合
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

	// 如果注销的是默认代理，清除标记
	if r.defaultName == name {
		r.defaultName = ""
		logger.Warn(fmt.Sprintf("Default proxy '%s' was unregistered", name))
	}

	logger.Info(fmt.Sprintf("Proxy '%s' unregistered", name))
	return nil
}

// Get 根据名称获取代理实例
// 支持两种查询方式：
// 1. 普通代理: "direct", "nonelane", 或其他自定义代理名称
// 2. Door 代理成员: "door:ShowName" 格式，例如 "door:日本 Tokyo"
func (r *Registry) Get(name string) (proxy.Proxy, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// 第一级: 尝试在普通 proxies 中查找
	p, exists := r.proxies[name]
	if exists {
		return p, nil
	}

	// 第二级: 检查是否为 door 代理成员格式 "door:ShowName"
	if strings.HasPrefix(name, "door:") {
		// 处理 door 成员查询
		showName := strings.TrimPrefix(name, "door:")

		// 检查 ShowName 是否为空
		if showName == "" {
			return nil, fmt.Errorf("invalid door proxy name '%s' - empty show name", name)
		}

		// 检查 doorGroup 是否存在
		if r.doorGroup == nil {
			return nil, fmt.Errorf("no door proxy group configured")
		}

		// 从 doorGroup 获取成员
		return r.doorGroup.GetMember(showName)
	}

	// 都未找到，返回错误
	return nil, fmt.Errorf("proxy '%s' not found", name)
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
// showName: 可选参数，指定要获取的门代理成员名称
// - 如果不提供参数或为空字符串，返回当前选中的或延迟最低的成员
// - 如果提供成员名称，返回指定的成员
func (r *Registry) GetDoor(showName ...string) (proxy.Proxy, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.doorGroup == nil {
		return nil, fmt.Errorf("no door proxy group configured")
	}

	// 如果指定了成员名称
	if len(showName) > 0 && showName[0] != "" {
		return r.doorGroup.GetMember(showName[0])
	}

	// 否则返回当前选中的或最佳成员
	return r.doorGroup.GetCurrentOrBest()
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
			Name:       name,
			Type:       p.Proto().String(),
			Addr:       p.Addr(),
			IsDefault:  name == r.defaultName,
			IsNonelane: name == "nonelane",
		}
	}

	// 展开 door 代理组的成员
	if r.doorGroup != nil && r.doorGroup.Count() > 0 {
		members := r.doorGroup.ListMembers()
		currentMember := r.doorGroup.GetCurrentMemberName()

		for _, member := range members {
			// 使用 "door:成员名" 作为键，以区分不同的 door 成员
			memberKey := fmt.Sprintf("door:%s", member.ShowName)
			info[memberKey] = ProxyInfo{
				Name:        memberKey,
				Type:        member.Proxy.Proto().String(),
				Addr:        member.Proxy.Addr(),
				IsDefault:   memberKey == r.defaultName || (r.defaultName == "door" && member.ShowName == currentMember),
				IsDoorProxy: true,
				IsNonelane:  false,
				Latency:     member.Latency,
				ShowName:    member.ShowName,
			}
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
	Latency     int64  `json:"latency,omitempty"`     // 延迟（毫秒），仅用于 door 成员
	ShowName    string `json:"show_name,omitempty"`   // 显示名称，仅用于 door 成员
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
	r.doorGroup = nil
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

	// 根据配置设置默认代理
	if cfg.IsDefault {
		if err := r.SetDefault(name); err != nil {
			return fmt.Errorf("failed to set default proxy: %w", err)
		}
	}

	return nil
}

// RegisterDoorFromConfig 从配置注册 door 代理集合
func (r *Registry) RegisterDoorFromConfig(cfg *proxyConfig.ProxyConfig) error {
	if cfg == nil {
		return fmt.Errorf("config cannot be nil")
	}
	if cfg.Type != "door" {
		return fmt.Errorf("config type must be 'door', got '%s'", cfg.Type)
	}
	if len(cfg.Members) == 0 {
		return fmt.Errorf("door proxy must have at least one member")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	// 创建新的 door proxy group
	doorGroup := NewDoorProxyGroup()

	// 注册每个成员
	for _, member := range cfg.Members {
		// 创建成员的代理配置
		memberCfg := &proxyConfig.ProxyConfig{
			Type:        member.Type,
			VLESS:       member.VLESS,
			Shadowsocks: member.Shadowsocks,
		}

		// 验证成员配置
		if err := memberCfg.Validate(); err != nil {
			return fmt.Errorf("invalid config for member '%s': %w", member.ShowName, err)
		}

		// 创建代理实例
		p, err := proxyConfig.CreateProxyFromConfig(memberCfg)
		if err != nil {
			return fmt.Errorf("failed to create proxy for member '%s': %w", member.ShowName, err)
		}

		// 添加到 door group
		if err := doorGroup.AddMember(member.ShowName, p, member.Latency); err != nil {
			return fmt.Errorf("failed to add member '%s': %w", member.ShowName, err)
		}

		logger.Info(fmt.Sprintf("Door member '%s' registered (type: %s, addr: %s, latency: %dms)",
			member.ShowName, p.Proto().String(), p.Addr(), member.Latency))
	}

	// 设置 door group
	r.doorGroup = doorGroup
	logger.Info(fmt.Sprintf("Door proxy group registered with %d members", doorGroup.Count()))

	return nil
}

// SetDoorMember 设置当前 door 成员
func (r *Registry) SetDoorMember(showName string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.doorGroup == nil {
		return fmt.Errorf("no door proxy group configured")
	}

	return r.doorGroup.SetCurrentMember(showName)
}

// EnableDoorAutoSelect 启用 door 自动选择最佳成员
func (r *Registry) EnableDoorAutoSelect() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.doorGroup == nil {
		return fmt.Errorf("no door proxy group configured")
	}

	r.doorGroup.EnableAutoSelect()
	logger.Info("Door auto-select enabled")
	return nil
}

// ListDoorMembers 列出 door 所有成员信息
func (r *Registry) ListDoorMembers() ([]DoorProxyMemberInfo, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.doorGroup == nil {
		return nil, fmt.Errorf("no door proxy group configured")
	}

	return r.doorGroup.ListMembers(), nil
}

// UpdateDoorMemberLatency 更新 door 成员的延迟
func (r *Registry) UpdateDoorMemberLatency(showName string, latency int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.doorGroup == nil {
		return fmt.Errorf("no door proxy group configured")
	}

	return r.doorGroup.UpdateLatency(showName, latency)
}

// GetDoorCurrentMember 获取当前选中的 door 成员名称
func (r *Registry) GetDoorCurrentMember() string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.doorGroup == nil {
		return ""
	}

	return r.doorGroup.GetCurrentMemberName()
}

// IsDoorAutoSelect 返回 door 是否启用自动选择
func (r *Registry) IsDoorAutoSelect() bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.doorGroup == nil {
		return false
	}

	return r.doorGroup.IsAutoSelect()
}
