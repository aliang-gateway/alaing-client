package services

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"aliang.one/nursorgate/processor/config"
)

func TestMergeCustomerPayload_PreservesOmittedFields(t *testing.T) {
	baseCfg := &config.Config{
		Core: &config.CoreConfig{APIServer: "https://api.example.com"},
		Customer: &config.CustomerConfig{
			Proxy: &config.CustomerProxyConfig{
				Type:   "http",
				Server: "127.0.0.1:8080",
			},
			AIRules: map[string]*config.CustomerAIRuleSetting{
				"openai": {
					Enble:   customerBoolPtr(true),
					Include: []string{"api.openai.com"},
				},
			},
			ProxyRules: []string{"domain,example.com,proxy"},
		},
	}

	mergedRaw, nextCfg, err := mergeCustomerPayload(baseCfg, []byte(`{
		"proxy":{"type":"socks5"},
		"ai_rules":{"openai":{"exclude":["chatgpt.com"]},"claude":{"enble":true}}
	}`))
	if err != nil {
		t.Fatalf("mergeCustomerPayload() error = %v", err)
	}

	if !strings.Contains(mergedRaw, `"server":"127.0.0.1:8080"`) {
		t.Fatalf("merged payload should preserve proxy.server, got %s", mergedRaw)
	}
	if nextCfg.Customer == nil || nextCfg.Customer.Proxy == nil {
		t.Fatalf("merged config missing customer.proxy: %+v", nextCfg.Customer)
	}
	if got := nextCfg.Customer.Proxy.Server; got != "127.0.0.1:8080" {
		t.Fatalf("proxy.server = %q, want preserved value", got)
	}
	if got := nextCfg.Customer.Proxy.Type; got != "socks5" {
		t.Fatalf("proxy.type = %q, want socks5", got)
	}
	if len(nextCfg.Customer.ProxyRules) != 1 {
		t.Fatalf("proxy_rules should remain unchanged, got %+v", nextCfg.Customer.ProxyRules)
	}

	openai := nextCfg.Customer.AIRules["openai"]
	if openai == nil {
		t.Fatalf("openai rule missing from merged config: %+v", nextCfg.Customer.AIRules)
	}
	if openai.Enble == nil || !*openai.Enble {
		t.Fatalf("openai.enble should be preserved as true, got %+v", openai.Enble)
	}
	if len(openai.Include) != 1 || openai.Include[0] != "api.openai.com" {
		t.Fatalf("openai.include = %v, want [api.openai.com]", openai.Include)
	}

	claude := nextCfg.Customer.AIRules["claude"]
	if claude == nil || claude.Enble == nil || !*claude.Enble {
		t.Fatalf("claude.enble should be added as true, got %+v", claude)
	}
}

func TestResolveBaseConfigForCustomerUpdate_PrefersStartupLocalConfigOverHomeConfig(t *testing.T) {
	config.ResetGlobalConfigForTest()
	config.ResetEffectiveConfigCommitCoordinatorForTest()
	t.Cleanup(func() {
		config.ResetGlobalConfigForTest()
		config.ResetEffectiveConfigCommitCoordinatorForTest()
		_ = os.Remove(startupLocalCustomerBaseConfigPath)
	})

	tempHome := t.TempDir()
	if err := os.MkdirAll(filepath.Join(tempHome, ".aliang"), 0755); err != nil {
		t.Fatalf("mkdir temp home .aliang failed: %v", err)
	}
	t.Setenv("HOME", tempHome)

	if err := os.WriteFile(startupLocalCustomerBaseConfigPath, []byte(`{
		"core":{"api_server":"https://sub2api.liang.home"},
		"customer":{"proxy":{"type":"socks5","server":"127.0.0.1:1080"}}
	}`), 0644); err != nil {
		t.Fatalf("write startup local config failed: %v", err)
	}

	if err := os.WriteFile(filepath.Join(tempHome, ".aliang", "config.json"), []byte(`{
		"core":{"api_server":"https://api.example.com"},
		"customer":{"proxy":{"type":"http","server":"127.0.0.1:1081"}}
	}`), 0644); err != nil {
		t.Fatalf("write home config failed: %v", err)
	}

	baseCfg, err := resolveBaseConfigForCustomerUpdate()
	if err != nil {
		t.Fatalf("resolveBaseConfigForCustomerUpdate() error = %v", err)
	}
	if got := baseCfg.APIBaseURL(); got != "https://sub2api.liang.home" {
		t.Fatalf("APIBaseURL = %q, want startup local config value", got)
	}
}

func customerBoolPtr(v bool) *bool {
	return &v
}
