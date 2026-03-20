package services

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"nursor.org/nursorgate/app/http/models"
	"nursor.org/nursorgate/app/http/storage"
)

func TestSoftwareConfigService_SaveActivateAndCloudSync(t *testing.T) {
	store, err := storage.NewSoftwareConfigStoreWithDBPath(t.TempDir() + "/service.db")
	if err != nil {
		t.Fatalf("create store failed: %v", err)
	}

	service := NewSoftwareConfigServiceWithStore(store, nil)

	saved, err := service.Save(models.SaveSoftwareConfigRequest{
		UUID:     "svc-1",
		Software: "opencode",
		Name:     "svc-config",
		FilePath: filepath.Join(t.TempDir(), "svc.json"),
		Version:  "v1",
		Format:   models.ConfigFormatJSON,
		Content:  `{"k":1}`,
	})
	if err != nil {
		t.Fatalf("save failed: %v", err)
	}
	if saved.UUID == "" {
		t.Fatal("expected uuid")
	}

	activatePath := filepath.Join(t.TempDir(), "active.yaml")
	active, err := service.Activate(models.ActivateSoftwareConfigRequest{
		UUID:     "svc-2",
		Software: "claude",
		Name:     "active-config",
		FilePath: activatePath,
		Version:  "v2",
		Format:   models.ConfigFormatYAML,
		Content:  "a: 1\n",
	})
	if err != nil {
		t.Fatalf("activate failed: %v", err)
	}
	if !active.InUse {
		t.Fatal("expected active in_use=true")
	}
	opencodeActive, err := service.Activate(models.ActivateSoftwareConfigRequest{
		UUID:     "svc-1",
		Software: "opencode",
		Name:     "svc-config",
		FilePath: saved.FilePath,
		Version:  "v1",
		Format:   models.ConfigFormatJSON,
		Content:  `{"k":1}`,
	})
	if err != nil {
		t.Fatalf("activate opencode failed: %v", err)
	}
	if !opencodeActive.InUse {
		t.Fatal("expected opencode active in_use=true")
	}

	claudeOnlyBeforePull, err := service.ListBySoftware("claude")
	if err != nil {
		t.Fatalf("list claude before pull failed: %v", err)
	}
	if len(claudeOnlyBeforePull) != 1 || !claudeOnlyBeforePull[0].InUse {
		t.Fatalf("expected one active claude config before pull, got: %+v", claudeOnlyBeforePull)
	}
	fileContent, err := os.ReadFile(activatePath)
	if err != nil {
		t.Fatalf("read active file failed: %v", err)
	}
	if string(fileContent) != "a: 1\n" {
		t.Fatalf("unexpected active file content: %s", string(fileContent))
	}

	pushServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST for push, got %s", r.Method)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer pushServer.Close()

	pushResp, err := service.PushToCloud(models.CloudPushRequest{CloudURL: pushServer.URL})
	if err != nil {
		t.Fatalf("push failed: %v", err)
	}
	if pushResp.PushedCount < 2 {
		t.Fatalf("expected pushed count >= 2, got %d", pushResp.PushedCount)
	}

	remoteNewerTime := time.Now().Add(2 * time.Minute).UTC().Format(time.RFC3339)
	remoteOlderTime := time.Now().Add(-2 * time.Minute).UTC().Format(time.RFC3339)
	pullServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("expected GET for pull, got %s", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"configs":[` +
			`{"uuid":"svc-2","software":"claude","name":"active-config","file_path":"` + activatePath + `","version":"v3","in_use":true,"format":"yaml","content":"a: 2\\n","updated_at":"` + remoteNewerTime + `"},` +
			`{"uuid":"svc-1","software":"opencode","name":"svc-config","file_path":"` + saved.FilePath + `","version":"v0","in_use":false,"format":"json","content":"{\"k\":0}","updated_at":"` + remoteOlderTime + `"},` +
			`{"uuid":"svc-3","software":"openai","name":"new-config","file_path":"` + filepath.Join(t.TempDir(), "new.json") + `","version":"v1","in_use":false,"format":"json","content":"{\"n\":1}","updated_at":"` + remoteNewerTime + `"}` +
			`]}`))
	}))
	defer pullServer.Close()

	pullResp, err := service.PullFromCloud(models.CloudPullRequest{CloudURL: pullServer.URL})
	if err != nil {
		t.Fatalf("pull failed: %v", err)
	}
	if pullResp.PulledCount != 3 || pullResp.InsertedCount != 1 {
		t.Fatalf("unexpected pull counters: %+v", pullResp)
	}

	claudeOnly, err := service.ListBySoftware("claude")
	if err != nil {
		t.Fatalf("list by software failed: %v", err)
	}
	if len(claudeOnly) != 1 || claudeOnly[0].Software != "claude" {
		t.Fatalf("expected one claude config, got: %+v", claudeOnly)
	}
	if !claudeOnly[0].InUse {
		t.Fatalf("expected claude config remain active after pull, got: %+v", claudeOnly[0])
	}

	opencodeOnly, err := service.ListBySoftware("opencode")
	if err != nil {
		t.Fatalf("list opencode failed: %v", err)
	}
	activeCnt := 0
	for i := range opencodeOnly {
		if opencodeOnly[i].InUse {
			activeCnt++
		}
	}
	if activeCnt != 1 {
		t.Fatalf("expected exactly one active opencode config, got %d from %+v", activeCnt, opencodeOnly)
	}

	if _, err := service.Save(models.SaveSoftwareConfigRequest{
		UUID:      "svc-1",
		Software:  "opencode",
		Name:      "svc-config",
		FilePath:  saved.FilePath,
		Version:   "v2",
		Format:    models.ConfigFormatJSON,
		Content:   `{"k":2}`,
		UpdatedAt: time.Now().UTC().Format(time.RFC3339),
	}); err != nil {
		t.Fatalf("save existing failed: %v", err)
	}

	list, err := service.List()
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}
	var updatedSvc1 *models.SoftwareConfig
	for i := range list {
		if list[i].UUID == "svc-1" {
			updatedSvc1 = &list[i]
			break
		}
	}
	if updatedSvc1 == nil {
		t.Fatal("svc-1 should exist")
	}
	if updatedSvc1.CreatedAt.IsZero() {
		t.Fatal("created_at should be preserved and non-zero")
	}

	if _, err := service.Save(models.SaveSoftwareConfigRequest{
		Software:  "opencode",
		Name:      "bad-time",
		FilePath:  filepath.Join(t.TempDir(), "bad.json"),
		Version:   "v1",
		Format:    models.ConfigFormatJSON,
		Content:   `{"a":1}`,
		UpdatedAt: "not-rfc3339",
	}); err == nil {
		t.Fatal("expected error for invalid updated_at")
	}
}
