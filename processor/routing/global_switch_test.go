package routing

import (
	"testing"

	"aliang.one/nursorgate/common/model"
)

// T035: Test NoneLane disabled - traffic should fallback to SOCKS or Direct
func Test_NoneLaneDisabled(t *testing.T) {
	sm := GetSwitchManager()
	sm.SetAliangEnabled(false)
	sm.SetSocksEnabled(true)
	sm.SetGeoIPEnabled(false)

	config := &model.RoutingRulesConfig{
		Settings: sm.GetStatus(),
		Aliang:   model.RoutingRuleSet{SetType: model.SetTypeAliang, Rules: []model.RoutingRule{{ID: "nl_rule_1", Type: model.RuleTypeDomain, Condition: "example.com", Enabled: true}}},
		ToSocks:  model.RoutingRuleSet{SetType: model.SetTypeToSocks, Rules: []model.RoutingRule{{ID: "socks_rule_1", Type: model.RuleTypeDomain, Condition: "example.com", Enabled: true}}},
		Direct:   model.RoutingRuleSet{SetType: model.SetTypeDirect, Rules: []model.RoutingRule{}},
	}

	decision, err := DecideRoute(config, &MatchContext{Domain: "example.com", IP: "1.2.3.4"})
	if err != nil {
		t.Fatalf("DecideRoute failed: %v", err)
	}
	if decision != RouteToSocks {
		t.Errorf("Expected RouteToSocks (NoneLane disabled), got %s", decision)
	}
	if sm.IsAliangEnabled() {
		t.Error("NoneLane switch should be disabled")
	}
}

// T036: Test SOCKS disabled - traffic should fallback to GeoIP or Direct
func Test_SocksDisabled(t *testing.T) {
	sm := GetSwitchManager()
	sm.SetAliangEnabled(false)
	sm.SetSocksEnabled(false)
	sm.SetGeoIPEnabled(false)

	config := &model.RoutingRulesConfig{
		Settings: sm.GetStatus(),
		Aliang:   model.RoutingRuleSet{SetType: model.SetTypeAliang, Rules: []model.RoutingRule{}},
		ToSocks:  model.RoutingRuleSet{SetType: model.SetTypeToSocks, Rules: []model.RoutingRule{{ID: "socks_rule_1", Type: model.RuleTypeDomain, Condition: "example.com", Enabled: true}}},
		Direct:   model.RoutingRuleSet{SetType: model.SetTypeDirect, Rules: []model.RoutingRule{}},
	}

	decision, err := DecideRoute(config, &MatchContext{Domain: "example.com", IP: "1.2.3.4"})
	if err != nil {
		t.Fatalf("DecideRoute failed: %v", err)
	}
	if decision != RouteDirect {
		t.Errorf("Expected RouteDirect (SOCKS disabled), got %s", decision)
	}
	if sm.IsSocksEnabled() {
		t.Error("SOCKS switch should be disabled")
	}
}

// T037: Test GeoIP disabled - GeoIP rules should not be evaluated
func Test_GeoIPDisabled(t *testing.T) {
	sm := GetSwitchManager()
	sm.SetAliangEnabled(false)
	sm.SetSocksEnabled(false)
	sm.SetGeoIPEnabled(false)

	config := &model.RoutingRulesConfig{
		Settings: sm.GetStatus(),
		Aliang:   model.RoutingRuleSet{SetType: model.SetTypeAliang, Rules: []model.RoutingRule{}},
		ToSocks:  model.RoutingRuleSet{SetType: model.SetTypeToSocks, Rules: []model.RoutingRule{{ID: "geo_rule_1", Type: model.RuleTypeGeoIP, Condition: "US", Enabled: true}}},
		Direct:   model.RoutingRuleSet{SetType: model.SetTypeDirect, Rules: []model.RoutingRule{}},
	}

	decision, err := DecideRoute(config, &MatchContext{Domain: "example.com", IP: "1.1.1.1"})
	if err != nil {
		t.Fatalf("DecideRoute failed: %v", err)
	}
	if decision != RouteDirect {
		t.Errorf("Expected RouteDirect (GeoIP disabled), got %s", decision)
	}
	if sm.IsGeoIPEnabled() {
		t.Error("GeoIP switch should be disabled")
	}
}

// T038: Test all switches disabled - should always route Direct
func Test_AllSwitchesDisabled(t *testing.T) {
	sm := GetSwitchManager()
	sm.DisableAll()

	config := &model.RoutingRulesConfig{
		Settings: sm.GetStatus(),
		Aliang:   model.RoutingRuleSet{SetType: model.SetTypeAliang, Rules: []model.RoutingRule{{ID: "nl_rule_1", Type: model.RuleTypeDomain, Condition: "example.com", Enabled: true}}},
		ToSocks:  model.RoutingRuleSet{SetType: model.SetTypeToSocks, Rules: []model.RoutingRule{{ID: "socks_rule_1", Type: model.RuleTypeDomain, Condition: "example.com", Enabled: true}}},
		Direct:   model.RoutingRuleSet{SetType: model.SetTypeDirect, Rules: []model.RoutingRule{}},
	}

	for _, domain := range []string{"example.com", "google.com", "youtube.com"} {
		decision, err := DecideRoute(config, &MatchContext{Domain: domain, IP: "1.2.3.4"})
		if err != nil {
			t.Fatalf("DecideRoute failed: %v", err)
		}
		if decision != RouteDirect {
			t.Errorf("Expected RouteDirect (all switches disabled), got %s", decision)
		}
	}

	if sm.IsAliangEnabled() || sm.IsSocksEnabled() || sm.IsGeoIPEnabled() {
		t.Error("All switches should be disabled")
	}
}

func Test_SwitchStateChanges(t *testing.T) {
	sm := GetSwitchManager()

	sm.SetAliangEnabled(true)
	if !sm.IsAliangEnabled() {
		t.Error("NoneLane should be enabled")
	}
	sm.SetAliangEnabled(false)
	if sm.IsAliangEnabled() {
		t.Error("NoneLane should be disabled")
	}

	sm.EnableAll()
	status := sm.GetStatus()
	if !status.AliangEnabled || !status.SocksEnabled || !status.GeoIPEnabled {
		t.Error("All switches should be enabled after EnableAll()")
	}

	sm.DisableAll()
	status = sm.GetStatus()
	if status.AliangEnabled || status.SocksEnabled || status.GeoIPEnabled {
		t.Error("All switches should be disabled after DisableAll()")
	}

	sm.ResetToDefaults()
	status = sm.GetStatus()
	if !status.AliangEnabled || !status.SocksEnabled || status.GeoIPEnabled {
		t.Error("Switches should be reset to defaults (NoneLane=ON, SOCKS=ON, GeoIP=OFF)")
	}
}

func Test_UpdateSwitchesFromConfig(t *testing.T) {
	sm := GetSwitchManager()
	config := &model.RoutingRulesConfig{
		Settings: model.RulesSettings{AliangEnabled: false, SocksEnabled: true, GeoIPEnabled: true},
		Aliang:   model.RoutingRuleSet{SetType: model.SetTypeAliang, Rules: []model.RoutingRule{}},
		ToSocks:  model.RoutingRuleSet{SetType: model.SetTypeToSocks, Rules: []model.RoutingRule{}},
		Direct:   model.RoutingRuleSet{SetType: model.SetTypeDirect, Rules: []model.RoutingRule{}},
	}

	sm.UpdateSwitches(config)
	status := sm.GetStatus()
	if status.AliangEnabled != false || status.SocksEnabled != true || status.GeoIPEnabled != true {
		t.Error("switch status mismatch after UpdateSwitches")
	}
}
