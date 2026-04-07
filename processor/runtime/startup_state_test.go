package runtime

import (
	"path/filepath"
	"sync"
	"testing"
	"time"

	authuser "aliang.one/nursorgate/processor/auth"
)

func TestStartupState_ResetGlobalStartupStateForTest_ResetsFields(t *testing.T) {
	ResetGlobalStartupStateForTest()
	state := GetStartupState()

	state.SetStatus(READY)
	state.SetFetchSuccess(true)

	ResetGlobalStartupStateForTest()
	state = GetStartupState()

	if got := state.GetStatus(); got != UNCONFIGURED {
		t.Fatalf("status after reset = %q, want %q", got, UNCONFIGURED)
	}
	if got := state.GetFetchSuccess(); got {
		t.Fatalf("fetchSuccess after reset = %v, want false", got)
	}
}

func TestAuthUserInfo_RemainsSingleSourceOfTruthOutsideStartupState(t *testing.T) {
	baseDir := t.TempDir()
	t.Setenv("HOME", filepath.Join(baseDir, "home"))
	t.Setenv("ALIANG_CACHE_DIR", filepath.Join(baseDir, "cache"))
	authuser.ResetAuthPersistenceForTest()
	t.Cleanup(authuser.ResetAuthPersistenceForTest)

	ResetGlobalStartupStateForTest()

	if err := authuser.SaveUserInfo(&authuser.UserInfo{
		AccessToken:  "persisted-access-token",
		RefreshToken: "persisted-refresh-token",
		Username:     "persisted-user",
		UpdatedAt:    time.Now(),
	}); err != nil {
		t.Fatalf("SaveUserInfo() error = %v", err)
	}

	authuser.SetCurrentUserInfo(nil)

	got := authuser.GetCurrentUserInfoOrLoad()
	if got == nil {
		t.Fatal("GetCurrentUserInfoOrLoad() returned nil")
	}
	if got.Username != "persisted-user" {
		t.Fatalf("username = %q, want %q", got.Username, "persisted-user")
	}
}

func TestStartupState_SetAndGetStatus_ThreadSafe(t *testing.T) {
	ResetGlobalStartupStateForTest()
	state := GetStartupState()

	statuses := []StartupStatus{UNCONFIGURED, CONFIGURING, CONFIGURED, READY}

	const goroutines = 64
	const iterations = 2000

	var wg sync.WaitGroup
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				state.SetStatus(statuses[(id+j)%len(statuses)])
				_ = state.GetStatus()
			}
		}(i)
	}
	wg.Wait()

	got := state.GetStatus()
	valid := false
	for _, s := range statuses {
		if got == s {
			valid = true
			break
		}
	}
	if !valid {
		t.Fatalf("final status %q is not a valid startup status", got)
	}
}

func TestStartupState_SetAndGetFetchSuccess_ThreadSafe(t *testing.T) {
	ResetGlobalStartupStateForTest()
	state := GetStartupState()

	const goroutines = 64
	const iterations = 2000

	var wg sync.WaitGroup
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				state.SetFetchSuccess((id+j)%2 == 0)
				_ = state.GetFetchSuccess()
			}
		}(i)
	}
	wg.Wait()

	_ = state.GetFetchSuccess()
}
