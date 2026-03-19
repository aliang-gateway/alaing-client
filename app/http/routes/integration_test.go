package routes

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRunStopAndCertHTTPIntegration(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	h := NewHandlers()
	mux := http.NewServeMux()
	RegisterRoutes(h, mux)

	stopReq := httptest.NewRequest(http.MethodPost, "/api/run/stop", bytes.NewReader([]byte(`{}`)))
	stopReq.Header.Set("Content-Type", "application/json")
	stopRec := httptest.NewRecorder()
	mux.ServeHTTP(stopRec, stopReq)

	if stopRec.Code != http.StatusOK {
		t.Fatalf("run stop http status=%d body=%s", stopRec.Code, stopRec.Body.String())
	}

	var stopResp map[string]interface{}
	if err := json.Unmarshal(stopRec.Body.Bytes(), &stopResp); err != nil {
		t.Fatalf("decode run stop response failed: %v", err)
	}

	stopData, _ := stopResp["data"].(map[string]interface{})
	if stopData["error"] != "not_running" {
		t.Fatalf("expected run stop error=not_running, got %#v", stopData["error"])
	}

	exportReq := httptest.NewRequest(http.MethodPost, "/api/cert/export", bytes.NewReader([]byte(`{"cert_type":"root-ca"}`)))
	exportReq.Header.Set("Content-Type", "application/json")
	exportRec := httptest.NewRecorder()
	mux.ServeHTTP(exportRec, exportReq)

	if exportRec.Code != http.StatusOK {
		t.Fatalf("cert export http status=%d body=%s", exportRec.Code, exportRec.Body.String())
	}

	statusReq := httptest.NewRequest(http.MethodGet, "/api/cert/status?cert_type=root-ca", nil)
	statusRec := httptest.NewRecorder()
	mux.ServeHTTP(statusRec, statusReq)

	if statusRec.Code != http.StatusOK {
		t.Fatalf("cert status http status=%d body=%s", statusRec.Code, statusRec.Body.String())
	}

	var statusResp map[string]interface{}
	if err := json.Unmarshal(statusRec.Body.Bytes(), &statusResp); err != nil {
		t.Fatalf("decode cert status response failed: %v", err)
	}

	statusData, _ := statusResp["data"].(map[string]interface{})
	if statusData["cert_type"] != "root-ca" {
		t.Fatalf("expected cert_type=root-ca, got %#v", statusData["cert_type"])
	}

	downloadReq := httptest.NewRequest(http.MethodGet, "/api/cert/download?cert_type=root-ca", nil)
	downloadRec := httptest.NewRecorder()
	mux.ServeHTTP(downloadRec, downloadReq)

	if downloadRec.Code != http.StatusOK {
		t.Fatalf("cert download http status=%d body=%s", downloadRec.Code, downloadRec.Body.String())
	}

	if contentDisposition := downloadRec.Header().Get("Content-Disposition"); !strings.Contains(contentDisposition, "root-ca.pem") {
		t.Fatalf("unexpected Content-Disposition: %s", contentDisposition)
	}
}
