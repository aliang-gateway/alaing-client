package cmd

import (
	"testing"
	"time"

	"nursor.org/nursorgate/common/model"
	"nursor.org/nursorgate/processor/nacos"
)

// T055: Test that Nacos listener is initialized within 5 seconds during startup
func TestStartupInitializesNacosListener(t *testing.T) {
	// Create a mock Nacos client for testing
	mockClient := nacos.NewMockConfigClient()

	// Create ConfigManager
	manager := nacos.NewConfigManager(mockClient)

	// Start listening
	startTime := time.Now()
	err := manager.StartListening()
	if err != nil {
		t.Fatalf("Failed to start Nacos listener: %v", err)
	}
	elapsed := time.Since(startTime)

	// Verify listener started within 5 seconds
	if elapsed > 5*time.Second {
		t.Errorf("Nacos listener initialization took too long: %v (expected < 5s)", elapsed)
	}

	// Verify listener is active
	if !manager.IsListening() {
		t.Error("Nacos listener should be active after StartListening()")
	}

	// Cleanup
	manager.StopListening()
}

// T056: Test that Nacos config changes trigger notifications
func TestNacosConfigChangeNotification(t *testing.T) {
	// Create a mock Nacos client for testing
	mockClient := nacos.NewMockConfigClient()

	// Create ConfigManager with auto_update enabled
	manager := nacos.NewConfigManager(mockClient)

	// Create initial config with auto_update=true
	initialConfig := model.NewRoutingRulesConfig()
	initialConfig.Settings.AutoUpdate = true

	// Start listening first (before setting config)
	err := manager.StartListening()
	if err != nil {
		t.Fatalf("Failed to start Nacos listener: %v", err)
	}

	// Load config to set up auto_update
	configJSON, _ := initialConfig.ToJSON()
	mockClient.Configs[nacos.RoutingRulesDataID] = string(configJSON)

	// Load config
	_, err = manager.LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	if !manager.GetAutoUpdateStatus() {
		t.Error("Auto-update should be enabled after loading config with AutoUpdate=true")
	}

	// Simulate a config change notification
	newConfig := model.NewRoutingRulesConfig()
	newConfig.Settings.AutoUpdate = true
	newConfig.NoneLane.Rules = []model.RoutingRule{
		{
			ID:        "test_rule_from_notification",
			Type:      model.RuleTypeDomain,
			Condition: "notified-domain.com",
			Enabled:   true,
		},
	}

	newConfigJSON, _ := newConfig.ToJSON()

	// Trigger the config change callback (simulating Nacos notification)
	manager.HandleConfigChange("", nacos.DefaultGroup, nacos.RoutingRulesDataID, string(newConfigJSON))

	// Verify the config was updated
	currentConfig := manager.GetCurrentConfig()
	if currentConfig == nil {
		t.Fatal("Current config should not be nil after notification")
	}

	if len(currentConfig.NoneLane.Rules) != 1 {
		t.Errorf("Expected 1 rule from notification, got %d", len(currentConfig.NoneLane.Rules))
	}

	if len(currentConfig.NoneLane.Rules) > 0 && currentConfig.NoneLane.Rules[0].ID != "test_rule_from_notification" {
		t.Errorf("Expected rule 'test_rule_from_notification', got '%s'", currentConfig.NoneLane.Rules[0].ID)
	}

	// Cleanup
	manager.StopListening()
}
