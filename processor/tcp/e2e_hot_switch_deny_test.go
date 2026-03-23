package tcp

import (
	"encoding/json"
	"fmt"
	"sync"
	"testing"

	"nursor.org/nursorgate/outbound"
	"nursor.org/nursorgate/processor/config"
	"nursor.org/nursorgate/processor/routing"
)

func TestE2EHotSwitchDeny(t *testing.T) {
	config.ResetRoutingApplyStoreForTest()
	t.Cleanup(config.ResetRoutingApplyStoreForTest)

	registry := outbound.GetRegistry()
	registry.Clear()
	t.Cleanup(registry.Clear)

	applyPayload := []byte(`{
  "version": 1,
  "ingress": {"mode": "http"},
  "egress": {
    "direct": {"enabled": true},
    "toAliang": {"enabled": true},
    "toSocks": {"enabled": true, "upstream": {"type": "socks"}}
  },
  "routing": {
    "default_egress": "direct",
    "rules": [
      {"id": "r1", "type": "domain", "condition": "deny.example", "enabled": true, "target": "toAliang"}
    ]
  }
}`)

	if _, err := config.GetRoutingApplyStore().Apply(applyPayload, func(cfg *config.CanonicalRoutingSchema) (any, error) {
		return routing.CompileRuntimeSnapshot(cfg)
	}); err != nil {
		t.Fatalf("apply config error = %v", err)
	}

	canonical := config.GetRoutingApplyStore().ActiveCanonicalSchema()
	if canonical == nil || canonical.Ingress.Mode != "http" {
		t.Fatalf("expected ingress mode http after apply, got %#v", canonical)
	}

	canonical.Ingress.Mode = "tun"
	updatedRaw, err := json.Marshal(canonical)
	if err != nil {
		t.Fatalf("marshal updated canonical error = %v", err)
	}
	if _, err := config.GetRoutingApplyStore().Apply(updatedRaw, func(cfg *config.CanonicalRoutingSchema) (any, error) {
		return routing.CompileRuntimeSnapshot(cfg)
	}); err != nil {
		t.Fatalf("hot switch apply error = %v", err)
	}

	snapshot, ok := config.GetRoutingApplyStore().ActiveSnapshot().(*routing.RuntimeSnapshot)
	if !ok || snapshot == nil {
		t.Fatal("active snapshot missing")
	}
	if snapshot.IngressMode() != "tun" {
		t.Fatalf("snapshot ingress mode=%q, want tun", snapshot.IngressMode())
	}

	decision, err := routing.DecideRouteFromSnapshot(snapshot, &routing.MatchContext{Domain: "deny.example", IP: "1.1.1.1"})
	if err != nil {
		t.Fatalf("DecideRouteFromSnapshot() error = %v", err)
	}
	if decision != routing.RouteToAliang {
		t.Fatalf("route decision=%s, want %s", decision, routing.RouteToAliang)
	}

	_, err = (&TCPConnectionHandler{}).getAliangProxyForExecution()
	if err == nil {
		t.Fatal("expected deny error for unavailable aliang proxy")
	}
	if !IsBranchDenyError(err) {
		t.Fatalf("error should be BranchDenyError, got %T", err)
	}
	if reason := BranchDenyReason(err); reason != DenyReasonToAliangUnavailable {
		t.Fatalf("deny reason=%q, want %q", reason, DenyReasonToAliangUnavailable)
	}

	activeMode := config.GetRoutingApplyStore().ActiveCanonicalSchema().Ingress.Mode
	if activeMode != snapshot.IngressMode() {
		t.Fatalf("ingress mode mismatch canonical=%q snapshot=%q", activeMode, snapshot.IngressMode())
	}
}

func TestE2EConcurrentApplyRead(t *testing.T) {
	config.ResetRoutingApplyStoreForTest()
	t.Cleanup(config.ResetRoutingApplyStoreForTest)

	registry := outbound.GetRegistry()
	registry.Clear()
	t.Cleanup(registry.Clear)

	apply := func(mode string, id string) (*config.RoutingApplyResult, error) {
		payload := []byte(fmt.Sprintf(`{
  "version": 1,
  "ingress": {"mode": %q},
  "egress": {
    "direct": {"enabled": true},
    "toAliang": {"enabled": true},
    "toSocks": {"enabled": true, "upstream": {"type": "socks"}}
  },
	"routing": {
	  "default_egress": "direct",
	  "rules": [
	    {"id": %q, "type": "domain", "condition": "concurrent.example", "enabled": true, "target": "toAliang"}
	  ]
	}
}`, mode, id))
		return config.GetRoutingApplyStore().Apply(payload, func(cfg *config.CanonicalRoutingSchema) (any, error) {
			return routing.CompileRuntimeSnapshot(cfg)
		})
	}

	if _, err := apply("http", "seed"); err != nil {
		t.Fatalf("seed apply error = %v", err)
	}

	store := config.GetRoutingApplyStore()
	const writers = 4
	const readers = 12
	const loops = 75

	var wg sync.WaitGroup
	errCh := make(chan error, writers+readers)

	for w := 0; w < writers; w++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			for i := 0; i < loops; i++ {
				mode := "http"
				if (idx+i)%2 == 0 {
					mode = "tun"
				}
				id := fmt.Sprintf("w%d-%d", idx, i)
				if _, err := apply(mode, id); err != nil {
					errCh <- fmt.Errorf("apply failed idx=%d iter=%d: %w", idx, i, err)
					return
				}
			}
		}(w)
	}

	for r := 0; r < readers; r++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < loops; i++ {
				stable := false
				for attempt := 0; attempt < 50; attempt++ {
					active, version, hash := store.ActiveSnapshotVersionHash()
					if active == nil {
						errCh <- fmt.Errorf("nil active snapshot at iter=%d", i)
						return
					}
					if version == 0 || hash == "" {
						errCh <- fmt.Errorf("invalid version/hash at iter=%d version=%d hash=%q", i, version, hash)
						return
					}
					canonical := store.ActiveCanonicalSchema()
					active2, version2, hash2 := store.ActiveSnapshotVersionHash()
					version3, hash3 := store.ActiveVersionHash()
					if version != version2 || hash != hash2 || version2 != version3 || hash2 != hash3 {
						continue
					}
					if canonical == nil {
						errCh <- fmt.Errorf("nil canonical schema at iter=%d", i)
						return
					}
					if canonical.Ingress.Mode != "http" && canonical.Ingress.Mode != "tun" {
						errCh <- fmt.Errorf("unexpected ingress mode=%q", canonical.Ingress.Mode)
						return
					}
					snapshot, ok := active2.(*routing.RuntimeSnapshot)
					if !ok || snapshot == nil {
						errCh <- fmt.Errorf("snapshot type mismatch")
						return
					}
					if snapshot.IngressMode() != canonical.Ingress.Mode {
						errCh <- fmt.Errorf("mode mismatch canonical=%q snapshot=%q", canonical.Ingress.Mode, snapshot.IngressMode())
						return
					}
					stable = true
					break
				}
				if !stable {
					errCh <- fmt.Errorf("unable to read stable state at iter=%d", i)
					return
				}
			}
		}()
	}

	wg.Wait()
	close(errCh)
	for err := range errCh {
		if err != nil {
			t.Fatal(err)
		}
	}
}
