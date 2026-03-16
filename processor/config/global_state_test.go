package config

import (
	"sync"
	"testing"
)

func TestConfig_ResetGlobalConfigForTest_ClearsConfigAndFlags(t *testing.T) {
	ResetGlobalConfigForTest()

	SetGlobalConfig(&Config{APIServer: "https://api.example.com"})
	SetUsingDefaultConfig(true)
	SetHasLocalUserInfo(true)

	ResetGlobalConfigForTest()

	if got := GetGlobalConfig(); got != nil {
		t.Fatalf("GetGlobalConfig() after reset = %#v, want nil", got)
	}
	if got := IsUsingDefaultConfig(); got {
		t.Fatalf("IsUsingDefaultConfig() after reset = %v, want false", got)
	}
	if got := HasLocalUserInfo(); got {
		t.Fatalf("HasLocalUserInfo() after reset = %v, want false", got)
	}
}

func TestConfig_SetAndGetGlobalConfig_ThreadSafe(t *testing.T) {
	ResetGlobalConfigForTest()

	const goroutines = 64
	const iterations = 2000

	var wg sync.WaitGroup
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				cfg := &Config{
					APIServer:    "https://api.example.com",
					CurrentProxy: "door",
				}
				if (id+j)%2 == 0 {
					cfg.CurrentProxy = "nonelane"
				}
				SetGlobalConfig(cfg)
				_ = GetGlobalConfig()
			}
		}(i)
	}
	wg.Wait()

	finalCfg := GetGlobalConfig()
	if finalCfg == nil {
		t.Fatal("GetGlobalConfig() returned nil after concurrent SetGlobalConfig calls")
	}
	if finalCfg.CurrentProxy != "door" && finalCfg.CurrentProxy != "nonelane" {
		t.Fatalf("unexpected CurrentProxy value %q", finalCfg.CurrentProxy)
	}
}
