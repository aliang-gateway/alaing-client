package routing

import (
	"fmt"
	"strings"

	"nursor.org/nursorgate/common/model"
	"nursor.org/nursorgate/processor/config"
)

type SnapshotAction string

const (
	SnapshotActionDirect   SnapshotAction = "direct"
	SnapshotActionToAliang SnapshotAction = "toAliang"
	SnapshotActionToSocks  SnapshotAction = "toSocks"
	SnapshotActionDeny     SnapshotAction = "deny"
)

type SnapshotBranchCapabilities struct {
	direct   bool
	toAliang bool
	toSocks  bool
}

func NewSnapshotBranchCapabilities(direct, toAliang, toSocks bool) SnapshotBranchCapabilities {
	return SnapshotBranchCapabilities{
		direct:   direct,
		toAliang: toAliang,
		toSocks:  toSocks,
	}
}

func (c SnapshotBranchCapabilities) Direct() bool {
	return c.direct
}

func (c SnapshotBranchCapabilities) ToAliang() bool {
	return c.toAliang
}

func (c SnapshotBranchCapabilities) ToSocks() bool {
	return c.toSocks
}

type SnapshotRule struct {
	id        string
	ruleType  string
	condition string
	enabled   bool
	target    SnapshotAction
}

func NewSnapshotRule(id, ruleType, condition string, enabled bool, target SnapshotAction) SnapshotRule {
	return SnapshotRule{
		id:        id,
		ruleType:  strings.ToLower(strings.TrimSpace(ruleType)),
		condition: strings.ToLower(strings.TrimSpace(condition)),
		enabled:   enabled,
		target:    target,
	}
}

func (r SnapshotRule) ID() string {
	return r.id
}

func (r SnapshotRule) Type() string {
	return r.ruleType
}

func (r SnapshotRule) Condition() string {
	return r.condition
}

func (r SnapshotRule) Enabled() bool {
	return r.enabled
}

func (r SnapshotRule) Target() SnapshotAction {
	return r.target
}

type RuntimeSnapshot struct {
	ingressMode     string
	capabilities    SnapshotBranchCapabilities
	rules           []SnapshotRule
	defaultAction   SnapshotAction
	explicitDeny    SnapshotAction
	hasExplicitDeny bool
	strictDeny      bool
}

func NewRuntimeSnapshotForDecision(capabilities SnapshotBranchCapabilities, rules []SnapshotRule, defaultAction SnapshotAction, strictDeny bool) *RuntimeSnapshot {
	rulesCopy := make([]SnapshotRule, len(rules))
	copy(rulesCopy, rules)
	hasDeny := false
	for _, rule := range rulesCopy {
		if rule.target == SnapshotActionDeny {
			hasDeny = true
			break
		}
	}

	return &RuntimeSnapshot{
		ingressMode:     "tun",
		capabilities:    capabilities,
		rules:           rulesCopy,
		defaultAction:   defaultAction,
		explicitDeny:    SnapshotActionDeny,
		hasExplicitDeny: hasDeny,
		strictDeny:      strictDeny,
	}
}

func (s *RuntimeSnapshot) IngressMode() string {
	if s == nil {
		return "tun"
	}
	return s.ingressMode
}

func (s *RuntimeSnapshot) BranchCapabilities() SnapshotBranchCapabilities {
	if s == nil {
		return SnapshotBranchCapabilities{direct: true}
	}
	return s.capabilities
}

func (s *RuntimeSnapshot) Rules() []SnapshotRule {
	if s == nil {
		return nil
	}
	out := make([]SnapshotRule, len(s.rules))
	copy(out, s.rules)
	return out
}

func (s *RuntimeSnapshot) DefaultAction() SnapshotAction {
	if s == nil {
		return SnapshotActionDirect
	}
	return s.defaultAction
}

func (s *RuntimeSnapshot) ExplicitDenyAction() SnapshotAction {
	if s == nil {
		return SnapshotActionDeny
	}
	return s.explicitDeny
}

func (s *RuntimeSnapshot) HasExplicitDenyRule() bool {
	if s == nil {
		return false
	}
	return s.hasExplicitDeny
}

func (s *RuntimeSnapshot) StrictUnavailableBranchDeny() bool {
	if s == nil {
		return false
	}
	return s.strictDeny
}

func CompileRuntimeSnapshot(canonical *config.CanonicalRoutingSchema) (*RuntimeSnapshot, error) {
	if canonical == nil {
		return nil, fmt.Errorf("canonical routing schema is nil")
	}

	if err := canonical.Validate(); err != nil {
		return nil, err
	}

	rules := make([]SnapshotRule, 0, len(canonical.Routing.Rules))
	hasDeny := false
	for _, rule := range canonical.Routing.Rules {
		target := SnapshotAction(rule.Target)
		rules = append(rules, SnapshotRule{
			id:        rule.ID,
			ruleType:  strings.ToLower(strings.TrimSpace(rule.Type)),
			condition: strings.ToLower(strings.TrimSpace(rule.Condition)),
			enabled:   rule.Enabled,
			target:    target,
		})
		if target == SnapshotActionDeny {
			hasDeny = true
		}
	}

	defaultAction := SnapshotActionDirect
	if canonical.Routing.DefaultEgress != "" {
		defaultAction = SnapshotAction(canonical.Routing.DefaultEgress)
	}

	return &RuntimeSnapshot{
		ingressMode: strings.ToLower(strings.TrimSpace(canonical.Ingress.Mode)),
		capabilities: SnapshotBranchCapabilities{
			direct:   canonical.Egress.Direct.Enabled,
			toAliang: canonical.Egress.ToAliang.Enabled,
			toSocks:  canonical.Egress.ToSocks.Enabled,
		},
		rules:           rules,
		defaultAction:   defaultAction,
		explicitDeny:    SnapshotActionDeny,
		hasExplicitDeny: hasDeny,
		strictDeny:      true,
	}, nil
}

func CompileRuntimeSnapshotFromLegacyConfig(cfg *model.RoutingRulesConfig) (*RuntimeSnapshot, error) {
	if cfg == nil {
		return nil, fmt.Errorf("routing rules config is nil")
	}

	rules := make([]SnapshotRule, 0, len(cfg.Aliang.Rules)+len(cfg.ToSocks.Rules)+len(cfg.Direct.Rules))
	for _, rule := range cfg.Aliang.Rules {
		rules = append(rules, SnapshotRule{
			id:        rule.ID,
			ruleType:  strings.ToLower(strings.TrimSpace(string(rule.Type))),
			condition: strings.ToLower(strings.TrimSpace(rule.Condition)),
			enabled:   rule.Enabled,
			target:    SnapshotActionToAliang,
		})
	}
	for _, rule := range cfg.ToSocks.Rules {
		rules = append(rules, SnapshotRule{
			id:        rule.ID,
			ruleType:  strings.ToLower(strings.TrimSpace(string(rule.Type))),
			condition: strings.ToLower(strings.TrimSpace(rule.Condition)),
			enabled:   rule.Enabled,
			target:    SnapshotActionToSocks,
		})
	}
	for _, rule := range cfg.Direct.Rules {
		rules = append(rules, SnapshotRule{
			id:        rule.ID,
			ruleType:  strings.ToLower(strings.TrimSpace(string(rule.Type))),
			condition: strings.ToLower(strings.TrimSpace(rule.Condition)),
			enabled:   rule.Enabled,
			target:    SnapshotActionDirect,
		})
	}

	hasDeny := false
	for _, rule := range rules {
		if rule.target == SnapshotActionDeny {
			hasDeny = true
			break
		}
	}

	return &RuntimeSnapshot{
		ingressMode: "tun",
		capabilities: SnapshotBranchCapabilities{
			direct:   true,
			toAliang: cfg.Settings.AliangEnabled,
			toSocks:  cfg.Settings.SocksEnabled,
		},
		rules:           rules,
		defaultAction:   SnapshotActionDirect,
		explicitDeny:    SnapshotActionDeny,
		hasExplicitDeny: hasDeny,
		strictDeny:      false,
	}, nil
}

func CompileRuntimeSnapshotFromRuntimeInputs(cfg *config.Config, switches model.RulesSettings) (*RuntimeSnapshot, error) {
	legacy := model.NewRoutingRulesConfig()
	legacy.Settings.AliangEnabled = switches.AliangEnabled
	legacy.Settings.SocksEnabled = switches.SocksEnabled
	legacy.Settings.GeoIPEnabled = switches.GeoIPEnabled

	if cfg != nil {
		legacy.Aliang.Rules = make([]model.RoutingRule, 0, len(cfg.SNIAllowlist))
		for i, domain := range cfg.SNIAllowlist {
			normalizedDomain := strings.ToLower(strings.TrimSpace(domain))
			if normalizedDomain == "" {
				continue
			}
			legacy.Aliang.Rules = append(legacy.Aliang.Rules, model.RoutingRule{
				ID:        fmt.Sprintf("aliang_allowlist_%d", i),
				Type:      model.RuleTypeDomain,
				Condition: normalizedDomain,
				Enabled:   true,
			})
		}
		legacy.Aliang.Count = len(legacy.Aliang.Rules)
		legacy.Settings.SocksEnabled = switches.SocksEnabled && cfg.SocksProxy != nil
	}

	snapshot, err := CompileRuntimeSnapshotFromLegacyConfig(legacy)
	if err != nil {
		return nil, err
	}

	if legacy.Settings.SocksEnabled {
		snapshot.defaultAction = SnapshotActionToSocks
	}

	return snapshot, nil
}
