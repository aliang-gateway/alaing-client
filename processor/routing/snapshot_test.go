package routing

import (
	"strings"
	"testing"

	"nursor.org/nursorgate/common/model"
	"nursor.org/nursorgate/processor/config"
)

func TestSnapshotCompile_DisabledTargetDeny(t *testing.T) {
	snapshot := NewRuntimeSnapshotForDecision(
		NewSnapshotBranchCapabilities(true, false, true),
		[]SnapshotRule{NewSnapshotRule("deny_disabled_target", "domain", "disabled-target.example", true, SnapshotActionToAliang)},
		SnapshotActionDirect,
		true,
	)

	decision, err := DecideRouteFromSnapshot(snapshot, &MatchContext{Domain: "disabled-target.example", IP: "1.1.1.1"})
	if err != nil {
		t.Fatalf("DecideRouteFromSnapshot() error = %v", err)
	}
	if decision != RouteDeny {
		t.Fatalf("DisabledTargetDeny: decision = %s, want %s", decision, RouteDeny)
	}
	if decision == RouteDirect {
		t.Fatalf("DisabledTargetDeny: decision must not fallback to %s", RouteDirect)
	}
}

func TestSnapshotCompile_UnavailableBranchDeny(t *testing.T) {
	snapshot := NewRuntimeSnapshotForDecision(
		NewSnapshotBranchCapabilities(false, true, false),
		nil,
		SnapshotActionDirect,
		true,
	)

	decision, err := DecideRouteFromSnapshot(snapshot, &MatchContext{Domain: "unavailable-branch.example", IP: "8.8.8.8"})
	if err != nil {
		t.Fatalf("DecideRouteFromSnapshot() error = %v", err)
	}
	if decision != RouteDeny {
		t.Fatalf("UnavailableBranchDeny: decision = %s, want %s", decision, RouteDeny)
	}
	if decision == RouteDirect {
		t.Fatalf("UnavailableBranchDeny: decision must not fallback to %s", RouteDirect)
	}
}

func TestSnapshotCompile_CanonicalDisabledBranchRejectedAtCompile(t *testing.T) {
	canonical := &config.CanonicalRoutingSchema{
		Version: config.CanonicalRoutingSchemaVersion,
		Ingress: config.CanonicalIngressConfig{Mode: "tun"},
		Egress: config.CanonicalEgressConfig{
			Direct:   config.CanonicalEgressBranch{Enabled: true},
			ToAliang: config.CanonicalEgressBranch{Enabled: false},
			ToSocks: config.CanonicalSocksEgressBranch{
				Enabled:  true,
				Upstream: config.CanonicalSocksUpstream{Type: "socks"},
			},
		},
		Routing: config.CanonicalRoutingConfig{
			Rules: []config.CanonicalRoutingRule{{
				ID:        "r1",
				Type:      "domain",
				Condition: "example.com",
				Enabled:   true,
				Target:    "toAliang",
			}},
			DefaultEgress: "direct",
		},
	}

	_, err := CompileRuntimeSnapshot(canonical)
	if err == nil {
		t.Fatal("expected compile error for rule targeting disabled branch")
	}
	if !strings.Contains(err.Error(), "target branch toAliang is disabled") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSnapshotCompile_LegacyDisabledBranchDoesNotHardFail(t *testing.T) {
	legacy := &model.RoutingRulesConfig{
		Settings: model.RulesSettings{
			AliangEnabled: false,
			SocksEnabled:  true,
			GeoIPEnabled:  false,
		},
		Aliang: model.RoutingRuleSet{SetType: model.SetTypeAliang, Rules: []model.RoutingRule{{
			ID: "a1", Type: model.RuleTypeDomain, Condition: "example.com", Enabled: true,
		}}},
		ToSocks: model.RoutingRuleSet{SetType: model.SetTypeToSocks, Rules: []model.RoutingRule{{
			ID: "s1", Type: model.RuleTypeDomain, Condition: "example.com", Enabled: true,
		}}},
		Direct: model.RoutingRuleSet{SetType: model.SetTypeDirect, Rules: []model.RoutingRule{}},
	}

	snapshot, err := CompileRuntimeSnapshotFromLegacyConfig(legacy)
	if err != nil {
		t.Fatalf("CompileRuntimeSnapshotFromLegacyConfig() error = %v", err)
	}

	decision, err := DecideRouteFromSnapshot(snapshot, &MatchContext{Domain: "example.com", IP: "1.1.1.1"})
	if err != nil {
		t.Fatalf("DecideRouteFromSnapshot() error = %v", err)
	}
	if decision != RouteToSocks {
		t.Fatalf("decision = %s, want %s", decision, RouteToSocks)
	}
}

func TestSnapshotCompile_LegacyDisabledBranchRuleSkipped(t *testing.T) {
	legacy := &model.RoutingRulesConfig{
		Settings: model.RulesSettings{
			AliangEnabled: false,
			SocksEnabled:  true,
			GeoIPEnabled:  false,
		},
		Aliang: model.RoutingRuleSet{SetType: model.SetTypeAliang, Rules: []model.RoutingRule{{
			ID: "a1", Type: model.RuleTypeDomain, Condition: "example.com", Enabled: true,
		}}},
		ToSocks: model.RoutingRuleSet{SetType: model.SetTypeToSocks, Rules: []model.RoutingRule{{
			ID: "s1", Type: model.RuleTypeDomain, Condition: "example.com", Enabled: true,
		}}},
		Direct: model.RoutingRuleSet{SetType: model.SetTypeDirect, Rules: []model.RoutingRule{}},
	}

	snapshot, err := CompileRuntimeSnapshotFromLegacyConfig(legacy)
	if err != nil {
		t.Fatalf("CompileRuntimeSnapshotFromLegacyConfig() error = %v", err)
	}

	decision, err := DecideRouteFromSnapshot(snapshot, &MatchContext{Domain: "example.com", IP: "1.1.1.1"})
	if err != nil {
		t.Fatalf("DecideRouteFromSnapshot() error = %v", err)
	}
	if decision != RouteToSocks {
		t.Fatalf("decision = %s, want %s", decision, RouteToSocks)
	}
}

func TestSnapshotSingleSource_LegacyAndDirectSnapshotDecideSame(t *testing.T) {
	legacy := &model.RoutingRulesConfig{
		Settings: model.RulesSettings{AliangEnabled: true, SocksEnabled: true, GeoIPEnabled: false},
		Aliang: model.RoutingRuleSet{SetType: model.SetTypeAliang, Rules: []model.RoutingRule{{
			ID: "a1", Type: model.RuleTypeDomain, Condition: "example.com", Enabled: true,
		}}},
		ToSocks: model.RoutingRuleSet{SetType: model.SetTypeToSocks, Rules: []model.RoutingRule{{
			ID: "s1", Type: model.RuleTypeDomain, Condition: "*.example.net", Enabled: true,
		}}},
		Direct: model.RoutingRuleSet{SetType: model.SetTypeDirect, Rules: []model.RoutingRule{}},
	}

	compiled, err := CompileRuntimeSnapshotFromLegacyConfig(legacy)
	if err != nil {
		t.Fatalf("CompileRuntimeSnapshotFromLegacyConfig() error = %v", err)
	}

	ctx := &MatchContext{Domain: "example.com", IP: "8.8.8.8"}
	fromLegacy, err := DecideRoute(legacy, ctx)
	if err != nil {
		t.Fatalf("DecideRoute() error = %v", err)
	}
	fromSnapshot, err := DecideRouteFromSnapshot(compiled, ctx)
	if err != nil {
		t.Fatalf("DecideRouteFromSnapshot() error = %v", err)
	}

	if fromLegacy != fromSnapshot {
		t.Fatalf("legacy path decision = %s, snapshot path decision = %s", fromLegacy, fromSnapshot)
	}
}

func TestCharacterizationLegacy_SnapshotFailClose_DenyOnDisabledOrUnavailable(t *testing.T) {
	tests := []struct {
		name     string
		snapshot *RuntimeSnapshot
		ctx      *MatchContext
	}{
		{
			name: "Disabled branch target denies instead of falling back",
			snapshot: NewRuntimeSnapshotForDecision(
				NewSnapshotBranchCapabilities(true, false, true),
				[]SnapshotRule{NewSnapshotRule("deny_disabled_target", "domain", "disabled.example", true, SnapshotActionToAliang)},
				SnapshotActionDirect,
				true,
			),
			ctx: &MatchContext{Domain: "disabled.example", IP: "1.1.1.1"},
		},
		{
			name: "Unavailable default branch denies instead of falling back",
			snapshot: NewRuntimeSnapshotForDecision(
				NewSnapshotBranchCapabilities(false, true, false),
				nil,
				SnapshotActionDirect,
				true,
			),
			ctx: &MatchContext{Domain: "unavailable.example", IP: "8.8.8.8"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			decision, err := DecideRouteFromSnapshot(tt.snapshot, tt.ctx)
			if err != nil {
				t.Fatalf("DecideRouteFromSnapshot() error = %v", err)
			}
			if decision != RouteDeny {
				t.Fatalf("decision = %s, want %s", decision, RouteDeny)
			}
			if decision == RouteDirect {
				t.Fatalf("decision must not fallback to %s", RouteDirect)
			}
		})
	}
}
