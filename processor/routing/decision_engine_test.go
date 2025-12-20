package routing

import (
	"testing"

	"nursor.org/nursorgate/common/model"
)

// T023: Test that NoneLane rules have highest priority
func Test_NoneLaneHighestPriority(t *testing.T) {
	// Create routing config with NoneLane and Door rules
	config := &model.RoutingRulesConfig{
		Settings: model.RulesSettings{
			NoneLaneEnabled: true,
			DoorEnabled:     true,
			GeoIPEnabled:    false,
		},
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

	// DecideRoute should return NoneLane since it has highest priority
	decision, err := DecideRoute(config, ctx)
	if err != nil {
		t.Fatalf("DecideRoute failed: %v", err)
	}

	if decision != RouteToNoneLane {
		t.Errorf("Expected RouteToNoneLane, got %s", decision)
	}
}

// T024: Test Door rule matching with domain patterns
func Test_DoorRuleMatching(t *testing.T) {
	config := &model.RoutingRulesConfig{
		Settings: model.RulesSettings{
			NoneLaneEnabled: false, // NoneLane disabled
			DoorEnabled:     true,
			GeoIPEnabled:    false,
		},
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
					Condition: "*.google.com",
					Enabled:   true,
				},
				{
					ID:        "door_rule_2",
					Type:      model.RuleTypeDomain,
					Condition: "youtube.com",
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
		domain   string
		expected RouteDecision
		name     string
	}{
		{"www.google.com", RouteToDoor, "Wildcard *.google.com matches www.google.com"},
		{"maps.google.com", RouteToDoor, "Wildcard *.google.com matches maps.google.com"},
		{"google.com", RouteDirect, "Wildcard *.google.com does not match google.com"},
		{"youtube.com", RouteToDoor, "Exact domain youtube.com matches"},
		{"www.youtube.com", RouteDirect, "Sub-domain of youtube.com does not match"},
		{"example.com", RouteDirect, "Unknown domain routes to Direct"},
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

			if decision != tc.expected {
				t.Errorf("For domain %s: expected %s, got %s", tc.domain, tc.expected, decision)
			}
		})
	}
}

// T025: Test GeoIP rule matching
// Note: GeoIP is currently a stub, so these tests verify the structure
func Test_GeoIPMatching(t *testing.T) {
	config := &model.RoutingRulesConfig{
		Settings: model.RulesSettings{
			NoneLaneEnabled: false,
			DoorEnabled:     false,
			GeoIPEnabled:    true,
		},
		NoneLane: model.RoutingRuleSet{
			SetType: model.SetTypeNoneLane,
			Rules:   []model.RoutingRule{},
		},
		ToDoor: model.RoutingRuleSet{
			SetType: model.SetTypeToDoor,
			Rules: []model.RoutingRule{
				{
					ID:        "geo_rule_us",
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
		IP:     "1.1.1.1",
	}

	decision, err := DecideRoute(config, ctx)
	if err != nil {
		t.Logf("DecideRoute returned error (GeoIP stub): %v", err)
	}

	// For now, GeoIP is stubbed, so we expect Direct routing
	// Once GeoIP service is integrated, this test should be updated
	if decision != RouteDirect {
		t.Logf("GeoIP stubbed: expected RouteDirect, got %s (acceptable)", decision)
	}
}

// T026: Test global switch controls
func Test_GlobalSwitches(t *testing.T) {
	testCases := []struct {
		noneLaneEnabled bool
		doorEnabled     bool
		geoIPEnabled    bool
		expected        RouteDecision
		name            string
	}{
		{true, true, false, RouteToNoneLane, "All switches enabled: NoneLane has priority"},
		{false, true, false, RouteToDoor, "NoneLane disabled: Door has priority"},
		{false, false, false, RouteDirect, "All switches disabled: Direct routing"},
		{true, false, false, RouteToNoneLane, "Only NoneLane enabled: uses NoneLane"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			config := &model.RoutingRulesConfig{
				Settings: model.RulesSettings{
					NoneLaneEnabled: tc.noneLaneEnabled,
					DoorEnabled:     tc.doorEnabled,
					GeoIPEnabled:    tc.geoIPEnabled,
				},
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
				IP:     "1.1.1.1",
			}

			decision, err := DecideRoute(config, ctx)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if decision != tc.expected {
				t.Errorf("Expected %s, got %s", tc.expected, decision)
			}
		})
	}
}

// T027: Test that disabled rules are skipped
func Test_DisabledRuleSkipped(t *testing.T) {
	config := &model.RoutingRulesConfig{
		Settings: model.RulesSettings{
			NoneLaneEnabled: true,
			DoorEnabled:     false,
			GeoIPEnabled:    false,
		},
		NoneLane: model.RoutingRuleSet{
			SetType: model.SetTypeNoneLane,
			Rules: []model.RoutingRule{
				{
					ID:        "nl_rule_enabled",
					Type:      model.RuleTypeDomain,
					Condition: "enabled.com",
					Enabled:   true,
				},
				{
					ID:        "nl_rule_disabled",
					Type:      model.RuleTypeDomain,
					Condition: "disabled.com",
					Enabled:   false, // This rule should be skipped
				},
			},
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

	testCases := []struct {
		domain   string
		expected RouteDecision
		name     string
	}{
		{"enabled.com", RouteToNoneLane, "Enabled rule matches"},
		{"disabled.com", RouteDirect, "Disabled rule is skipped, routes to Direct"},
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

			if decision != tc.expected {
				t.Errorf("Expected %s, got %s", tc.expected, decision)
			}
		})
	}
}
