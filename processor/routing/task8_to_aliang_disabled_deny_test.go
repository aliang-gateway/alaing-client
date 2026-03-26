package routing

import "testing"

func TestToAliangDisabledDeny(t *testing.T) {
	snapshot := NewRuntimeSnapshotForDecision(
		NewSnapshotBranchCapabilities(true, false, true),
		[]SnapshotRule{NewSnapshotRule("r1", "domain", "disabled-aliang.example", true, SnapshotActionToAliang)},
		SnapshotActionDirect,
		true,
	)

	decision, err := DecideRouteFromSnapshot(snapshot, &MatchContext{Domain: "disabled-aliang.example", IP: "1.1.1.1"})
	if err != nil {
		t.Fatalf("DecideRouteFromSnapshot() error = %v", err)
	}
	if decision != RouteDeny {
		t.Fatalf("decision = %s, want %s", decision, RouteDeny)
	}
}
