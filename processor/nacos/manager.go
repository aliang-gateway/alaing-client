package nacos

import (
	"fmt"
	"sync"
	"time"

	"nursor.org/nursorgate/common/logger"
	"nursor.org/nursorgate/common/model"
	"nursor.org/nursorgate/processor/routing"

	"github.com/nacos-group/nacos-sdk-go/clients/config_client"
	"github.com/nacos-group/nacos-sdk-go/vo"
)

const (
	RoutingRulesDataID = "routing-rules"
	DefaultGroup       = "DEFAULT_GROUP"
)

// T048: ConfigManager manages routing configuration with Nacos synchronization
type ConfigManager struct {
	mu         sync.RWMutex
	client     config_client.IConfigClient
	config     *model.RoutingRulesConfig
	autoUpdate bool
	stopCh     chan struct{}
	listening  bool
}

var (
	defaultManager *ConfigManager
	managerOnce    sync.Once
)

// T049: NewConfigManager creates a new ConfigManager instance
func NewConfigManager(client config_client.IConfigClient) *ConfigManager {
	return &ConfigManager{
		client:     client,
		autoUpdate: true, // Default to auto-update enabled
		stopCh:     make(chan struct{}),
		listening:  false,
	}
}

// GetManager returns the singleton ConfigManager instance
func GetManager() *ConfigManager {
	managerOnce.Do(func() {
		// Note: client should be set via SetClient before using
		defaultManager = &ConfigManager{
			autoUpdate: true,
			stopCh:     make(chan struct{}),
			listening:  false,
		}
	})
	return defaultManager
}

// SetClient sets the Nacos client for the manager
func (cm *ConfigManager) SetClient(client config_client.IConfigClient) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.client = client
}

// LoadConfig loads routing configuration from Nacos
func (cm *ConfigManager) LoadConfig() (*model.RoutingRulesConfig, error) {
	if cm.client == nil {
		return nil, fmt.Errorf("Nacos client not initialized")
	}

	configContent, err := cm.client.GetConfig(vo.ConfigParam{
		DataId: RoutingRulesDataID,
		Group:  DefaultGroup,
	})

	if err != nil {
		return nil, fmt.Errorf("failed to load config from Nacos: %w", err)
	}

	// If config is empty, return default
	if configContent == "" {
		logger.Warn("Routing config is empty in Nacos, using default")
		return model.NewRoutingRulesConfig(), nil
	}

	// Parse config
	config, err := model.NewRoutingRulesConfigFromJSON([]byte(configContent))
	if err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	cm.mu.Lock()
	cm.config = config
	cm.autoUpdate = config.Settings.AutoUpdate
	cm.mu.Unlock()

	logger.Info(fmt.Sprintf("Loaded routing config from Nacos (auto_update=%v)", cm.autoUpdate))
	return config, nil
}

// SaveConfig saves routing configuration to Nacos
// CRITICAL: This automatically sets auto_update=false (T050 requirement)
func (cm *ConfigManager) SaveConfig(config *model.RoutingRulesConfig) error {
	if cm.client == nil {
		return fmt.Errorf("Nacos client not initialized")
	}

	// Validate config
	if err := config.Validate(); err != nil {
		return fmt.Errorf("config validation failed: %w", err)
	}

	// CRITICAL: Set auto_update=false when config is modified via API
	config.Settings.AutoUpdate = false
	config.UpdatedAt = time.Now()

	// Serialize to JSON
	configJSON, err := config.ToJSON()
	if err != nil {
		return fmt.Errorf("failed to serialize config: %w", err)
	}

	// Publish to Nacos
	success, err := cm.client.PublishConfig(vo.ConfigParam{
		DataId:  RoutingRulesDataID,
		Group:   DefaultGroup,
		Content: string(configJSON),
	})

	if err != nil || !success {
		return fmt.Errorf("failed to publish config to Nacos: %w", err)
	}

	// Update local state
	cm.mu.Lock()
	cm.config = config
	cm.autoUpdate = false
	cm.mu.Unlock()

	logger.Info("Routing config saved to Nacos (auto_update set to false)")
	return nil
}

// T052: HandleConfigChange is the callback function for Nacos config changes (exported for testing)
// It checks auto_update flag before applying changes
func (cm *ConfigManager) HandleConfigChange(namespace, group, dataId, data string) {
	cm.handleConfigChange(namespace, group, dataId, data)
}

// handleConfigChange is the internal callback function for Nacos config changes
// It checks auto_update flag before applying changes
func (cm *ConfigManager) handleConfigChange(namespace, group, dataId, data string) {
	logger.Info(fmt.Sprintf("Nacos config change detected (dataId=%s, group=%s)", dataId, group))

	cm.mu.RLock()
	autoUpdate := cm.autoUpdate
	cm.mu.RUnlock()

	// Check auto_update flag
	if !autoUpdate {
		logger.Warn("Auto-update is DISABLED, ignoring Nacos config change")
		logger.Info("To re-enable auto-update, call PUT /api/config/routing/auto-update")
		return
	}

	logger.Info("Auto-update is ENABLED, applying Nacos config change")

	// Parse new config
	newConfig, err := model.NewRoutingRulesConfigFromJSON([]byte(data))
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to parse Nacos config change: %v", err))
		return
	}

	// Validate new config
	if err := newConfig.Validate(); err != nil {
		logger.Error(fmt.Sprintf("Invalid Nacos config change: %v", err))
		return
	}

	// Update local state
	cm.mu.Lock()
	cm.config = newConfig
	cm.autoUpdate = newConfig.Settings.AutoUpdate
	cm.mu.Unlock()

	// Apply changes to switch manager
	switchMgr := routing.GetSwitchManager()
	switchMgr.UpdateSwitches(newConfig)

	logger.Info(fmt.Sprintf("Nacos config applied successfully (auto_update=%v)", newConfig.Settings.AutoUpdate))
}

// T051: EnableAutoUpdate enables automatic synchronization from Nacos
func (cm *ConfigManager) EnableAutoUpdate() error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if cm.autoUpdate {
		logger.Info("Auto-update is already enabled")
		return nil
	}

	cm.autoUpdate = true

	// Update the config in Nacos with auto_update=true
	if cm.config != nil && cm.client != nil {
		cm.config.Settings.AutoUpdate = true
		cm.config.Settings.UpdatedAt = time.Now()

		configJSON, err := cm.config.ToJSON()
		if err != nil {
			return fmt.Errorf("failed to serialize config: %w", err)
		}

		success, err := cm.client.PublishConfig(vo.ConfigParam{
			DataId:  RoutingRulesDataID,
			Group:   DefaultGroup,
			Content: string(configJSON),
		})

		if err != nil || !success {
			cm.autoUpdate = false // Rollback on failure
			return fmt.Errorf("failed to update auto_update flag in Nacos: %w", err)
		}
	}

	logger.Info("Auto-update ENABLED - Nacos config changes will be applied automatically")
	return nil
}

// DisableAutoUpdate disables automatic synchronization
func (cm *ConfigManager) DisableAutoUpdate() {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if !cm.autoUpdate {
		logger.Info("Auto-update is already disabled")
		return
	}

	cm.autoUpdate = false
	logger.Info("Auto-update DISABLED - Nacos config changes will be ignored")
}

// T053: GetAutoUpdateStatus returns the current auto_update status
func (cm *ConfigManager) GetAutoUpdateStatus() bool {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return cm.autoUpdate
}

// GetCurrentConfig returns the current routing configuration
func (cm *ConfigManager) GetCurrentConfig() *model.RoutingRulesConfig {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return cm.config
}

// StartListening starts the Nacos config listener (US5)
func (cm *ConfigManager) StartListening() error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if cm.listening {
		return fmt.Errorf("listener already started")
	}

	if cm.client == nil {
		return fmt.Errorf("Nacos client not initialized")
	}

	err := cm.client.ListenConfig(vo.ConfigParam{
		DataId: RoutingRulesDataID,
		Group:  DefaultGroup,
		OnChange: func(namespace, group, dataId, data string) {
			cm.handleConfigChange(namespace, group, dataId, data)
		},
	})

	if err != nil {
		return fmt.Errorf("failed to start Nacos listener: %w", err)
	}

	cm.listening = true
	logger.Info("Nacos config listener started successfully")
	return nil
}

// StopListening stops the Nacos config listener (US5)
func (cm *ConfigManager) StopListening() error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if !cm.listening {
		return nil // Already stopped
	}

	if cm.client == nil {
		return fmt.Errorf("Nacos client not initialized")
	}

	err := cm.client.CancelListenConfig(vo.ConfigParam{
		DataId: RoutingRulesDataID,
		Group:  DefaultGroup,
	})

	if err != nil {
		return fmt.Errorf("failed to stop Nacos listener: %w", err)
	}

	cm.listening = false
	close(cm.stopCh)
	logger.Info("Nacos config listener stopped")
	return nil
}

// IsListening returns whether the listener is active
func (cm *ConfigManager) IsListening() bool {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return cm.listening
}
