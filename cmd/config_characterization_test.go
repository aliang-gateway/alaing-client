package cmd

import (
	"os"
	"path/filepath"
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

func writeStartupTestConfigFile(t *testing.T, dir string, fileName string, content string) string {
	t.Helper()
	path := filepath.Join(dir, fileName)
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("failed to write test config file %s: %v", path, err)
	}
	return path
}

func TestApplyStartupConfig_UsesExplicitConfigPathOverLocalConfigNewJSON(t *testing.T) {
	resetConfigPipelineStateForTest(t)

	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(originalWd)
	})

	tempDir := t.TempDir()
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("failed to change working directory: %v", err)
	}

	writeStartupTestConfigFile(t, tempDir, "config.new.json", `{"api_server":`)
	explicitPath := writeStartupTestConfigFile(t, tempDir, "explicit.json", `{
		"api_server": "https://api.explicit.example.com",
		"currentProxy": "direct",
		"baseProxies": {
			"nonelane": {
				"type": "nonelane",
				"core_server": "explicit-gateway.example.com:443"
			}
		}
	}`)

	if err := ApplyStartupConfig(explicitPath); err != nil {
		t.Fatalf("expected explicit --config path to be used even when config.new.json exists, got: %v", err)
	}

	globalCfg := processorconfig.GetGlobalConfig()
	if globalCfg == nil {
		t.Fatalf("expected global config to be set")
	}
	if globalCfg.APIServer != "https://api.explicit.example.com" {
		t.Fatalf("expected explicit config APIServer, got %q", globalCfg.APIServer)
	}
	if IsUsingDefaultConfig() {
		t.Fatalf("expected custom config flag for explicit path")
	}
}

func TestApplyStartupConfig_UsesConfigNewJSONWhenNoExplicitPath(t *testing.T) {
	resetConfigPipelineStateForTest(t)

	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(originalWd)
	})

	tempDir := t.TempDir()
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("failed to change working directory: %v", err)
	}

	writeStartupTestConfigFile(t, tempDir, "config.new.json", `{
		"api_server": "https://api.local-config-new.example.com",
		"currentProxy": "direct",
		"baseProxies": {
			"nonelane": {
				"type": "nonelane",
				"core_server": "local-gateway.example.com:443"
			}
		}
	}`)

	if err := ApplyStartupConfig(""); err != nil {
		t.Fatalf("expected config.new.json to be used when no explicit path, got: %v", err)
	}

	globalCfg := processorconfig.GetGlobalConfig()
	if globalCfg == nil {
		t.Fatalf("expected global config to be set")
	}
	if globalCfg.APIServer != "https://api.local-config-new.example.com" {
		t.Fatalf("expected config.new.json APIServer, got %q", globalCfg.APIServer)
	}
	if IsUsingDefaultConfig() {
		t.Fatalf("expected custom config flag when loading config.new.json")
	}
}

func TestApplyStartupConfig_UsesEmbeddedDefaultWhenNoExplicitAndNoConfigNewJSON(t *testing.T) {
	resetConfigPipelineStateForTest(t)

	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(originalWd)
	})

	tempDir := t.TempDir()
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("failed to change working directory: %v", err)
	}

	if err := ApplyStartupConfig(""); err != nil {
		t.Fatalf("expected embedded default config to be used, got: %v", err)
	}

	globalCfg := processorconfig.GetGlobalConfig()
	if globalCfg == nil {
		t.Fatalf("expected global config to be set")
	}
	if !IsUsingDefaultConfig() {
		t.Fatalf("expected embedded default to mark using-default flag true")
	}
}

func TestApplyStartupConfig_FailsFastWhenConfigNewJSONExistsButInvalid(t *testing.T) {
	resetConfigPipelineStateForTest(t)

	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(originalWd)
	})

	tempDir := t.TempDir()
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("failed to change working directory: %v", err)
	}

	writeStartupTestConfigFile(t, tempDir, "config.new.json", `{"api_server":`)

	err = ApplyStartupConfig("")
	if err == nil {
		t.Fatalf("expected fail-fast error when config.new.json exists but is invalid")
	}
	if processorconfig.GetGlobalConfig() != nil {
		t.Fatalf("expected global config to remain unset on fail-fast invalid config.new.json")
	}
}
