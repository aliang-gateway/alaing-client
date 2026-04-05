package services

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"aliang.one/nursorgate/app/http/models"
	auth "aliang.one/nursorgate/processor/auth"
	"aliang.one/nursorgate/processor/config"
)

func TestQuickSetupService_Catalog_Unauthenticated(t *testing.T) {
	previous := quickSetupGetAPIKeysFn
	quickSetupGetAPIKeysFn = func() ([]auth.UserAPIKey, error) {
		return nil, fmt.Errorf("no user session")
	}
	t.Cleanup(func() {
		quickSetupGetAPIKeysFn = previous
	})

	svc := NewQuickSetupService()
	result := svc.Catalog()
	if result["status"] != "unauthenticated" {
		t.Fatalf("expected unauthenticated status, got %#v", result["status"])
	}
}

func TestQuickSetupService_Render_MultiProvider(t *testing.T) {
	config.ResetGlobalConfigForTest()
	config.SetGlobalConfig(&config.Config{
		Core: &config.CoreConfig{APIServer: "https://api.example.com"},
	})
	t.Cleanup(config.ResetGlobalConfigForTest)

	previous := quickSetupGetAPIKeysFn
	quickSetupGetAPIKeysFn = func() ([]auth.UserAPIKey, error) {
		openaiGroupID := int64(1)
		anthropicGroupID := int64(2)
		return []auth.UserAPIKey{
			{
				ID:              11,
				Key:             "sk-openai-real",
				Name:            "OpenAI Key",
				GroupID:         &openaiGroupID,
				Status:          "active",
				Provider:        "openai",
				Masked:          false,
				SecretAvailable: true,
				Group: &auth.APIKeyGroup{
					ID:       1,
					Name:     "OpenAI Group",
					Platform: "openai",
				},
			},
			{
				ID:              22,
				Key:             "sk-ant-real",
				Name:            "Anthropic Key",
				GroupID:         &anthropicGroupID,
				Status:          "active",
				Provider:        "anthropic",
				Masked:          false,
				SecretAvailable: true,
				Group: &auth.APIKeyGroup{
					ID:       2,
					Name:     "Anthropic Group",
					Platform: "anthropic",
				},
			},
		}, nil
	}
	t.Cleanup(func() {
		quickSetupGetAPIKeysFn = previous
	})

	svc := NewQuickSetupService()
	resp, err := svc.Render(models.QuickSetupRenderRequest{Software: "codex"})
	if err != nil {
		t.Fatalf("render failed: %v", err)
	}
	if len(resp.Variants) != 2 {
		t.Fatalf("expected 2 variants, got %d", len(resp.Variants))
	}

	if !strings.Contains(resp.Variants[0].Files[0].Content, "model_provider") {
		t.Fatalf("expected codex config.toml content, got %q", resp.Variants[0].Files[0].Content)
	}
	if !strings.Contains(resp.Variants[0].Files[1].Content, "OPENAI_API_KEY") {
		t.Fatalf("expected auth.json content, got %q", resp.Variants[0].Files[1].Content)
	}
}

func TestQuickSetupService_Apply_WritesFiles(t *testing.T) {
	tempHome := t.TempDir()
	t.Setenv("HOME", tempHome)

	svc := NewQuickSetupService()
	resp, err := svc.Apply(models.QuickSetupApplyRequest{
		Software: "codex",
		Files: []models.QuickSetupApplyFile{
			{
				Path:    "~/.codex/config.toml",
				Content: "model = \"gpt-5-codex\"\n",
				Kind:    "file",
			},
			{
				Path:    "~/.codex/auth.json",
				Content: "{\"OPENAI_API_KEY\":\"sk-test\"}",
				Kind:    "file",
			},
		},
	})
	if err != nil {
		t.Fatalf("apply failed: %v", err)
	}
	if len(resp.Written) != 2 {
		t.Fatalf("expected 2 written files, got %d", len(resp.Written))
	}

	configPath := filepath.Join(tempHome, ".codex", "config.toml")
	content, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read written config failed: %v", err)
	}
	if !strings.Contains(string(content), "gpt-5-codex") {
		t.Fatalf("unexpected config content: %s", string(content))
	}
}
