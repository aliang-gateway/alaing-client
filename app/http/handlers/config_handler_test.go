package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"nursor.org/nursorgate/app/http/services"
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

func TestConfigHandler_CustomerConfigGetAndUpdate(t *testing.T) {
	config.ResetGlobalConfigForTest()
	config.ResetEffectiveConfigCommitCoordinatorForTest()
	t.Cleanup(func() {
		config.ResetGlobalConfigForTest()
		config.ResetEffectiveConfigCommitCoordinatorForTest()
	})

	config.SetGlobalConfig(&config.Config{
		Core: &config.CoreConfig{
			APIServer:    "https://api.example.com",
			AliangServer: &config.AliangServerConfig{Type: "aliang", CoreServer: "ai-gateway.nursor.org:443"},
		},
		Customer: &config.CustomerConfig{
			Proxy: &config.CustomerProxyConfig{Type: "http"},
			AIRules: map[string]*config.CustomerAIRuleSetting{
				"openai": {
					Enble:   boolPtr(true),
					Include: []string{"api.openai.com"},
				},
			},
		},
	})

	h := NewConfigHandler()
	h.customerConfigService = services.NewCustomerConfigService()

	getReq := httptest.NewRequest(http.MethodGet, "/api/config/customer", nil)
	getRec := httptest.NewRecorder()
	h.HandleCustomerConfig(getRec, getReq)
	if getRec.Code != http.StatusOK {
		t.Fatalf("initial get status=%d body=%s", getRec.Code, getRec.Body.String())
	}
	getData := decodeCommonResponseData(t, getRec)
	initialCustomer, _ := getData["customer"].(map[string]interface{})
	if initialCustomer == nil {
		t.Fatalf("expected customer object in get response: %v", getData)
	}

	payload := []byte(`{"proxy":{"type":"http"},"ai_rules":{"claude":{"enble":true,"exclude":["claude.ai"]}},"proxy_rules":["*.example.com"]}`)
	updateReq := httptest.NewRequest(http.MethodPost, "/api/config/customer", bytes.NewReader(payload))
	updateRec := httptest.NewRecorder()
	h.HandleCustomerConfig(updateRec, updateReq)
	if updateRec.Code != http.StatusOK {
		t.Fatalf("update status=%d body=%s", updateRec.Code, updateRec.Body.String())
	}
	updateData := decodeCommonResponseData(t, updateRec)
	updatedCustomer, _ := updateData["customer"].(map[string]interface{})
	if updatedCustomer == nil {
		t.Fatalf("expected customer object in update response: %v", updateData)
	}
	aiRules, _ := updatedCustomer["ai_rules"].(map[string]interface{})
	if _, ok := aiRules["claude"]; !ok {
		t.Fatalf("expected claude ai rule in updated customer config: %v", updatedCustomer)
	}

	coordinator := config.GetEffectiveConfigCommitCoordinator()
	if coordinator.Version() == 0 {
		t.Fatalf("expected coordinator version increment after update")
	}
	committed := coordinator.LastCommittedSnapshot()
	if committed == nil {
		t.Fatal("expected committed snapshot")
	}
	if committed.FilePath != "~/.aliang/config.json" {
		t.Fatalf("expected customer commit file path ~/.aliang/config.json, got %q", committed.FilePath)
	}
	if !strings.Contains(committed.Content, `"claude"`) {
		t.Fatalf("expected committed snapshot content to include updated customer rule: %s", committed.Content)
	}

	updatedCfg := config.GetGlobalConfig()
	if updatedCfg == nil || updatedCfg.Customer == nil {
		t.Fatal("expected global config customer to be updated")
	}
	if _, ok := updatedCfg.Customer.AIRules["claude"]; !ok {
		t.Fatalf("expected global config customer ai_rules to contain claude: %+v", updatedCfg.Customer)
	}

}

func TestConfigHandler_CustomerConfigRejectsForbiddenOrUnknownFields(t *testing.T) {
	config.ResetGlobalConfigForTest()
	config.ResetEffectiveConfigCommitCoordinatorForTest()
	t.Cleanup(func() {
		config.ResetGlobalConfigForTest()
		config.ResetEffectiveConfigCommitCoordinatorForTest()
	})

	config.SetGlobalConfig(&config.Config{
		Core: &config.CoreConfig{
			APIServer: "https://api.example.com",
		},
		Customer: &config.CustomerConfig{
			Proxy: &config.CustomerProxyConfig{Type: "http"},
		},
	})

	h := NewConfigHandler()
	h.customerConfigService = services.NewCustomerConfigService()

	cases := []struct {
		name         string
		method       string
		payload      string
		expectErrSub string
	}{
		{
			name:         "reject unknown customer field",
			method:       http.MethodPost,
			payload:      `{"proxy":{"type":"http"},"forbidden":true}`,
			expectErrSub: "customer.forbidden is forbidden",
		},
		{
			name:         "reject core field wrapper",
			method:       http.MethodPut,
			payload:      `{"customer":{"proxy":{"type":"http"}},"core":{"engine":{"mtu":1400}}}`,
			expectErrSub: "customer.core is forbidden",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			before := config.GetEffectiveConfigCommitCoordinator().Version()
			req := httptest.NewRequest(tc.method, "/api/config/customer", bytes.NewReader([]byte(tc.payload)))
			rec := httptest.NewRecorder()
			h.HandleCustomerConfig(rec, req)

			if rec.Code != http.StatusBadRequest {
				t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
			}
			data := decodeCommonResponseData(t, rec)
			errMsg, _ := data["error_msg"].(string)
			if errMsg != "Configuration validation failed" {
				t.Fatalf("expected deterministic validation failure message, got %q", errMsg)
			}
			details, _ := data["details"].(map[string]interface{})
			detailErr, _ := details["error"].(string)
			if !strings.Contains(detailErr, tc.expectErrSub) {
				t.Fatalf("expected error detail to contain %q, got %q", tc.expectErrSub, detailErr)
			}
			after := config.GetEffectiveConfigCommitCoordinator().Version()
			if after != before {
				t.Fatalf("expected no coordinator commit on validation failure: before=%d after=%d", before, after)
			}
		})
	}
}

func TestConfigHandler_CustomerConfigUpdateSucceedsWhenGlobalConfigNilAndFileExists(t *testing.T) {
	config.ResetGlobalConfigForTest()
	config.ResetEffectiveConfigCommitCoordinatorForTest()
	t.Cleanup(func() {
		config.ResetGlobalConfigForTest()
		config.ResetEffectiveConfigCommitCoordinatorForTest()
		_ = os.Remove("./config.new.json")
	})

	tempHome := t.TempDir()
	if err := os.MkdirAll(filepath.Join(tempHome, ".aliang"), 0755); err != nil {
		t.Fatalf("mkdir temp home .aliang failed: %v", err)
	}
	t.Setenv("HOME", tempHome)

	seed := []byte(`{
		"core":{"api_server":"https://sub2api.liang.home","aliangServer":{"type":"aliang","core_server":"ai-gateway.nursor.org:443"}},
		"customer":{
			"proxy":{"type":"http"},
			"ai_rules":{"openai":{"enble":true,"exclude":["api.openai.com"]}},
			"proxy_rules":[]
		}
	}`)
	if err := os.WriteFile("./config.new.json", seed, 0644); err != nil {
		t.Fatalf("seed config.new.json failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tempHome, ".aliang", "config.json"), []byte(`{
		"core":{"api_server":"https://api.example.com","aliangServer":{"type":"aliang","core_server":"wrong.example.com:443"}},
		"customer":{"proxy":{"type":"http"}}
	}`), 0644); err != nil {
		t.Fatalf("seed temp home config failed: %v", err)
	}

	h := NewConfigHandler()
	h.customerConfigService = services.NewCustomerConfigService()

	payload := []byte(`{"customer":{"proxy":{"type":"socks5","server":"127.0.0.1:1080"},"ai_rules":{},"proxy_rules":[]}}`)
	req := httptest.NewRequest(http.MethodPost, "/api/config/customer", bytes.NewReader(payload))
	rec := httptest.NewRecorder()
	h.HandleCustomerConfig(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}

	data := decodeCommonResponseData(t, rec)
	customer, _ := data["customer"].(map[string]interface{})
	if customer == nil {
		t.Fatalf("expected customer object in response: %v", data)
	}
	proxy, _ := customer["proxy"].(map[string]interface{})
	if proxy == nil {
		t.Fatalf("expected proxy object in customer response: %v", customer)
	}
	if proxy["type"] != "socks5" {
		t.Fatalf("expected updated proxy type socks5, got %#v", proxy["type"])
	}
	if proxy["server"] != "127.0.0.1:1080" {
		t.Fatalf("expected updated proxy server 127.0.0.1:1080, got %#v", proxy["server"])
	}

	committed := config.GetEffectiveConfigCommitCoordinator().LastCommittedSnapshot()
	if committed == nil {
		t.Fatal("expected committed snapshot")
	}
	if !strings.Contains(committed.Content, `"api_server":"https://sub2api.liang.home"`) {
		t.Fatalf("expected committed snapshot to preserve startup api_server, got %s", committed.Content)
	}
}

func TestConfigHandler_CustomerConfigUpdateSucceedsWhenGlobalConfigAndFileMissing(t *testing.T) {
	config.ResetGlobalConfigForTest()
	config.ResetEffectiveConfigCommitCoordinatorForTest()
	_ = os.Remove("./config.new.json")
	t.Cleanup(func() {
		config.ResetGlobalConfigForTest()
		config.ResetEffectiveConfigCommitCoordinatorForTest()
		_ = os.Remove("./config.new.json")
	})

	h := NewConfigHandler()
	h.customerConfigService = services.NewCustomerConfigService()

	payload := []byte(`{"customer":{"proxy":{"type":"socks5","server":"127.0.0.1:1080"},"ai_rules":{},"proxy_rules":[]}}`)
	req := httptest.NewRequest(http.MethodPost, "/api/config/customer", bytes.NewReader(payload))
	rec := httptest.NewRecorder()
	h.HandleCustomerConfig(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}

	data := decodeCommonResponseData(t, rec)
	customer, _ := data["customer"].(map[string]interface{})
	if customer == nil {
		t.Fatalf("expected customer object in response: %v", data)
	}
	proxy, _ := customer["proxy"].(map[string]interface{})
	if proxy == nil {
		t.Fatalf("expected proxy object in customer response: %v", customer)
	}
	if proxy["type"] != "socks5" {
		t.Fatalf("expected updated proxy type socks5, got %#v", proxy["type"])
	}
}

func boolPtr(v bool) *bool {
	return &v
}
