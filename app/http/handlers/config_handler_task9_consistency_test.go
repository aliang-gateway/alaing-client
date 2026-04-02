package handlers

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"aliang.one/nursorgate/processor/config"
)

func TestConfigStatusConsistency(t *testing.T) {
	config.ResetRoutingApplyStoreForTest()
	h := NewConfigHandler()
	rh := NewRulesHandler()

	payload := []byte(`{
		"version": 1,
		"ingress": {"mode": "tun"},
		"egress": {
			"direct": {"enabled": true},
			"toAliang": {"enabled": false},
			"toSocks": {"enabled": true, "upstream": {"type": "socks"}}
		},
		"routing": {
			"rules": [
				{"id":"aliang-a","type":"domain","condition":"aliang.example","enabled":true,"target":"direct"},
				{"id":"socks-a","type":"domain","condition":"socks.example","enabled":true,"target":"toSocks"}
			]
		}
	}`)

	postReq := httptest.NewRequest(http.MethodPost, "/api/config/routing", bytes.NewReader(payload))
	postRec := httptest.NewRecorder()
	h.HandleRoutingConfig(postRec, postReq)
	if postRec.Code != http.StatusOK {
		t.Fatalf("expected post 200, got %d body=%s", postRec.Code, postRec.Body.String())
	}

	getCfgReq := httptest.NewRequest(http.MethodGet, "/api/config/routing", nil)
	getCfgRec := httptest.NewRecorder()
	h.HandleRoutingConfig(getCfgRec, getCfgReq)
	if getCfgRec.Code != http.StatusOK {
		t.Fatalf("expected get config 200, got %d body=%s", getCfgRec.Code, getCfgRec.Body.String())
	}

	cfgData := decodeCommonResponseData(t, getCfgRec)
	egress, ok := cfgData["egress"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected canonical egress object, got %#v", cfgData["egress"])
	}
	toAliang, ok := egress["toAliang"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected canonical toAliang object, got %#v", egress["toAliang"])
	}
	if toAliang["enabled"] != false {
		t.Fatalf("expected toAliang.enabled=false, got %#v", toAliang["enabled"])
	}
	toSocks, ok := egress["toSocks"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected canonical toSocks object, got %#v", egress["toSocks"])
	}
	if toSocks["enabled"] != true {
		t.Fatalf("expected toSocks.enabled=true, got %#v", toSocks["enabled"])
	}

	getSwitchReq := httptest.NewRequest(http.MethodGet, "/api/rules/switches/status", nil)
	getSwitchRec := httptest.NewRecorder()
	rh.HandleGetGlobalSwitchStatus(getSwitchRec, getSwitchReq)
	if getSwitchRec.Code != http.StatusOK {
		t.Fatalf("expected switch status 200, got %d body=%s", getSwitchRec.Code, getSwitchRec.Body.String())
	}

	switchData := decodeCommonResponseData(t, getSwitchRec)
	if switchData["aliangEnabled"] != false {
		t.Fatalf("expected aliangEnabled=false, got %#v", switchData["aliangEnabled"])
	}
	if switchData["socksEnabled"] != true {
		t.Fatalf("expected socksEnabled=true, got %#v", switchData["socksEnabled"])
	}

	engineReq := httptest.NewRequest(http.MethodGet, "/api/rules/engine/status", nil)
	engineRec := httptest.NewRecorder()
	rh.HandleGetRuleEngineStatus(engineRec, engineReq)
	if engineRec.Code != http.StatusOK {
		t.Fatalf("expected engine status 200, got %d body=%s", engineRec.Code, engineRec.Body.String())
	}

	engineData := decodeCommonResponseData(t, engineRec)
	if engineData["aliangEnabled"] != false {
		t.Fatalf("expected engine aliangEnabled=false, got %#v", engineData["aliangEnabled"])
	}
	if engineData["socksEnabled"] != true {
		t.Fatalf("expected engine socksEnabled=true, got %#v", engineData["socksEnabled"])
	}

	canonical := config.GetRoutingApplyStore().ActiveCanonicalSchema()
	if canonical == nil {
		t.Fatal("expected active canonical schema")
	}
	if canonical.Egress.ToAliang.Enabled {
		t.Fatal("expected canonical egress.toAliang.enabled=false")
	}
}

func TestRejectLegacyStoreWrite(t *testing.T) {
	config.ResetRoutingApplyStoreForTest()
	h := NewConfigHandler()
	rh := NewRulesHandler()

	payload := []byte(`{
		"version": 1,
		"ingress": {"mode": "tun"},
		"egress": {
			"direct": {"enabled": true},
			"toAliang": {"enabled": true},
			"toSocks": {"enabled": true, "upstream": {"type": "socks"}}
		},
		"routing": {"rules": []}
	}`)

	seedReq := httptest.NewRequest(http.MethodPost, "/api/config/routing", bytes.NewReader(payload))
	seedRec := httptest.NewRecorder()
	h.HandleRoutingConfig(seedRec, seedReq)
	if seedRec.Code != http.StatusOK {
		t.Fatalf("expected seed 200, got %d body=%s", seedRec.Code, seedRec.Body.String())
	}

	_, beforeVersion, beforeHash := config.GetRoutingApplyStore().ActiveSnapshotVersionHash()

	legacySwitchReq := httptest.NewRequest(http.MethodPost, "/api/rules/switches/aliang", bytes.NewReader([]byte(`{"enabled":false}`)))
	legacySwitchRec := httptest.NewRecorder()
	rh.HandleSetGlobalSwitch(legacySwitchRec, legacySwitchReq)
	if legacySwitchRec.Code != http.StatusBadRequest {
		t.Fatalf("expected legacy switch write 400, got %d body=%s", legacySwitchRec.Code, legacySwitchRec.Body.String())
	}
	legacySwitchData := decodeCommonResponseData(t, legacySwitchRec)
	legacySwitchMsg, _ := legacySwitchData["error_msg"].(string)
	if legacySwitchMsg != "Legacy mutable switch write path is removed; update canonical routing config via /api/config/routing" {
		t.Fatalf("unexpected switch rejection message: %q", legacySwitchMsg)
	}

	legacyRuleReq := httptest.NewRequest(http.MethodPut, "/api/config/routing/rules/rule-x/status", bytes.NewReader([]byte(`{"enabled":false}`)))
	legacyRuleRec := httptest.NewRecorder()
	h.HandleToggleRuleStatus(legacyRuleRec, legacyRuleReq)
	if legacyRuleRec.Code != http.StatusBadRequest {
		t.Fatalf("expected legacy rule write 400, got %d body=%s", legacyRuleRec.Code, legacyRuleRec.Body.String())
	}
	legacyRuleData := decodeCommonResponseData(t, legacyRuleRec)
	legacyRuleMsg, _ := legacyRuleData["error_msg"].(string)
	if legacyRuleMsg != "Legacy mutable rule toggle write path is removed; update canonical routing config via /api/config/routing" {
		t.Fatalf("unexpected rule rejection message: %q", legacyRuleMsg)
	}

	bulkReq := httptest.NewRequest(http.MethodPost, "/api/rules/switches/bulk", bytes.NewReader([]byte(`{"aliang_enabled":false,"socks_enabled":false}`)))
	bulkRec := httptest.NewRecorder()
	rh.HandleBulkSwitchControl(bulkRec, bulkReq)
	if bulkRec.Code != http.StatusBadRequest {
		t.Fatalf("expected bulk legacy write 400, got %d body=%s", bulkRec.Code, bulkRec.Body.String())
	}

	_, afterVersion, afterHash := config.GetRoutingApplyStore().ActiveSnapshotVersionHash()
	if afterVersion != beforeVersion {
		t.Fatalf("expected canonical version unchanged, got before=%d after=%d", beforeVersion, afterVersion)
	}
	if afterHash != beforeHash {
		t.Fatalf("expected canonical hash unchanged, got before=%q after=%q", beforeHash, afterHash)
	}

	currentCfgReq := httptest.NewRequest(http.MethodGet, "/api/config/routing", nil)
	currentCfgRec := httptest.NewRecorder()
	h.HandleRoutingConfig(currentCfgRec, currentCfgReq)
	if currentCfgRec.Code != http.StatusOK {
		t.Fatalf("expected current cfg 200, got %d body=%s", currentCfgRec.Code, currentCfgRec.Body.String())
	}

	cfgData := decodeCommonResponseData(t, currentCfgRec)
	egress, ok := cfgData["egress"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected canonical egress object, got %#v", cfgData["egress"])
	}
	toAliang, ok := egress["toAliang"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected canonical toAliang object, got %#v", egress["toAliang"])
	}
	toSocks, ok := egress["toSocks"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected canonical toSocks object, got %#v", egress["toSocks"])
	}
	if toAliang["enabled"] != true || toSocks["enabled"] != true {
		t.Fatalf("expected unchanged canonical switches true/true, got toAliang=%#v toSocks=%#v", toAliang["enabled"], toSocks["enabled"])
	}
}
