package nacos

import (
	"fmt"
	"time"

	"github.com/nacos-group/nacos-sdk-go/vo"
	"nursor.org/nursorgate/common/logger"
	"nursor.org/nursorgate/common/model"
)

// T076: SyncStatus contains synchronization status information
type SyncStatus struct {
	IsSynced      bool                   `json:"is_synced"`
	LocalVersion  int                    `json:"local_version"`
	RemoteVersion int                    `json:"remote_version"`
	LastSyncTime  *time.Time             `json:"last_sync_time,omitempty"`
	TimeDiff      int64                  `json:"time_diff_ms,omitempty"`
	IsAutoUpdate  bool                   `json:"is_auto_update"`
	Differences   []string               `json:"differences,omitempty"`
	Config        map[string]interface{} `json:"config,omitempty"`
}

// T076: GetSyncStatus returns the current sync status between local and Nacos config
func GetSyncStatus(manager *ConfigManager) *SyncStatus {
	status := &SyncStatus{
		Config: make(map[string]interface{}),
	}

	if manager == nil {
		status.IsSynced = false
		status.Config["error"] = "ConfigManager not initialized"
		return status
	}

	// Get local config
	localConfig := manager.GetCurrentConfig()
	if localConfig == nil {
		status.IsSynced = false
		status.Config["error"] = "No local config loaded"
		return status
	}

	status.LocalVersion = localConfig.Version
	status.IsAutoUpdate = manager.GetAutoUpdateStatus()
	status.LastSyncTime = &localConfig.UpdatedAt

	// Try to get remote config from Nacos
	if manager.client == nil {
		status.IsSynced = false
		status.Config["error"] = "Nacos client not initialized"
		return status
	}

	configContent, err := manager.client.GetConfig(vo.ConfigParam{
		DataId: RoutingRulesDataID,
		Group:  DefaultGroup,
	})

	if err != nil {
		status.IsSynced = false
		status.Config["error"] = fmt.Sprintf("Failed to get remote config: %v", err)
		return status
	}

	// Parse remote config
	remoteConfig, err := model.NewRoutingRulesConfigFromJSON([]byte(configContent))
	if err != nil {
		status.IsSynced = false
		status.Config["error"] = fmt.Sprintf("Failed to parse remote config: %v", err)
		return status
	}

	status.RemoteVersion = remoteConfig.Version

	// Compare versions
	if localConfig.Version == remoteConfig.Version {
		status.IsSynced = true
		status.TimeDiff = 0
	} else {
		status.IsSynced = false
		status.TimeDiff = remoteConfig.UpdatedAt.Sub(localConfig.UpdatedAt).Milliseconds()
		status.Differences = compareConfigs(localConfig, remoteConfig)
	}

	return status
}

// T077: compareConfigs returns a list of differences between two configs
func compareConfigs(local, remote *model.RoutingRulesConfig) []string {
	var diffs []string

	// Compare basic settings
	if local.Settings.NoneLaneEnabled != remote.Settings.NoneLaneEnabled {
		diffs = append(diffs, "NoneLaneEnabled differs")
	}
	if local.Settings.DoorEnabled != remote.Settings.DoorEnabled {
		diffs = append(diffs, "DoorEnabled differs")
	}
	if local.Settings.GeoIPEnabled != remote.Settings.GeoIPEnabled {
		diffs = append(diffs, "GeoIPEnabled differs")
	}
	if local.Settings.AutoUpdate != remote.Settings.AutoUpdate {
		diffs = append(diffs, "AutoUpdate differs")
	}

	// Compare rule counts
	if len(local.NoneLane.Rules) != len(remote.NoneLane.Rules) {
		diffs = append(diffs, fmt.Sprintf("NoneLane rule count differs: local=%d, remote=%d",
			len(local.NoneLane.Rules), len(remote.NoneLane.Rules)))
	}
	if len(local.ToDoor.Rules) != len(remote.ToDoor.Rules) {
		diffs = append(diffs, fmt.Sprintf("ToDoor rule count differs: local=%d, remote=%d",
			len(local.ToDoor.Rules), len(remote.ToDoor.Rules)))
	}
	if len(local.BlackList.Rules) != len(remote.BlackList.Rules) {
		diffs = append(diffs, fmt.Sprintf("BlackList rule count differs: local=%d, remote=%d",
			len(local.BlackList.Rules), len(remote.BlackList.Rules)))
	}

	return diffs
}

// T078: ManualSync forcefully syncs local config with Nacos
func ManualSync(manager *ConfigManager) error {
	if manager == nil {
		return fmt.Errorf("ConfigManager not initialized")
	}

	if manager.client == nil {
		return fmt.Errorf("Nacos client not initialized")
	}

	logger.Info("Starting manual sync with Nacos...")

	// Load config from Nacos
	configContent, err := manager.client.GetConfig(vo.ConfigParam{
		DataId: RoutingRulesDataID,
		Group:  DefaultGroup,
	})

	if err != nil {
		return fmt.Errorf("failed to get config from Nacos: %w", err)
	}

	// Parse and validate
	newConfig, err := model.NewRoutingRulesConfigFromJSON([]byte(configContent))
	if err != nil {
		return fmt.Errorf("failed to parse remote config: %w", err)
	}

	if err := newConfig.Validate(); err != nil {
		return fmt.Errorf("remote config validation failed: %w", err)
	}

	// Update local state
	manager.mu.Lock()
	manager.config = newConfig
	manager.autoUpdate = newConfig.Settings.AutoUpdate
	manager.mu.Unlock()

	logger.Info("Manual sync completed successfully")
	return nil
}
