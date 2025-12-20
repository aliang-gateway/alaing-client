package routing

import (
	"testing"

	"nursor.org/nursorgate/common/model"
)

// T035: Test NoneLane disabled - traffic should fallback to Door or Direct
func Test_NoneLaneDisabled(t *testing.T) {
	// Create a SwitchManager and disable NoneLane
	sm := GetSwitchManager()
	sm.SetNoneLaneEnabled(false)
	sm.SetDoorEnabled(true)
	sm.SetGeoIPEnabled(false)

	config := &model.RoutingRulesConfig{
		Settings: sm.GetStatus(),
		NoneLane: model.RoutingRuleSet{
			SetType: model.SetTypeNoneLane,
			Rules: []model.RoutingRule{
				{
					ID:        "nl_rule_1",
					Type:      model.RuleTypeDomain,
					Condition: "example.com",
					Enabled:   true,
				},
			},
		},
		ToDoor: model.RoutingRuleSet{
			SetType: model.SetTypeToDoor,
			Rules: []model.RoutingRule{
				{
					ID:        "door_rule_1",
					Type:      model.RuleTypeDomain,
					Condition: "example.com",
					Enabled:   true,
				},
			},
		},
		BlackList: model.RoutingRuleSet{
			SetType: model.SetTypeBlacklist,
			Rules:   []model.RoutingRule{},
		},
	}

	ctx := &MatchContext{
		Domain: "example.com",
		IP:     "1.2.3.4",
	}

	decision, err := DecideRoute(config, ctx)
	if err != nil {
		t.Fatalf("DecideRoute failed: %v", err)
	}

	// With NoneLane disabled, should fallback to Door
	if decision != RouteToDoor {
		t.Errorf("Expected RouteToDoor (NoneLane disabled), got %s", decision)
	}

	// Verify switch status
	if sm.IsNoneLaneEnabled() {
		t.Error("NoneLane switch should be disabled")
	}
}

// T036: Test Door disabled - traffic should fallback to GeoIP or Direct
func Test_DoorDisabled(t *testing.T) {
	sm := GetSwitchManager()
	sm.SetNoneLaneEnabled(false)
	sm.SetDoorEnabled(false) // Disable Door
	sm.SetGeoIPEnabled(false)

	config := &model.RoutingRulesConfig{
		Settings: sm.GetStatus(),
		NoneLane: model.RoutingRuleSet{
			SetType: model.SetTypeNoneLane,
			Rules:   []model.RoutingRule{},
		},
		ToDoor: model.RoutingRuleSet{
			SetType: model.SetTypeToDoor,
			Rules: []model.RoutingRule{
				{
					ID:        "door_rule_1",
					Type:      model.RuleTypeDomain,
					Condition: "example.com",
					Enabled:   true,
				},
			},
		},
		BlackList: model.RoutingRuleSet{
			SetType: model.SetTypeBlacklist,
			Rules:   []model.RoutingRule{},
		},
	}

	ctx := &MatchContext{
		Domain: "example.com",
		IP:     "1.2.3.4",
	}

	decision, err := DecideRoute(config, ctx)
	if err != nil {
		t.Fatalf("DecideRoute failed: %v", err)
	}

	// With Door disabled, should fallback to Direct
	if decision != RouteDirect {
		t.Errorf("Expected RouteDirect (Door disabled), got %s", decision)
	}

	// Verify switch status
	if sm.IsDoorEnabled() {
		t.Error("Door switch should be disabled")
	}
}

// T037: Test GeoIP disabled - GeoIP rules should not be evaluated
func Test_GeoIPDisabled(t *testing.T) {
	sm := GetSwitchManager()
	sm.SetNoneLaneEnabled(false)
	sm.SetDoorEnabled(false)
	sm.SetGeoIPEnabled(false) // Disable GeoIP

	config := &model.RoutingRulesConfig{
		Settings: sm.GetStatus(),
		NoneLane: model.RoutingRuleSet{
			SetType: model.SetTypeNoneLane,
			Rules:   []model.RoutingRule{},
		},
		ToDoor: model.RoutingRuleSet{
			SetType: model.SetTypeToDoor,
			Rules: []model.RoutingRule{
				{
					ID:        "geo_rule_1",
					Type:      model.RuleTypeGeoIP,
					Condition: "US",
					Enabled:   true,
				},
			},
		},
		BlackList: model.RoutingRuleSet{
			SetType: model.SetTypeBlacklist,
			Rules:   []model.RoutingRule{},
		},
	}

	ctx := &MatchContext{
		Domain: "example.com",
		IP:     "1.1.1.1", // US IP
	}

	decision, err := DecideRoute(config, ctx)
	if err != nil {
		t.Fatalf("DecideRoute failed: %v", err)
	}

	// With GeoIP disabled, should fallback to Direct
	if decision != RouteDirect {
		t.Errorf("Expected RouteDirect (GeoIP disabled), got %s", decision)
	}

	// Verify switch status
	if sm.IsGeoIPEnabled() {
		t.Error("GeoIP switch should be disabled")
	}
}

// T038: Test all switches disabled - should always route Direct
func Test_AllSwitchesDisabled(t *testing.T) {
	sm := GetSwitchManager()
	sm.DisableAll() // Disable all switches

	config := &model.RoutingRulesConfig{
		Settings: sm.GetStatus(),
		NoneLane: model.RoutingRuleSet{
			SetType: model.SetTypeNoneLane,
			Rules: []model.RoutingRule{
				{
					ID:        "nl_rule_1",
					Type:      model.RuleTypeDomain,
					Condition: "example.com",
					Enabled:   true,
				},
			},
		},
		ToDoor: model.RoutingRuleSet{
			SetType: model.SetTypeToDoor,
			Rules: []model.RoutingRule{
				{
					ID:        "door_rule_1",
					Type:      model.RuleTypeDomain,
					Condition: "example.com",
					Enabled:   true,
				},
			},
		},
		BlackList: model.RoutingRuleSet{
			SetType: model.SetTypeBlacklist,
			Rules:   []model.RoutingRule{},
		},
	}

	testCases := []struct {
		domain string
		name   string
	}{
		{"example.com", "Domain with NoneLane and Door rules"},
		{"google.com", "Domain without rules"},
		{"youtube.com", "Different domain"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := &MatchContext{
				Domain: tc.domain,
				IP:     "1.2.3.4",
			}

			decision, err := DecideRoute(config, ctx)
			if err != nil {
				t.Fatalf("DecideRoute failed: %v", err)
			}

			// With all switches disabled, should always route Direct
			if decision != RouteDirect {
				t.Errorf("Expected RouteDirect (all switches disabled), got %s", decision)
			}
		})
	}

	// Verify all switches are disabled
	if sm.IsNoneLaneEnabled() {
		t.Error("NoneLane switch should be disabled")
	}
	if sm.IsDoorEnabled() {
		t.Error("Door switch should be disabled")
	}
	if sm.IsGeoIPEnabled() {
		t.Error("GeoIP switch should be disabled")
	}
}

// Additional test: Test switch state changes
func Test_SwitchStateChanges(t *testing.T) {
	sm := GetSwitchManager()

	// Test enabling NoneLane
	sm.SetNoneLaneEnabled(true)
	if !sm.IsNoneLaneEnabled() {
		t.Error("NoneLane should be enabled after SetNoneLaneEnabled(true)")
	}

	// Test disabling NoneLane
	sm.SetNoneLaneEnabled(false)
	if sm.IsNoneLaneEnabled() {
		t.Error("NoneLane should be disabled after SetNoneLaneEnabled(false)")
	}

	// Test EnableAll
	sm.EnableAll()
	status := sm.GetStatus()
	if !status.NoneLaneEnabled || !status.DoorEnabled || !status.GeoIPEnabled {
		t.Error("All switches should be enabled after EnableAll()")
	}

	// Test DisableAll
	sm.DisableAll()
	status = sm.GetStatus()
	if status.NoneLaneEnabled || status.DoorEnabled || status.GeoIPEnabled {
		t.Error("All switches should be disabled after DisableAll()")
	}

	// Test ResetToDefaults
	sm.ResetToDefaults()
	status = sm.GetStatus()
	if !status.NoneLaneEnabled || !status.DoorEnabled || status.GeoIPEnabled {
		t.Error("Switches should be reset to defaults (NoneLane=ON, Door=ON, GeoIP=OFF)")
	}
}

// Test UpdateSwitches from config
func Test_UpdateSwitchesFromConfig(t *testing.T) {
	sm := GetSwitchManager()

	config := &model.RoutingRulesConfig{
		Settings: model.RulesSettings{
			NoneLaneEnabled: false,
			DoorEnabled:     true,
			GeoIPEnabled:    true,
		},
		NoneLane: model.RoutingRuleSet{
			SetType: model.SetTypeNoneLane,
			Rules:   []model.RoutingRule{},
		},
		ToDoor: model.RoutingRuleSet{
			SetType: model.SetTypeToDoor,
			Rules:   []model.RoutingRule{},
		},
		BlackList: model.RoutingRuleSet{
			SetType: model.SetTypeBlacklist,
			Rules:   []model.RoutingRule{},
		},
	}

	sm.UpdateSwitches(config)

	status := sm.GetStatus()
	if status.NoneLaneEnabled != false {
		t.Error("NoneLane should be disabled after UpdateSwitches")
	}
	if status.DoorEnabled != true {
		t.Error("Door should be enabled after UpdateSwitches")
	}
	if status.GeoIPEnabled != true {
		t.Error("GeoIP should be enabled after UpdateSwitches")
	}
}
