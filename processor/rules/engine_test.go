package rules

import (
	"net/netip"
	"testing"

	"nursor.org/nursorgate/processor/cache"
	"nursor.org/nursorgate/processor/routing"
)

func TestEvaluateWithSnapshot_DisabledTargetDeny(t *testing.T) {
	engine := &RuleEngine{}
	snapshot := routing.NewRuntimeSnapshotForDecision(
		routing.NewSnapshotBranchCapabilities(true, false, true),
		[]routing.SnapshotRule{routing.NewSnapshotRule("disabled_target", "domain", "disabled-target.example", true, routing.SnapshotActionToAliang)},
		routing.SnapshotActionDirect,
		true,
	)

	result := engine.evaluateWithSnapshot(snapshot, &EvaluationContext{
		Domain:  "disabled-target.example",
		DstIP:   netip.MustParseAddr("1.1.1.1"),
		DstPort: 443,
	})

	if result.Route != cache.RouteDeny {
		t.Fatalf("DisabledTargetDeny: route = %s, want %s", result.Route, cache.RouteDeny)
	}
	if result.Route == cache.RouteDirect {
		t.Fatalf("DisabledTargetDeny: route must not fallback to %s", cache.RouteDirect)
	}
}

func TestEvaluateWithSnapshot_UnavailableBranchDeny(t *testing.T) {
	engine := &RuleEngine{}
	snapshot := routing.NewRuntimeSnapshotForDecision(
		routing.NewSnapshotBranchCapabilities(false, true, false),
		nil,
		routing.SnapshotActionDirect,
		true,
	)

	result := engine.evaluateWithSnapshot(snapshot, &EvaluationContext{
		Domain:  "unavailable-branch.example",
		DstIP:   netip.MustParseAddr("8.8.8.8"),
		DstPort: 80,
	})

	if result.Route != cache.RouteDeny {
		t.Fatalf("UnavailableBranchDeny: route = %s, want %s", result.Route, cache.RouteDeny)
	}
	if result.Route == cache.RouteDirect {
		t.Fatalf("UnavailableBranchDeny: route must not fallback to %s", cache.RouteDirect)
	}
}

func TestCharacterizationLegacy_EvaluateWithSnapshot_DenyFailCloseBaseline(t *testing.T) {
	engine := &RuleEngine{}

	tests := []struct {
		name     string
		snapshot *routing.RuntimeSnapshot
		ctx      *EvaluationContext
	}{
		{
			name: "Disabled target branch resolves to deny",
			snapshot: routing.NewRuntimeSnapshotForDecision(
				routing.NewSnapshotBranchCapabilities(true, false, true),
				[]routing.SnapshotRule{routing.NewSnapshotRule("disabled_target", "domain", "disabled-target.example", true, routing.SnapshotActionToAliang)},
				routing.SnapshotActionDirect,
				true,
			),
			ctx: &EvaluationContext{Domain: "disabled-target.example", DstIP: netip.MustParseAddr("1.1.1.1"), DstPort: 443},
		},
		{
			name: "Unavailable branch resolves to deny",
			snapshot: routing.NewRuntimeSnapshotForDecision(
				routing.NewSnapshotBranchCapabilities(false, true, false),
				nil,
				routing.SnapshotActionDirect,
				true,
			),
			ctx: &EvaluationContext{Domain: "unavailable-branch.example", DstIP: netip.MustParseAddr("8.8.8.8"), DstPort: 80},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := engine.evaluateWithSnapshot(tt.snapshot, tt.ctx)
			if result.Route != cache.RouteDeny {
				t.Fatalf("route = %s, want %s", result.Route, cache.RouteDeny)
			}
			if result.Route == cache.RouteDirect {
				t.Fatalf("route must not fallback to %s", cache.RouteDirect)
			}
		})
	}
}
