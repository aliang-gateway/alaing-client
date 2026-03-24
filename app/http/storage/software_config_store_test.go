package storage

import (
	"encoding/json"
	"testing"
	"time"

	"nursor.org/nursorgate/app/http/models"
)

func TestSoftwareConfigStore_ActivateAndMergeByLatest(t *testing.T) {
	store, err := NewSoftwareConfigStoreWithDBPath(t.TempDir() + "/configs.db")
	if err != nil {
		t.Fatalf("create store failed: %v", err)
	}

	baseTime := time.Now().Add(-10 * time.Minute)
	first := models.SoftwareConfig{
		UUID:      "cfg-1",
		Software:  "opencode",
		Name:      "config-one",
		FilePath:  "/tmp/a.json",
		Version:   "v1",
		InUse:     false,
		Format:    models.ConfigFormatJSON,
		Content:   `{"a":1}`,
		CreatedAt: baseTime,
		UpdatedAt: baseTime,
	}
	second := models.SoftwareConfig{
		UUID:      "cfg-2",
		Software:  "opencode",
		Name:      "config-two",
		FilePath:  "/tmp/b.yaml",
		Version:   "v1",
		InUse:     false,
		Format:    models.ConfigFormatYAML,
		Content:   "a: 1",
		CreatedAt: baseTime,
		UpdatedAt: baseTime,
	}

	if err := store.Upsert(first); err != nil {
		t.Fatalf("upsert first failed: %v", err)
	}
	if err := store.Upsert(second); err != nil {
		t.Fatalf("upsert second failed: %v", err)
	}

	if err := store.Activate(first); err != nil {
		t.Fatalf("activate first failed: %v", err)
	}
	if err := store.Activate(second); err != nil {
		t.Fatalf("activate second failed: %v", err)
	}

	list, err := store.List()
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}

	inUseCount := 0
	activeUUID := ""
	for _, cfg := range list {
		if cfg.InUse {
			inUseCount++
			activeUUID = cfg.UUID
		}
	}
	if inUseCount != 1 || activeUUID != "cfg-2" {
		t.Fatalf("expected cfg-2 active only, got inUseCount=%d active=%s", inUseCount, activeUUID)
	}

	remoteOld := models.SoftwareConfig{
		UUID:      "cfg-2",
		Software:  "opencode",
		Name:      "config-two",
		FilePath:  "/tmp/b.yaml",
		Version:   "v-old",
		InUse:     true,
		Format:    models.ConfigFormatYAML,
		Content:   "a: old",
		CreatedAt: baseTime,
		UpdatedAt: baseTime.Add(-1 * time.Minute),
	}
	remoteNew := models.SoftwareConfig{
		UUID:      "cfg-3",
		Software:  "opencode",
		Name:      "config-three",
		FilePath:  "/tmp/c.json",
		Version:   "v1",
		InUse:     false,
		Format:    models.ConfigFormatJSON,
		Content:   `{"c":1}`,
		CreatedAt: baseTime,
		UpdatedAt: baseTime.Add(5 * time.Minute),
	}

	inserted, updated, kept, err := store.MergeByLatest([]models.SoftwareConfig{remoteOld, remoteNew})
	if err != nil {
		t.Fatalf("merge failed: %v", err)
	}
	if inserted != 1 || updated != 0 || kept != 1 {
		t.Fatalf("unexpected merge result inserted=%d updated=%d kept=%d", inserted, updated, kept)
	}

	remoteActiveNewer := models.SoftwareConfig{
		UUID:      "cfg-3",
		Software:  "opencode",
		Name:      "config-three",
		FilePath:  "/tmp/c.json",
		Version:   "v2",
		InUse:     true,
		Format:    models.ConfigFormatJSON,
		Content:   `{"c":2}`,
		CreatedAt: baseTime,
		UpdatedAt: baseTime.Add(20 * time.Minute),
	}
	if _, _, _, err := store.MergeByLatest([]models.SoftwareConfig{remoteActiveNewer}); err != nil {
		t.Fatalf("merge active newer failed: %v", err)
	}
	list2, err := store.List()
	if err != nil {
		t.Fatalf("list2 failed: %v", err)
	}
	inUseCount2 := 0
	activeUUID2 := ""
	for _, cfg := range list2 {
		if cfg.InUse {
			inUseCount2++
			activeUUID2 = cfg.UUID
		}
	}
	if inUseCount2 != 1 || activeUUID2 != "cfg-3" {
		t.Fatalf("expected cfg-3 active only after merge, got inUseCount=%d active=%s", inUseCount2, activeUUID2)
	}

	if err := store.SetSelected("cfg-3", true); err != nil {
		t.Fatalf("set selected failed: %v", err)
	}
	selected, err := store.ListSelectedBySoftware("opencode")
	if err != nil {
		t.Fatalf("list selected failed: %v", err)
	}
	if len(selected) != 1 || selected[0].UUID != "cfg-3" {
		t.Fatalf("expected cfg-3 selected, got %+v", selected)
	}

	logsErr := store.SaveOperationLog(models.SoftwareConfigOperationLog{
		Action:     "copy",
		Software:   "opencode",
		ConfigUUID: "cfg-3",
		ConfigName: "config-three",
		Detail:     "copied",
	})
	if logsErr != nil {
		t.Fatalf("save operation log failed: %v", logsErr)
	}

	listedByUUIDs, err := store.ListByUUIDs([]string{"cfg-2", "cfg-3"})
	if err != nil {
		t.Fatalf("list by uuids failed: %v", err)
	}
	if len(listedByUUIDs) != 2 {
		t.Fatalf("expected 2 configs by uuids, got %d", len(listedByUUIDs))
	}

	if err := store.DeleteByUUID("cfg-2"); err != nil {
		t.Fatalf("delete by uuid failed: %v", err)
	}
	if _, found, err := store.FindByUUID("cfg-2"); err != nil {
		t.Fatalf("find deleted cfg-2 failed: %v", err)
	} else if found {
		t.Fatal("expected cfg-2 deleted")
	}

	if err := store.SaveEffectiveConfigSnapshot(models.SoftwareEffectiveConfigSnapshot{
		Software:       "opencode",
		ConfigUUID:     "cfg-3",
		ConfigName:     "config-three",
		ConfigFilePath: "/tmp/c.json",
		ConfigVersion:  "v2",
		ConfigFormat:   models.ConfigFormatJSON,
		SnapshotJSON:   `{"core":{"aliangServer":{"type":"vmess","core_server":"ai-gateway.nursor.org:443"}},"customer":{"proxy":{"type":"http"}},"currentProxy":"direct"}`,
	}); err != nil {
		t.Fatalf("save effective snapshot failed: %v", err)
	}

	latest, err := store.GetLatestEffectiveConfigSnapshot()
	if err != nil {
		t.Fatalf("get latest effective snapshot failed: %v", err)
	}
	if latest.ConfigUUID != "cfg-3" || latest.Software != "opencode" {
		t.Fatalf("unexpected latest effective snapshot metadata: %+v", latest)
	}
	var latestPayload map[string]interface{}
	if err := json.Unmarshal([]byte(latest.SnapshotJSON), &latestPayload); err != nil {
		t.Fatalf("latest snapshot json invalid: %v", err)
	}
	if _, ok := latestPayload["core"]; !ok {
		t.Fatalf("expected latest snapshot core section, got %v", latestPayload)
	}
}
