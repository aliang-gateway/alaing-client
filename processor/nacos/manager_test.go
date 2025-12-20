package nacos

import (
	"testing"
	"time"

	"nursor.org/nursorgate/common/model"
)

// T044: Test auto-update enabled - Nacos changes are automatically applied
func Test_AutoUpdateEnabled(t *testing.T) {
	// Create manager with mock client
	mockClient := NewMockConfigClient()
	manager := NewConfigManager(mockClient)

	// Create initial config with auto_update=true
	initialConfig := model.NewRoutingRulesConfig()
	initialConfig.Settings.AutoUpdate = true
	initialConfig.NoneLane.Rules = []model.RoutingRule{
		{
			ID:        "test_rule_1",
			Type:      model.RuleTypeDomain,
			Condition: "example.com",
			Enabled:   true,
		},
	}

	// Set initial config
	configJSON, _ := initialConfig.ToJSON()
	mockClient.Configs[RoutingRulesDataID] = string(configJSON)

	// Load config
	loadedConfig, err := manager.LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	// Verify auto_update is enabled
	if !manager.GetAutoUpdateStatus() {
		t.Error("Auto-update should be enabled after loading config with AutoUpdate=true")
	}

	// Verify config was loaded
	if loadedConfig == nil {
		t.Fatal("Loaded config should not be nil")
	}

	if len(loadedConfig.NoneLane.Rules) != 1 {
		t.Errorf("Expected 1 rule, got %d", len(loadedConfig.NoneLane.Rules))
	}
}

// T045: Test auto-update disabled - Nacos changes are ignored
func Test_AutoUpdateDisabled(t *testing.T) {
	mockClient := NewMockConfigClient()
	manager := NewConfigManager(mockClient)

	// Create initial config
	initialConfig := model.NewRoutingRulesConfig()
	initialConfig.Settings.AutoUpdate = true

	configJSON, _ := initialConfig.ToJSON()
	mockClient.Configs[RoutingRulesDataID] = string(configJSON)

	// Load initial config
	_, err := manager.LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	// Disable auto-update
	manager.DisableAutoUpdate()

	// Verify auto-update is disabled
	if manager.GetAutoUpdateStatus() {
		t.Error("Auto-update should be disabled after calling DisableAutoUpdate()")
	}

	// Simulate Nacos config change
	newConfig := model.NewRoutingRulesConfig()
	newConfig.NoneLane.Rules = []model.RoutingRule{
		{
			ID:        "new_rule",
			Type:      model.RuleTypeDomain,
			Condition: "newdomain.com",
			Enabled:   true,
		},
	}
	newConfigJSON, _ := newConfig.ToJSON()

	// Call handleConfigChange directly (simulating Nacos callback)
	manager.handleConfigChange("", DefaultGroup, RoutingRulesDataID, string(newConfigJSON))

	// Verify config was NOT updated (because auto_update=false)
	currentConfig := manager.GetCurrentConfig()
	if currentConfig == nil {
		t.Fatal("Current config should not be nil")
	}

	// The config should still be the initial one, not the new one
	if len(currentConfig.NoneLane.Rules) != 0 {
		t.Error("Config should not be updated when auto_update is disabled")
	}
}

// T046: Test API modification detection - POST /config/routing automatically sets auto_update=false
func Test_APIModificationDetection(t *testing.T) {
	mockClient := NewMockConfigClient()
	manager := NewConfigManager(mockClient)

	// Create config
	config := model.NewRoutingRulesConfig()
	config.Settings.AutoUpdate = true // Initially true

	// Save config via API (simulating POST /config/routing)
	err := manager.SaveConfig(config)
	if err != nil {
		t.Fatalf("SaveConfig failed: %v", err)
	}

	// Verify auto_update was automatically set to false
	if manager.GetAutoUpdateStatus() {
		t.Error("Auto-update should be automatically disabled after SaveConfig()")
	}

	// Verify the saved config has AutoUpdate=false
	savedConfig := manager.GetCurrentConfig()
	if savedConfig == nil {
		t.Fatal("Saved config should not be nil")
	}

	if savedConfig.Settings.AutoUpdate {
		t.Error("Saved config should have AutoUpdate=false")
	}
}

// T047: Test manual resume sync - PUT /config/routing/auto-update restores synchronization
func Test_ManualResumeSync(t *testing.T) {
	mockClient := NewMockConfigClient()
	manager := NewConfigManager(mockClient)

	// Create and load initial config
	initialConfig := model.NewRoutingRulesConfig()
	initialConfig.Settings.AutoUpdate = true

	configJSON, _ := initialConfig.ToJSON()
	mockClient.Configs[RoutingRulesDataID] = string(configJSON)

	_, err := manager.LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	// Simulate API modification (auto_update becomes false)
	modifiedConfig := model.NewRoutingRulesConfig()
	manager.SaveConfig(modifiedConfig)

	// Verify auto-update is disabled
	if manager.GetAutoUpdateStatus() {
		t.Error("Auto-update should be disabled after SaveConfig")
	}

	// Manually re-enable auto-update (simulating PUT /config/routing/auto-update)
	err = manager.EnableAutoUpdate()
	if err != nil {
		t.Fatalf("EnableAutoUpdate failed: %v", err)
	}

	// Verify auto-update is now enabled
	if !manager.GetAutoUpdateStatus() {
		t.Error("Auto-update should be enabled after EnableAutoUpdate()")
	}

	// Simulate Nacos config change after re-enabling
	newConfig := model.NewRoutingRulesConfig()
	newConfig.NoneLane.Rules = []model.RoutingRule{
		{
			ID:        "resumed_rule",
			Type:      model.RuleTypeDomain,
			Condition: "resumed.com",
			Enabled:   true,
		},
	}
	newConfigJSON, _ := newConfig.ToJSON()

	// Handle config change
	manager.handleConfigChange("", DefaultGroup, RoutingRulesDataID, string(newConfigJSON))

	// Verify config WAS updated (because auto_update=true now)
	currentConfig := manager.GetCurrentConfig()
	if currentConfig == nil {
		t.Fatal("Current config should not be nil")
	}

	if len(currentConfig.NoneLane.Rules) != 1 {
		t.Error("Config should be updated after auto-update is re-enabled")
	}

	if len(currentConfig.NoneLane.Rules) > 0 && currentConfig.NoneLane.Rules[0].ID != "resumed_rule" {
		t.Errorf("Expected rule ID 'resumed_rule', got '%s'", currentConfig.NoneLane.Rules[0].ID)
	}
}

// Additional test: Test state transitions
func Test_AutoUpdateStateTransitions(t *testing.T) {
	mockClient := NewMockConfigClient()
	manager := NewConfigManager(mockClient)

	// Initial state: auto_update should be true (default)
	if !manager.GetAutoUpdateStatus() {
		t.Error("Initial auto_update should be true")
	}

	// Disable
	manager.DisableAutoUpdate()
	if manager.GetAutoUpdateStatus() {
		t.Error("Auto-update should be false after DisableAutoUpdate()")
	}

	// Enable
	manager.EnableAutoUpdate()
	if !manager.GetAutoUpdateStatus() {
		t.Error("Auto-update should be true after EnableAutoUpdate()")
	}

	// Multiple disables should be idempotent
	manager.DisableAutoUpdate()
	manager.DisableAutoUpdate()
	if manager.GetAutoUpdateStatus() {
		t.Error("Auto-update should still be false")
	}
}

// Test GetCurrentConfig when no config is loaded
func Test_GetCurrentConfigWhenEmpty(t *testing.T) {
	mockClient := NewMockConfigClient()
	manager := NewConfigManager(mockClient)

	config := manager.GetCurrentConfig()
	if config != nil {
		t.Error("GetCurrentConfig should return nil when no config is loaded")
	}
}

// Test LoadConfig with empty Nacos response
func Test_LoadConfigEmptyNacos(t *testing.T) {
	mockClient := NewMockConfigClient()
	manager := NewConfigManager(mockClient)

	// mockClient has no configs, so GetConfig will return empty string
	config, err := manager.LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig should not fail with empty Nacos: %v", err)
	}

	if config == nil {
		t.Error("LoadConfig should return default config when Nacos is empty")
	}

	// Verify it's a valid default config
	if config.Settings.NoneLaneEnabled != true || config.Settings.DoorEnabled != true {
		t.Error("Default config should have NoneLane and Door enabled")
	}
}

// Test SaveConfig updates timestamp
func Test_SaveConfigUpdatesTimestamp(t *testing.T) {
	mockClient := NewMockConfigClient()
	manager := NewConfigManager(mockClient)

	config := model.NewRoutingRulesConfig()
	oldTime := config.UpdatedAt

	// Wait a bit to ensure timestamp changes
	time.Sleep(10 * time.Millisecond)

	err := manager.SaveConfig(config)
	if err != nil {
		t.Fatalf("SaveConfig failed: %v", err)
	}

	savedConfig := manager.GetCurrentConfig()
	if savedConfig == nil {
		t.Fatal("Saved config should not be nil")
	}

	if !savedConfig.UpdatedAt.After(oldTime) {
		t.Error("SaveConfig should update the timestamp")
	}
}
