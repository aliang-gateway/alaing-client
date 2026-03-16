package cmd

import (
	"os"
	"testing"

	processorconfig "nursor.org/nursorgate/processor/config"
)

func resetConfigPipelineStateForTest(t *testing.T) {
	t.Helper()
	processorconfig.ResetGlobalConfigForTest()
	setUseDefaultConfig(false)
	t.Cleanup(func() {
		processorconfig.ResetGlobalConfigForTest()
		setUseDefaultConfig(false)
	})
}

func TestLoadAndApplyConfig_WhenFileMissing_ReturnsError(t *testing.T) {
	resetConfigPipelineStateForTest(t)

	tempDir := t.TempDir()
	tempFile, err := os.CreateTemp(tempDir, "config-*.json")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	missingPath := tempFile.Name()
	_ = tempFile.Close()
	if err := os.Remove(missingPath); err != nil {
		t.Fatalf("failed to remove temp file for missing-path test: %v", err)
	}

	err = LoadAndApplyConfig(missingPath)
	if err == nil {
		t.Fatalf("expected error for missing config file, got nil")
	}
}

func TestApplyDefaultConfig_SetsUsingDefaultConfigTrue(t *testing.T) {
	resetConfigPipelineStateForTest(t)

	if IsUsingDefaultConfig() {
		t.Fatalf("expected default config flag to start false")
	}

	if err := ApplyDefaultConfig(); err != nil {
		t.Fatalf("expected ApplyDefaultConfig to succeed, got: %v", err)
	}

	if !IsUsingDefaultConfig() {
		t.Fatalf("expected IsUsingDefaultConfig() to be true after ApplyDefaultConfig")
	}
}

func TestLoadConfigFromBytes_WithInvalidJSON_ReturnsError(t *testing.T) {
	resetConfigPipelineStateForTest(t)

	invalidJSON := []byte(`{"api_server":`)

	cfg, err := LoadConfigFromBytes(invalidJSON)
	if err == nil {
		t.Fatalf("expected error for invalid JSON, got nil")
	}
	if cfg != nil {
		t.Fatalf("expected nil config on invalid JSON, got %#v", cfg)
	}
}

func TestLoadConfigFromBytes_WithValidJSON_ReturnsConfig(t *testing.T) {
	resetConfigPipelineStateForTest(t)

	validJSON := []byte(`{
		"api_server": "https://api.example.com",
		"currentProxy": "direct",
		"baseProxies": {
			"nonelane": {
				"type": "nonelane",
				"core_server": "gateway.example.com:443"
			}
		}
	}`)

	cfg, err := LoadConfigFromBytes(validJSON)
	if err != nil {
		t.Fatalf("expected no error for valid JSON, got: %v", err)
	}
	if cfg == nil {
		t.Fatalf("expected parsed config, got nil")
	}
	if cfg.APIServer != "https://api.example.com" {
		t.Fatalf("expected APIServer to be parsed, got: %q", cfg.APIServer)
	}
	if cfg.CurrentProxy != "direct" {
		t.Fatalf("expected CurrentProxy to be parsed, got: %q", cfg.CurrentProxy)
	}
	if cfg.BaseProxies == nil {
		t.Fatalf("expected BaseProxies to be parsed, got nil")
	}
	nonelaneCfg, ok := cfg.BaseProxies["nonelane"]
	if !ok || nonelaneCfg == nil {
		t.Fatalf("expected nonelane base proxy to be present")
	}
	if nonelaneCfg.CoreServer != "gateway.example.com:443" {
		t.Fatalf("expected nonelane core_server to be parsed, got: %q", nonelaneCfg.CoreServer)
	}

}
