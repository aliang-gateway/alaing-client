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

func TestStartupState_GetUserInfo_ReturnsCopy(t *testing.T) {
	ResetGlobalStartupStateForTest()
	state := GetStartupState()

	original := &authuser.UserInfo{
		Username:    "alice",
		Status:      "active",
		Concurrency: 3,
	}
	state.SetUserInfo(original)

	copyInfo := state.GetUserInfo()
	if copyInfo == nil {
		t.Fatal("GetUserInfo() returned nil")
	}

	copyInfo.Username = "mutated"
	copyInfo.Status = "changed"
	copyInfo.Concurrency = 999

	gotAgain := state.GetUserInfo()
	if gotAgain == nil {
		t.Fatal("GetUserInfo() returned nil on second read")
	}

	if gotAgain.Username != "alice" {
		t.Fatalf("username mutated in state: got %q, want %q", gotAgain.Username, "alice")
	}
	if gotAgain.Status != "active" {
		t.Fatalf("status mutated in state: got %q, want %q", gotAgain.Status, "active")
	}
	if gotAgain.Concurrency != 3 {
		t.Fatalf("concurrency mutated in state: got %d, want %d", gotAgain.Concurrency, 3)
	}
}

func TestStartupState_SetUserInfo_SyncsSharedAuthState(t *testing.T) {
	ResetGlobalStartupStateForTest()
	state := GetStartupState()

	state.SetUserInfo(&authuser.UserInfo{
		Username: "shared-user",
		Status:   "active",
	})

	got := authuser.GetCurrentUserInfo()
	if got == nil {
		t.Fatal("GetCurrentUserInfo() returned nil")
	}
	if got.Username != "shared-user" {
		t.Fatalf("username = %q, want %q", got.Username, "shared-user")
	}
}

func TestStartupState_GetUserInfo_FallsBackToPersistedAuthState(t *testing.T) {
	baseDir := t.TempDir()
	t.Setenv("HOME", filepath.Join(baseDir, "home"))
	t.Setenv("ALIANG_CACHE_DIR", filepath.Join(baseDir, "cache"))
	authuser.ResetAuthPersistenceForTest()
	t.Cleanup(authuser.ResetAuthPersistenceForTest)

	ResetGlobalStartupStateForTest()
	state := GetStartupState()

	if err := authuser.SaveUserInfo(&authuser.UserInfo{
		AccessToken:  "persisted-access-token",
		RefreshToken: "persisted-refresh-token",
		Username:     "persisted-user",
		UpdatedAt:    time.Now(),
	}); err != nil {
		t.Fatalf("SaveUserInfo() error = %v", err)
	}

	authuser.SetCurrentUserInfo(nil)

	got := state.GetUserInfo()
	if got == nil {
		t.Fatal("GetUserInfo() returned nil")
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
