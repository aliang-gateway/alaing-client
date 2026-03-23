package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"nursor.org/nursorgate/processor/config"
)

func decodeCommonResponseData(t *testing.T, rec *httptest.ResponseRecorder) map[string]interface{} {
	t.Helper()

	var raw map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &raw); err != nil {
		t.Fatalf("failed to decode response body: %v; body=%s", err, rec.Body.String())
	}
	data, ok := raw["data"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected response data object, got %#v", raw["data"])
	}
	return data
}

func TestLegacyReject_ConfigHandler_RoutingConfigLegacyAliasesRejectedViaCanonicalApply(t *testing.T) {
	h := NewConfigHandler()

	config.ResetRoutingApplyStoreForTest()

	legacyPayload := []byte(`{
		"none_lane": {
			"set_type": "none_lane",
			"rules": []
		},
		"to_door": {
			"set_type": "to_door",
			"rules": []
		},
		"black_list": {
			"set_type": "black_list",
			"rules": []
		},
		"aliang": {
			"set_type": "aliang",
			"rules": []
		},
		"settings": {
			"aliang_enabled": true,
			"socks_enabled": true,
			"geoip_enabled": false,
			"auto_update": true
		},
		"version": 1,
		"created_at": "2026-01-01T00:00:00Z",
		"updated_at": "2026-01-01T00:00:00Z"
	}`)

	req := httptest.NewRequest(http.MethodPost, "/api/config/routing", bytes.NewReader(legacyPayload))
	rec := httptest.NewRecorder()
	h.HandleRoutingConfig(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for legacy aliases payload, got %d body=%s", rec.Code, rec.Body.String())
	}

	data := decodeCommonResponseData(t, rec)
	errMsg, _ := data["error_msg"].(string)
	if errMsg != "Configuration validation failed" {
		t.Fatalf("expected deterministic validation failure message, got %q", errMsg)
	}
	details, _ := data["details"].(map[string]interface{})
	detailError, _ := details["error"].(string)
	if !strings.Contains(detailError, "non-canonical routing payload is not supported") {
		t.Fatalf("expected migration guidance in error details, got %q", detailError)
	}
}

func TestCanonicalOnly_ConfigHandler_CanonicalPayloadAcceptedAndLegacyToggleWriteRejected(t *testing.T) {
	h := NewConfigHandler()

	config.ResetRoutingApplyStoreForTest()

	canonicalPayload := []byte(`{
		"version": 1,
		"ingress": {"mode": "tun"},
		"egress": {
			"direct": {"enabled": true},
			"toAliang": {"enabled": true},
			"toSocks": {"enabled": true, "upstream": {"type": "socks"}}
		},
		"routing": {
			"rules": [
				{"id": "socks-1", "type": "domain", "condition": "s.example.com", "enabled": true, "target": "toSocks"},
				{"id": "aliang-1", "type": "domain", "condition": "a.example.com", "enabled": true, "target": "toAliang"}
			]
		}
	}`)

	updateReq := httptest.NewRequest(http.MethodPost, "/api/config/routing", bytes.NewReader(canonicalPayload))
	updateRec := httptest.NewRecorder()
	h.HandleRoutingConfig(updateRec, updateReq)
	if updateRec.Code != http.StatusOK {
		t.Fatalf("expected canonical payload to succeed, got %d body=%s", updateRec.Code, updateRec.Body.String())
	}

	toggleReq := httptest.NewRequest(http.MethodPut, "/api/config/routing/rules/to_door/status", bytes.NewReader([]byte(`{"enabled":false}`)))
	toggleRec := httptest.NewRecorder()
	h.HandleToggleRuleStatus(toggleRec, toggleReq)

	if toggleRec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 when toggling legacy mutable path, got %d body=%s", toggleRec.Code, toggleRec.Body.String())
	}
	data := decodeCommonResponseData(t, toggleRec)
	errMsg, _ := data["error_msg"].(string)
	if errMsg != "Legacy mutable rule toggle write path is removed; update canonical routing config via /api/config/routing" {
		t.Fatalf("expected legacy write rejection message, got %q", errMsg)
	}
}

func TestLegacyReject_ConfigHandler_RoutingBoundary(t *testing.T) {
	h := NewConfigHandler()

	t.Run("legacy alias payload rejected through canonical apply", func(t *testing.T) {
		config.ResetRoutingApplyStoreForTest()

		legacyPayload := []byte(`{
			"none_lane": {
				"set_type": "none_lane",
				"rules": []
			},
			"to_door": {
				"set_type": "to_door",
				"rules": []
			},
			"black_list": {
				"set_type": "black_list",
				"rules": []
			},
			"aliang": {
				"set_type": "aliang",
				"rules": []
			},
			"settings": {
				"aliang_enabled": true,
				"socks_enabled": true,
				"geoip_enabled": false,
				"auto_update": true
			},
			"version": 1,
			"created_at": "2026-01-01T00:00:00Z",
			"updated_at": "2026-01-01T00:00:00Z"
		}`)

		req := httptest.NewRequest(http.MethodPost, "/api/config/routing", bytes.NewReader(legacyPayload))
		rec := httptest.NewRecorder()
		h.HandleRoutingConfig(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 for legacy aliases through canonical apply, got %d body=%s", rec.Code, rec.Body.String())
		}

		data := decodeCommonResponseData(t, rec)
		errMsg, _ := data["error_msg"].(string)
		if errMsg != "Configuration validation failed" {
			t.Fatalf("expected deterministic validation failure message, got %q", errMsg)
		}
		details, _ := data["details"].(map[string]interface{})
		detailError, _ := details["error"].(string)
		if !strings.Contains(detailError, "non-canonical routing payload is not supported") {
			t.Fatalf("expected migration guidance in error details, got %q", detailError)
		}
	})

	t.Run("legacy mutable toggle path fails closed with deterministic error", func(t *testing.T) {
		config.ResetRoutingApplyStoreForTest()

		canonicalPayload := []byte(`{
			"version": 1,
			"ingress": {"mode": "tun"},
			"egress": {
				"direct": {"enabled": true},
				"toAliang": {"enabled": true},
				"toSocks": {"enabled": true, "upstream": {"type": "socks"}}
			},
			"routing": {
				"rules": [
					{"id": "socks-1", "type": "domain", "condition": "s.example.com", "enabled": true, "target": "toSocks"},
					{"id": "aliang-1", "type": "domain", "condition": "a.example.com", "enabled": true, "target": "toAliang"}
				]
			}
		}`)

		updateReq := httptest.NewRequest(http.MethodPost, "/api/config/routing", bytes.NewReader(canonicalPayload))
		updateRec := httptest.NewRecorder()
		h.HandleRoutingConfig(updateRec, updateReq)
		if updateRec.Code != http.StatusOK {
			t.Fatalf("expected canonical payload to succeed, got %d body=%s", updateRec.Code, updateRec.Body.String())
		}

		toggleReq := httptest.NewRequest(http.MethodPut, "/api/config/routing/rules/to_door/status", bytes.NewReader([]byte(`{"enabled":false}`)))
		toggleRec := httptest.NewRecorder()
		h.HandleToggleRuleStatus(toggleRec, toggleReq)

		if toggleRec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 when toggling legacy mutable path, got %d body=%s", toggleRec.Code, toggleRec.Body.String())
		}

		data := decodeCommonResponseData(t, toggleRec)
		errMsg, _ := data["error_msg"].(string)
		if errMsg != "Legacy mutable rule toggle write path is removed; update canonical routing config via /api/config/routing" {
			t.Fatalf("expected legacy write rejection message, got %q", errMsg)
		}
	})
}

func TestCanonicalOnly_ConfigHandler_RoutingBoundary(t *testing.T) {
	h := NewConfigHandler()

	t.Run("canonical payload accepted through canonical apply", func(t *testing.T) {
		config.ResetRoutingApplyStoreForTest()

		canonicalPayload := []byte(`{
			"version": 1,
			"ingress": {"mode": "tun"},
			"egress": {
				"direct": {"enabled": true},
				"toAliang": {"enabled": true},
				"toSocks": {"enabled": true, "upstream": {"type": "socks"}}
			},
			"routing": {
				"default_egress": "direct",
				"rules": [
					{"id": "socks-1", "type": "domain", "condition": "s.example.com", "enabled": true, "target": "toSocks"},
					{"id": "aliang-1", "type": "domain", "condition": "a.example.com", "enabled": true, "target": "toAliang"}
				]
			}
		}`)

		req := httptest.NewRequest(http.MethodPost, "/api/config/routing", bytes.NewReader(canonicalPayload))
		rec := httptest.NewRecorder()
		h.HandleRoutingConfig(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200 for canonical payload, got %d body=%s", rec.Code, rec.Body.String())
		}

		data := decodeCommonResponseData(t, rec)
		source, _ := data["source"].(string)
		if source != "canonical" {
			t.Fatalf("expected source canonical, got %q", source)
		}
		if _, ok := data["version"].(float64); !ok {
			t.Fatalf("expected version in success payload, got %#v", data["version"])
		}
	})
}
