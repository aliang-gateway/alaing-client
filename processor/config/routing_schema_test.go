package config

import (
	"errors"
	"strings"
	"testing"
)

func TestCanonicalOnly_NormalizeRoutingSchemaJSON_ValidateMatrixValid(t *testing.T) {
	tests := []struct {
		name    string
		payload []byte
	}{
		{
			name: "valid canonical tun with explicit default egress",
			payload: []byte(`{
				"version": 1,
				"ingress": {"mode": "tun"},
				"egress": {
					"direct": {"enabled": true},
					"toAliang": {"enabled": true},
					"toSocks": {"enabled": true, "upstream": {"type": "http"}}
				},
				"routing": {
					"default_egress": "toAliang",
					"rules": [
						{"id": "r1", "type": "domain", "condition": "x.com", "enabled": true, "target": "toSocks"},
						{"id": "r2", "type": "domain", "condition": "y.com", "enabled": true, "target": "direct"}
					]
				}
			}`),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NormalizeRoutingSchemaJSON(tt.payload)
			if err != nil {
				t.Fatalf("NormalizeRoutingSchemaJSON() error = %v, want nil", err)
			}
		})
	}
}

func TestCanonicalOnly_NormalizeRoutingSchemaJSON_ValidateMatrixInvalid(t *testing.T) {
	tests := []struct {
		name     string
		payload  []byte
		errType  string
		errField string
		errLike  string
	}{
		{
			name: "reject unknown canonical field",
			payload: []byte(`{
				"version": 1,
				"ingress": {"mode": "tun", "extra": true},
				"egress": {
					"direct": {"enabled": true},
					"toAliang": {"enabled": true},
					"toSocks": {"enabled": true, "upstream": {"type": "socks"}}
				},
				"routing": {"rules": []}
			}`),
			errLike: "failed to parse canonical routing schema",
		},
		{
			name: "reject invalid ingress mode enum",
			payload: []byte(`{
				"version": 1,
				"ingress": {"mode": "tcp"},
				"egress": {
					"direct": {"enabled": true},
					"toAliang": {"enabled": true},
					"toSocks": {"enabled": true, "upstream": {"type": "socks"}}
				},
				"routing": {"rules": []}
			}`),
			errType:  "invalid_enum",
			errField: "ingress.mode",
			errLike:  "must be one of [tun http]",
		},
		{
			name: "reject direct disabled",
			payload: []byte(`{
				"version": 1,
				"ingress": {"mode": "http"},
				"egress": {
					"direct": {"enabled": false},
					"toAliang": {"enabled": true},
					"toSocks": {"enabled": true, "upstream": {"type": "socks"}}
				},
				"routing": {"rules": []}
			}`),
			errType:  "branch_required_enabled",
			errField: "egress.direct.enabled",
			errLike:  "direct must exist and be enabled",
		},
		{
			name: "reject invalid toSocks upstream type without coercion",
			payload: []byte(`{
				"version": 1,
				"ingress": {"mode": "tun"},
				"egress": {
					"direct": {"enabled": true},
					"toAliang": {"enabled": true},
					"toSocks": {"enabled": true, "upstream": {"type": "grpc"}}
				},
				"routing": {"rules": []}
			}`),
			errType:  "invalid_enum",
			errField: "egress.toSocks.upstream.type",
			errLike:  "must be one of [socks http]",
		},
		{
			name: "reject unresolvable default egress target disabled branch",
			payload: []byte(`{
				"version": 1,
				"ingress": {"mode": "tun"},
				"egress": {
					"direct": {"enabled": true},
					"toAliang": {"enabled": false},
					"toSocks": {"enabled": true, "upstream": {"type": "socks"}}
				},
				"routing": {
					"default_egress": "toAliang",
					"rules": []
				}
			}`),
			errType:  "unresolvable_default_egress",
			errField: "routing.default_egress",
			errLike:  "default egress target toAliang is disabled",
		},
		{
			name: "reject default egress deny",
			payload: []byte(`{
				"version": 1,
				"ingress": {"mode": "tun"},
				"egress": {
					"direct": {"enabled": true},
					"toAliang": {"enabled": true},
					"toSocks": {"enabled": true, "upstream": {"type": "socks"}}
				},
				"routing": {
					"default_egress": "deny",
					"rules": []
				}
			}`),
			errLike: "routing.default_egress normalize failed: unknown target: deny",
		},
		{
			name: "reject unknown default egress target",
			payload: []byte(`{
				"version": 1,
				"ingress": {"mode": "tun"},
				"egress": {
					"direct": {"enabled": true},
					"toAliang": {"enabled": true},
					"toSocks": {"enabled": true, "upstream": {"type": "socks"}}
				},
				"routing": {
					"default_egress": "mystery",
					"rules": []
				}
			}`),
			errLike: "routing.default_egress normalize failed: unknown target: mystery",
		},
		{
			name: "reject unknown rule target",
			payload: []byte(`{
				"version": 1,
				"ingress": {"mode": "tun"},
				"egress": {
					"direct": {"enabled": true},
					"toAliang": {"enabled": true},
					"toSocks": {"enabled": true, "upstream": {"type": "socks"}}
				},
				"routing": {
					"rules": [
						{"id": "r1", "type": "domain", "condition": "x.com", "enabled": true, "target": "mystery"}
					]
				}
			}`),
			errLike: "routing.rules[0].target normalize failed: unknown target: mystery",
		},
		{
			name: "reject rule targeting disabled branch",
			payload: []byte(`{
				"version": 1,
				"ingress": {"mode": "tun"},
				"egress": {
					"direct": {"enabled": true},
					"toAliang": {"enabled": false},
					"toSocks": {"enabled": true, "upstream": {"type": "socks"}}
				},
				"routing": {
					"rules": [
						{"id": "r1", "type": "domain", "condition": "x.com", "enabled": true, "target": "toAliang"}
					]
				}
			}`),
			errType:  "disabled_target_branch",
			errField: "routing.rules[0].target",
			errLike:  "target branch toAliang is disabled",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NormalizeRoutingSchemaJSON(tt.payload)
			if err == nil {
				t.Fatalf("NormalizeRoutingSchemaJSON() error = nil, want non-nil")
			}

			if tt.errType != "" || tt.errField != "" {
				var verr *RoutingSchemaValidationError
				if !errors.As(err, &verr) {
					t.Fatalf("error type = %T, want *RoutingSchemaValidationError, err=%v", err, err)
				}
				if tt.errType != "" && verr.Code != tt.errType {
					t.Fatalf("error code = %q, want %q", verr.Code, tt.errType)
				}
				if tt.errField != "" && verr.Field != tt.errField {
					t.Fatalf("error field = %q, want %q", verr.Field, tt.errField)
				}
			}

			if tt.errLike != "" && !strings.Contains(err.Error(), tt.errLike) {
				t.Fatalf("error = %q, want contains %q", err.Error(), tt.errLike)
			}
		})
	}
}

func TestCanonicalOnly_NormalizeRoutingSchemaJSON_LegacyAliasesRejected(t *testing.T) {
	tests := []struct {
		name    string
		payload []byte
		errLike string
	}{
		{
			name: "reject none_lane legacy target",
			payload: []byte(`{
				"version": 1,
				"ingress": {"mode": "tun"},
				"egress": {
					"direct": {"enabled": true},
					"toAliang": {"enabled": true},
					"toSocks": {"enabled": true, "upstream": {"type": "socks"}}
				},
				"routing": {
					"rules": [
						{"id": "r1", "type": "domain", "condition": "none.example.com", "enabled": true, "target": "none_lane"}
					]
				}
			}`),
			errLike: "unknown target: none_lane",
		},
		{
			name: "reject to_door legacy target",
			payload: []byte(`{
				"version": 1,
				"ingress": {"mode": "tun"},
				"egress": {
					"direct": {"enabled": true},
					"toAliang": {"enabled": true},
					"toSocks": {"enabled": true, "upstream": {"type": "socks"}}
				},
				"routing": {
					"rules": [
						{"id": "r1", "type": "domain", "condition": "door.example.com", "enabled": true, "target": "to_door"}
					]
				}
			}`),
			errLike: "unknown target: to_door",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NormalizeRoutingSchemaJSON(tt.payload)
			if err == nil {
				t.Fatalf("NormalizeRoutingSchemaJSON() error = nil, want non-nil")
			}
			if !strings.Contains(err.Error(), tt.errLike) {
				t.Fatalf("error = %q, want contains %q", err.Error(), tt.errLike)
			}
		})
	}
}

func TestNormalizeRoutingSchemaJSON_NormalizeRejectUnknownTarget(t *testing.T) {
	tests := []struct {
		name    string
		payload []byte
		errLike string
	}{
		{
			name: "unknown canonical target rejected",
			payload: []byte(`{
				"version": 1,
				"ingress": {"mode": "tun"},
				"egress": {
					"direct": {"enabled": true},
					"toAliang": {"enabled": true},
					"toSocks": {"enabled": true, "upstream": {"type": "socks"}}
				},
				"routing": {
					"rules": [
						{"id": "r1", "type": "domain", "condition": "x.com", "enabled": true, "target": "mystery"}
					]
				}
			}`),
			errLike: "unknown target: mystery",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NormalizeRoutingSchemaJSON(tt.payload)
			if err == nil {
				t.Fatalf("NormalizeRoutingSchemaJSON() error = nil, want non-nil")
			}
			if !strings.Contains(err.Error(), tt.errLike) {
				t.Fatalf("error = %q, want contains %q", err.Error(), tt.errLike)
			}
		})
	}
}

func TestLegacyReject_NormalizeRoutingSchemaJSON_LegacyAliasAdapter(t *testing.T) {
	legacy := []byte(`{
		"none_lane": {
			"set_type": "none_lane",
			"rules": [{"id": "n1", "type": "domain", "condition": "none.example.com", "enabled": true}]
		},
		"to_door": {
			"set_type": "to_door",
			"rules": [{"id": "d1", "type": "domain", "condition": "door.example.com", "enabled": true}]
		},
		"black_list": {
			"set_type": "black_list",
			"rules": [{"id": "b1", "type": "domain", "condition": "blocked.example.com", "enabled": true}]
		},
		"settings": {
			"aliang_enabled": true,
			"socks_enabled": true
		},
		"version": 1
	}`)

	_, err := NormalizeRoutingSchemaJSON(legacy)
	if err == nil {
		t.Fatalf("NormalizeRoutingSchemaJSON() error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "unsupported legacy keys") {
		t.Fatalf("error = %q, want contains unsupported legacy keys", err.Error())
	}
}

func TestLegacyReject_NormalizeRoutingSchemaJSON_LegacyAliasRejectUnknown(t *testing.T) {
	tests := []struct {
		name    string
		payload []byte
		errLike string
	}{
		{
			name: "reject unknown top-level legacy variant with migration guidance",
			payload: []byte(`{
				"geoip_list": {
					"set_type": "geoip_list",
					"rules": [{"id": "g1", "type": "domain", "condition": "geo.example.com", "enabled": true}]
				},
				"version": 1
			}`),
			errLike: "unsupported legacy keys [geoip_list version]; migrate to canonical keys ingress/egress/routing",
		},
		{
			name: "reject to_door legacy key",
			payload: []byte(`{
				"to_door": {
					"set_type": "to_door",
					"rules": [{"id": "d1", "type": "domain", "condition": "door.example.com", "enabled": true}]
				},
				"version": 1
			}`),
			errLike: "unsupported legacy keys [to_door version]; migrate to canonical keys ingress/egress/routing",
		},
		{
			name: "reject none_lane legacy key",
			payload: []byte(`{
				"none_lane": {
					"set_type": "none_lane",
					"rules": [{"id": "n1", "type": "domain", "condition": "none.example.com", "enabled": true}]
				},
				"version": 1
			}`),
			errLike: "unsupported legacy keys [none_lane version]; migrate to canonical keys ingress/egress/routing",
		},
		{
			name: "reject black_list legacy key",
			payload: []byte(`{
				"black_list": {
					"set_type": "black_list",
					"rules": [{"id": "b1", "type": "domain", "condition": "blocked.example.com", "enabled": true}]
				},
				"version": 1
			}`),
			errLike: "unsupported legacy keys [black_list version]; migrate to canonical keys ingress/egress/routing",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NormalizeRoutingSchemaJSON(tt.payload)
			if err == nil {
				t.Fatalf("NormalizeRoutingSchemaJSON() error = nil, want non-nil")
			}
			if !strings.Contains(err.Error(), tt.errLike) {
				t.Fatalf("error = %q, want contains %q", err.Error(), tt.errLike)
			}
		})
	}
}

func TestLegacyReject_NormalizeRoutingSchemaJSON_LegacyMetadataKeysRejected(t *testing.T) {
	legacy := []byte(`{
		"routing": {"rules": []},
		"ingress": {"mode": "tun"},
		"egress": {
			"direct": {"enabled": true},
			"toAliang": {"enabled": true},
			"toSocks": {"enabled": true, "upstream": {"type": "socks"}}
		},
		"version": 1,
		"created_at": "2026-01-01T00:00:00Z",
		"updated_at": "2026-01-01T00:00:00Z"
	}`)

	_, err := NormalizeRoutingSchemaJSON(legacy)
	if err == nil {
		t.Fatalf("NormalizeRoutingSchemaJSON() error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "failed to parse canonical routing schema") {
		t.Fatalf("error = %q, want contains canonical parse failure", err.Error())
	}
}

func TestLegacyReject_NormalizeRoutingSchemaJSON_LegacyBoundary(t *testing.T) {
	tests := []struct {
		name            string
		payload         []byte
		wantErrContains string
	}{
		{
			name: "known legacy aliases are rejected",
			payload: []byte(`{
				"none_lane": {
					"set_type": "none_lane",
					"rules": [{"id": "n1", "type": "domain", "condition": "none.example.com", "enabled": true}]
				},
				"to_door": {
					"set_type": "to_door",
					"rules": [{"id": "d1", "type": "domain", "condition": "door.example.com", "enabled": true}]
				},
				"black_list": {
					"set_type": "black_list",
					"rules": [{"id": "b1", "type": "domain", "condition": "blocked.example.com", "enabled": true}]
				},
				"settings": {"aliang_enabled": true, "socks_enabled": true},
				"version": 1
			}`),
			wantErrContains: "unsupported legacy keys",
		},
		{
			name: "unknown legacy key is rejected with migration guidance",
			payload: []byte(`{
				"geoip_list": {
					"set_type": "geoip_list",
					"rules": [{"id": "g1", "type": "domain", "condition": "geo.example.com", "enabled": true}]
				},
				"version": 1
			}`),
			wantErrContains: "unsupported legacy keys [geoip_list version]; migrate to canonical keys ingress/egress/routing",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NormalizeRoutingSchemaJSON(tt.payload)
			if tt.wantErrContains != "" {
				if err == nil {
					t.Fatalf("NormalizeRoutingSchemaJSON() error = nil, want contains %q", tt.wantErrContains)
				}
				if !strings.Contains(err.Error(), tt.wantErrContains) {
					t.Fatalf("error = %q, want contains %q", err.Error(), tt.wantErrContains)
				}
				return
			}

			if err != nil {
				t.Fatalf("NormalizeRoutingSchemaJSON() error = %v", err)
			}
			if got == nil {
				t.Fatal("expected canonical payload result to be non-nil")
			}
		})
	}
}

func TestCanonicalOnly_NormalizeRoutingSchemaJSON_CanonicalHappyPath(t *testing.T) {
	payload := []byte(`{
		"version": 1,
		"ingress": {"mode": "tun"},
		"egress": {
			"direct": {"enabled": true},
			"toAliang": {"enabled": true},
			"toSocks": {"enabled": true, "upstream": {"type": "http"}}
		},
		"routing": {
			"default_egress": "toAliang",
			"rules": [
				{"id": "r1", "type": "domain", "condition": "x.com", "enabled": true, "target": "toSocks"}
			]
		}
	}`)

	got, err := NormalizeRoutingSchemaJSON(payload)
	if err != nil {
		t.Fatalf("NormalizeRoutingSchemaJSON() error = %v", err)
	}
	if got.Ingress.Mode != "tun" {
		t.Fatalf("ingress.mode = %q, want tun", got.Ingress.Mode)
	}
	if got.Routing.DefaultEgress != "toAliang" {
		t.Fatalf("default_egress = %q, want toAliang", got.Routing.DefaultEgress)
	}
	if len(got.Routing.Rules) != 1 || got.Routing.Rules[0].Target != "toSocks" {
		t.Fatalf("rules target = %#v, want [toSocks]", got.Routing.Rules)
	}
}
