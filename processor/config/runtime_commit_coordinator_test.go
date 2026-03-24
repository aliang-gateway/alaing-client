package config

import (
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestEffectiveConfigCommitCoordinator_CommitSuccess(t *testing.T) {
	ResetEffectiveConfigCommitCoordinatorForTest()
	coordinator := GetEffectiveConfigCommitCoordinator()

	var order []string
	result, err := coordinator.Commit(
		&EffectiveConfigSnapshot{UUID: "cfg-1", Software: "opencode", FilePath: "/tmp/a", Content: "a"},
		func(snapshot *EffectiveConfigSnapshot) error {
			order = append(order, "file:"+snapshot.UUID)
			return nil
		},
		func(snapshot *EffectiveConfigSnapshot) error {
			order = append(order, "db:"+snapshot.UUID)
			return nil
		},
	)
	if err != nil {
		t.Fatalf("Commit() error = %v", err)
	}
	if result.Version != 1 {
		t.Fatalf("version = %d, want 1", result.Version)
	}
	if len(order) != 2 || order[0] != "file:cfg-1" || order[1] != "db:cfg-1" {
		t.Fatalf("unexpected stage order: %+v", order)
	}

	active := coordinator.ActiveSnapshot()
	if active == nil || active.UUID != "cfg-1" {
		t.Fatalf("active snapshot = %+v, want cfg-1", active)
	}
	committed := coordinator.LastCommittedSnapshot()
	if committed == nil || committed.UUID != "cfg-1" {
		t.Fatalf("committed snapshot = %+v, want cfg-1", committed)
	}
}

func TestEffectiveConfigCommitCoordinator_RollbackOnFileFailure(t *testing.T) {
	ResetEffectiveConfigCommitCoordinatorForTest()
	coordinator := GetEffectiveConfigCommitCoordinator()

	_, err := coordinator.Commit(
		&EffectiveConfigSnapshot{UUID: "seed", Software: "opencode", FilePath: "/tmp/a", Content: "seed"},
		func(snapshot *EffectiveConfigSnapshot) error { return nil },
		func(snapshot *EffectiveConfigSnapshot) error { return nil },
	)
	if err != nil {
		t.Fatalf("seed Commit() error = %v", err)
	}

	err = nil
	_, err = coordinator.Commit(
		&EffectiveConfigSnapshot{UUID: "next", Software: "opencode", FilePath: "/tmp/a", Content: "next"},
		func(snapshot *EffectiveConfigSnapshot) error { return fmt.Errorf("file fail") },
		func(snapshot *EffectiveConfigSnapshot) error {
			t.Fatal("db callback should not run when file callback fails")
			return nil
		},
	)
	if err == nil {
		t.Fatal("expected file-stage commit failure")
	}

	active := coordinator.ActiveSnapshot()
	if active == nil || active.UUID != "seed" {
		t.Fatalf("active after file failure = %+v, want seed", active)
	}
	committed := coordinator.LastCommittedSnapshot()
	if committed == nil || committed.UUID != "seed" {
		t.Fatalf("committed after file failure = %+v, want seed", committed)
	}
	if got := coordinator.Version(); got != 1 {
		t.Fatalf("version after file failure = %d, want 1", got)
	}
}

func TestEffectiveConfigCommitCoordinator_RollbackOnDBFailure(t *testing.T) {
	ResetEffectiveConfigCommitCoordinatorForTest()
	coordinator := GetEffectiveConfigCommitCoordinator()

	fileState := ""
	_, err := coordinator.Commit(
		&EffectiveConfigSnapshot{UUID: "seed", Software: "opencode", FilePath: "/tmp/a", Content: "seed"},
		func(snapshot *EffectiveConfigSnapshot) error {
			fileState = snapshot.Content
			return nil
		},
		func(snapshot *EffectiveConfigSnapshot) error { return nil },
	)
	if err != nil {
		t.Fatalf("seed Commit() error = %v", err)
	}

	stages := make([]string, 0, 3)
	_, err = coordinator.Commit(
		&EffectiveConfigSnapshot{UUID: "next", Software: "opencode", FilePath: "/tmp/a", Content: "next"},
		func(snapshot *EffectiveConfigSnapshot) error {
			stages = append(stages, "file:"+snapshot.UUID)
			fileState = snapshot.Content
			return nil
		},
		func(snapshot *EffectiveConfigSnapshot) error {
			stages = append(stages, "db:"+snapshot.UUID)
			return fmt.Errorf("db fail")
		},
	)
	if err == nil {
		t.Fatal("expected db-stage commit failure")
	}

	if fileState != "seed" {
		t.Fatalf("file rollback failed, fileState=%q want %q", fileState, "seed")
	}
	if len(stages) != 3 || stages[0] != "file:next" || stages[1] != "db:next" || stages[2] != "file:seed" {
		t.Fatalf("unexpected stage sequence with rollback: %+v", stages)
	}

	active := coordinator.ActiveSnapshot()
	if active == nil || active.UUID != "seed" {
		t.Fatalf("active after db failure = %+v, want seed", active)
	}
	committed := coordinator.LastCommittedSnapshot()
	if committed == nil || committed.UUID != "seed" {
		t.Fatalf("committed after db failure = %+v, want seed", committed)
	}
	if got := coordinator.Version(); got != 1 {
		t.Fatalf("version after db failure = %d, want 1", got)
	}
}

func TestEffectiveConfigCommitCoordinator_SerializesConcurrentCommits(t *testing.T) {
	ResetEffectiveConfigCommitCoordinatorForTest()
	coordinator := GetEffectiveConfigCommitCoordinator()

	const writers = 20
	var inFile int64
	var maxInFile int64
	var wg sync.WaitGroup
	errCh := make(chan error, writers)

	for i := 0; i < writers; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			_, err := coordinator.Commit(
				&EffectiveConfigSnapshot{UUID: fmt.Sprintf("cfg-%d", idx), Software: "opencode", FilePath: "/tmp/a", Content: fmt.Sprintf("%d", idx)},
				func(snapshot *EffectiveConfigSnapshot) error {
					curr := atomic.AddInt64(&inFile, 1)
					for {
						prev := atomic.LoadInt64(&maxInFile)
						if curr <= prev || atomic.CompareAndSwapInt64(&maxInFile, prev, curr) {
							break
						}
					}
					time.Sleep(5 * time.Millisecond)
					atomic.AddInt64(&inFile, -1)
					return nil
				},
				func(snapshot *EffectiveConfigSnapshot) error { return nil },
			)
			if err != nil {
				errCh <- err
			}
		}(i)
	}

	wg.Wait()
	close(errCh)
	for err := range errCh {
		if err != nil {
			t.Fatalf("commit error = %v", err)
		}
	}

	if atomic.LoadInt64(&maxInFile) != 1 {
		t.Fatalf("expected serialized file stage, max concurrent file callbacks = %d", atomic.LoadInt64(&maxInFile))
	}
	if got := coordinator.Version(); got != writers {
		t.Fatalf("version after concurrent commits = %d, want %d", got, writers)
	}
}
