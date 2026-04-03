package routing

import (
	"strings"
	"testing"

	"aliang.one/nursorgate/common/model"
	"aliang.one/nursorgate/processor/config"
)

func boolPtr(v bool) *bool { return &v }

func TestCompileRuntimeSnapshotFromRuntimeInputs_AIRulesPrecedeProxyRules(t *testing.T) {
	cfg := &config.Config{
		Customer: &config.CustomerConfig{
			Proxy: &config.CustomerProxyConfig{Type: "socks5"},
			AIRules: map[string]*config.CustomerAIRuleSetting{
				"openai": {
					Enble:   boolPtr(true),
					Include: []string{"api.openai.com"},
				},
			},
			ProxyRules: []string{"domain,api.openai.com,proxy"},
		},
	}

	snapshot, err := CompileRuntimeSnapshotFromRuntimeInputs(cfg, model.RulesSettings{AliangEnabled: true, SocksEnabled: true})
	if err != nil {
		t.Fatalf("CompileRuntimeSnapshotFromRuntimeInputs() error = %v", err)
	}

	decision, err := DecideRouteFromSnapshot(snapshot, &MatchContext{Domain: "api.openai.com", IP: "1.1.1.1"})
	if err != nil {
		t.Fatalf("DecideRouteFromSnapshot() error = %v", err)
	}
	if decision != RouteToAliang {
		t.Fatalf("decision = %s, want %s", decision, RouteToAliang)
	}
}

func TestCompileRuntimeSnapshotFromRuntimeInputs_NonAIRulesMatchProxyRules(t *testing.T) {
	cfg := &config.Config{
		Customer: &config.CustomerConfig{
			Proxy:      &config.CustomerProxyConfig{Type: "socks5"},
			AIRules:    map[string]*config.CustomerAIRuleSetting{},
			ProxyRules: []string{"domain,cursor.com,proxy"},
		},
	}

	snapshot, err := CompileRuntimeSnapshotFromRuntimeInputs(cfg, model.RulesSettings{AliangEnabled: true, SocksEnabled: true})
	if err != nil {
		t.Fatalf("CompileRuntimeSnapshotFromRuntimeInputs() error = %v", err)
	}

	decision, err := DecideRouteFromSnapshot(snapshot, &MatchContext{Domain: "cursor.com", IP: "8.8.8.8"})
	if err != nil {
		t.Fatalf("DecideRouteFromSnapshot() error = %v", err)
	}
	if decision != RouteToSocks {
		t.Fatalf("decision = %s, want %s", decision, RouteToSocks)
	}
}

func TestCompileRuntimeSnapshotFromRuntimeInputs_UnmatchedTrafficDefaultsToDirect(t *testing.T) {
	cfg := &config.Config{
		Customer: &config.CustomerConfig{
			Proxy:      &config.CustomerProxyConfig{Type: "socks5"},
			AIRules:    map[string]*config.CustomerAIRuleSetting{},
			ProxyRules: []string{"domain,cursor.com,proxy"},
		},
	}

	snapshot, err := CompileRuntimeSnapshotFromRuntimeInputs(cfg, model.RulesSettings{AliangEnabled: true, SocksEnabled: true})
	if err != nil {
		t.Fatalf("CompileRuntimeSnapshotFromRuntimeInputs() error = %v", err)
	}

	decision, err := DecideRouteFromSnapshot(snapshot, &MatchContext{Domain: "functional.events.data.microsoft.com", IP: "20.189.173.13"})
	if err != nil {
		t.Fatalf("DecideRouteFromSnapshot() error = %v", err)
	}
	if decision != RouteDirect {
		t.Fatalf("decision = %s, want %s", decision, RouteDirect)
	}
}

func TestCompileRuntimeSnapshotFromRuntimeInputs_ProxyTypeMapsToToSocksUpstreamType(t *testing.T) {
	tests := []struct {
		name          string
		proxyEnabled  *bool
		proxyType     string
		wantUpstream  string
		socksEnabled  bool
		wantToSocksOn bool
	}{
		{name: "customer proxy http maps to http upstream", proxyEnabled: nil, proxyType: "http", wantUpstream: "http", socksEnabled: true, wantToSocksOn: true},
		{name: "customer proxy socks5 maps to socks upstream", proxyEnabled: nil, proxyType: "socks5", wantUpstream: "socks", socksEnabled: true, wantToSocksOn: true},
		{name: "disabled customer proxy turns toSocks off", proxyEnabled: boolPtr(false), proxyType: "socks5", wantUpstream: "", socksEnabled: true, wantToSocksOn: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			canonical, err := compileCanonicalRoutingFromRuntimeInputs(&config.Config{
				Customer: &config.CustomerConfig{
					Proxy: &config.CustomerProxyConfig{Enable: tt.proxyEnabled, Type: tt.proxyType},
				},
			}, model.RulesSettings{AliangEnabled: true, SocksEnabled: tt.socksEnabled})
			if err != nil {
				t.Fatalf("compileCanonicalRoutingFromRuntimeInputs() error = %v", err)
			}

			if canonical.Egress.ToSocks.Enabled != tt.wantToSocksOn {
				t.Fatalf("toSocks.enabled = %v, want %v", canonical.Egress.ToSocks.Enabled, tt.wantToSocksOn)
			}
			if canonical.Egress.ToSocks.Upstream.Type != tt.wantUpstream {
				t.Fatalf("toSocks.upstream.type = %q, want %q", canonical.Egress.ToSocks.Upstream.Type, tt.wantUpstream)
			}
		})
	}
}

func TestCompileRuntimeSnapshotFromRuntimeInputs_DisabledProxyRulesFallbackToDirect(t *testing.T) {
	canonical, err := compileCanonicalRoutingFromRuntimeInputs(&config.Config{
		Customer: &config.CustomerConfig{
			Proxy: &config.CustomerProxyConfig{Enable: boolPtr(false), Type: "socks5"},
			ProxyRules: []string{
				"domain,cursor.com,proxy",
			},
		},
	}, model.RulesSettings{AliangEnabled: true, SocksEnabled: true})
	if err != nil {
		t.Fatalf("compileCanonicalRoutingFromRuntimeInputs() error = %v", err)
	}

	if canonical.Egress.ToSocks.Enabled {
		t.Fatal("expected toSocks to be disabled when customer proxy is disabled")
	}
	if len(canonical.Routing.Rules) != 1 {
		t.Fatalf("expected one routing rule, got %d", len(canonical.Routing.Rules))
	}
	if canonical.Routing.Rules[0].Target != "direct" {
		t.Fatalf("proxy rule target = %q, want %q", canonical.Routing.Rules[0].Target, "direct")
	}
}

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
