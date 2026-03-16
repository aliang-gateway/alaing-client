package runtime

import (
	"sync"
	"testing"

	authuser "nursor.org/nursorgate/processor/auth"
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
		Username:  "alice",
		PlanName:  "pro",
		AIAskUsed: 3,
	}
	state.SetUserInfo(original)

	copyInfo := state.GetUserInfo()
	if copyInfo == nil {
		t.Fatal("GetUserInfo() returned nil")
	}

	copyInfo.Username = "mutated"
	copyInfo.PlanName = "changed"
	copyInfo.AIAskUsed = 999

	gotAgain := state.GetUserInfo()
	if gotAgain == nil {
		t.Fatal("GetUserInfo() returned nil on second read")
	}

	if gotAgain.Username != "alice" {
		t.Fatalf("username mutated in state: got %q, want %q", gotAgain.Username, "alice")
	}
	if gotAgain.PlanName != "pro" {
		t.Fatalf("plan mutated in state: got %q, want %q", gotAgain.PlanName, "pro")
	}
	if gotAgain.AIAskUsed != 3 {
		t.Fatalf("AIAskUsed mutated in state: got %d, want %d", gotAgain.AIAskUsed, 3)
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
