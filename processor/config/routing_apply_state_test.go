package config_test

import (
	"fmt"
	"sync"
	"testing"

	"aliang.one/nursorgate/processor/config"
)

func TestAtomicApplySuccess(t *testing.T) {
	config.ResetRoutingApplyStoreForTest()

	payload := []byte(`{
		"version": 1,
		"ingress": {"mode": "tun"},
		"egress": {
			"direct": {"enabled": true},
			"toAliang": {"enabled": true},
			"toSocks": {"enabled": true, "upstream": {"type": "socks"}}
		},
		"routing": {
			"default_egress": "direct",
			"rules": []
		}
	}`)

	store := config.GetRoutingApplyStore()
	beforeVersion, beforeHash := store.ActiveVersionHash()
	if beforeVersion != 0 || beforeHash != "" {
		t.Fatalf("initial state should be empty, got version=%d hash=%q", beforeVersion, beforeHash)
	}

	builderCalls := 0
	result, err := store.Apply(payload, func(canonical *config.CanonicalRoutingSchema) (any, error) {
		builderCalls++
		return struct{ marker string }{marker: canonical.Ingress.Mode}, nil
	})
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}

	if builderCalls != 1 {
		t.Fatalf("snapshot builder calls = %d, want 1", builderCalls)
	}
	if result.Version != 1 {
		t.Fatalf("result version = %d, want 1", result.Version)
	}
	if result.Hash == "" {
		t.Fatal("result hash is empty")
	}

	active, version, hash := store.ActiveSnapshotVersionHash()
	if active == nil {
		t.Fatal("active snapshot is nil after successful apply")
	}
	if version != 1 {
		t.Fatalf("active version = %d, want 1", version)
	}
	if hash != result.Hash {
		t.Fatalf("active hash = %q, want %q", hash, result.Hash)
	}
}

func TestAtomicApplyRollback(t *testing.T) {
	config.ResetRoutingApplyStoreForTest()

	validPayload := []byte(`{
		"version": 1,
		"ingress": {"mode": "tun"},
		"egress": {
			"direct": {"enabled": true},
			"toAliang": {"enabled": true},
			"toSocks": {"enabled": true, "upstream": {"type": "socks"}}
		},
		"routing": {
			"default_egress": "direct",
			"rules": []
		}
	}`)

	store := config.GetRoutingApplyStore()
	_, err := store.Apply(validPayload, func(canonical *config.CanonicalRoutingSchema) (any, error) {
		return struct{ marker string }{marker: canonical.Ingress.Mode}, nil
	})
	if err != nil {
		t.Fatalf("seed Apply() error = %v", err)
	}

	seedSnapshot, seedVersion, seedHash := store.ActiveSnapshotVersionHash()
	if seedSnapshot == nil {
		t.Fatal("seed snapshot is nil")
	}

	failedPayload := []byte(`{
		"version": 1,
		"ingress": {"mode": "tun"},
		"egress": {
			"direct": {"enabled": true},
			"toAliang": {"enabled": false},
			"toSocks": {"enabled": true, "upstream": {"type": "socks"}}
		},
		"routing": {
			"default_egress": "direct",
			"rules": [
				{"id":"s1","type":"domain","condition":"example.com","enabled":true,"target":"toAliang"}
			]
		}
	}`)

	const readers = 16
	const loops = 1000

	errCh := make(chan error, readers+1)
	var wg sync.WaitGroup

	for i := 0; i < readers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < loops; j++ {
				snapshot, version, hash := store.ActiveSnapshotVersionHash()
				if version != seedVersion || hash != seedHash || snapshot != seedSnapshot {
					errCh <- fmt.Errorf("observed mixed state version=%d hash=%q snapshot_changed=%v", version, hash, snapshot != seedSnapshot)
					return
				}
			}
		}()
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		for j := 0; j < loops; j++ {
			if _, applyErr := store.Apply(failedPayload, func(canonical *config.CanonicalRoutingSchema) (any, error) {
				return struct{ marker string }{marker: canonical.Ingress.Mode}, nil
			}); applyErr == nil {
				errCh <- fmt.Errorf("expected apply error on iteration %d", j)
				return
			}
		}
	}()

	wg.Wait()
	close(errCh)
	for err := range errCh {
		if err != nil {
			t.Fatal(err)
		}
	}

	finalSnapshot, finalVersion, finalHash := store.ActiveSnapshotVersionHash()
	if finalVersion != seedVersion {
		t.Fatalf("final version = %d, want %d", finalVersion, seedVersion)
	}
	if finalHash != seedHash {
		t.Fatalf("final hash = %q, want %q", finalHash, seedHash)
	}
	if finalSnapshot != seedSnapshot {
		t.Fatal("final snapshot changed after failed apply")
	}
}

func TestReloadMetrics(t *testing.T) {
	config.ResetRoutingApplyStoreForTest()

	payload := []byte(`{
		"version": 1,
		"ingress": {"mode": "tun"},
		"egress": {
			"direct": {"enabled": true},
			"toAliang": {"enabled": true},
			"toSocks": {"enabled": true, "upstream": {"type": "socks"}}
		},
		"routing": {
			"default_egress": "direct",
			"rules": []
		}
	}`)

	store := config.GetRoutingApplyStore()
	counters := store.ApplyCounters()
	if counters.AttemptCount != 0 || counters.SuccessCount != 0 || counters.FailureCount != 0 || counters.RollbackCount != 0 {
		t.Fatalf("expected zeroed counters, got %+v", counters)
	}

	if _, err := store.Apply(payload, func(canonical *config.CanonicalRoutingSchema) (any, error) {
		return struct{ marker string }{marker: canonical.Ingress.Mode}, nil
	}); err != nil {
		t.Fatalf("Apply() error = %v", err)
	}

	_, _ = store.ActiveVersionHash()
	_, _, _ = store.ActiveSnapshotVersionHash()

	if _, err := store.Apply(nil, func(canonical *config.CanonicalRoutingSchema) (any, error) {
		return struct{ marker string }{marker: canonical.Ingress.Mode}, nil
	}); err == nil {
		t.Fatal("expected error on empty payload")
	}

	if _, err := store.Apply(payload, func(canonical *config.CanonicalRoutingSchema) (any, error) {
		return nil, fmt.Errorf("boom")
	}); err == nil {
		t.Fatal("expected error on snapshot build")
	}

	counters = store.ApplyCounters()
	if counters.AttemptCount != 3 {
		t.Fatalf("attempts = %d, want 3", counters.AttemptCount)
	}
	if counters.SuccessCount != 1 {
		t.Fatalf("successes = %d, want 1", counters.SuccessCount)
	}
	if counters.FailureCount != 2 {
		t.Fatalf("failures = %d, want 2", counters.FailureCount)
	}
	if counters.RollbackCount != 1 {
		t.Fatalf("rollbacks = %d, want 1", counters.RollbackCount)
	}
	if counters.ActiveVersionHashReads != 1 {
		t.Fatalf("version reads = %d, want 1", counters.ActiveVersionHashReads)
	}
	if counters.ActiveSnapshotVersionHashes != 1 {
		t.Fatalf("snapshot reads = %d, want 1", counters.ActiveSnapshotVersionHashes)
	}
}
