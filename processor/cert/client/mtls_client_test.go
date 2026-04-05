package client

import (
	"crypto/x509"
	"encoding/pem"
	"testing"
)

func TestEmbeddedMTLSClientCertificateHasClientAuthUsage(t *testing.T) {
	block, _ := pem.Decode(mtlsClientCertPEM)
	if block == nil {
		t.Fatal("failed to decode embedded mTLS client certificate PEM")
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		t.Fatalf("parse embedded mTLS client certificate: %v", err)
	}

	hasClientAuth := false
	for _, usage := range cert.ExtKeyUsage {
		if usage == x509.ExtKeyUsageClientAuth {
			hasClientAuth = true
			break
		}
	}

	if !hasClientAuth {
		t.Fatalf("expected embedded mTLS client certificate to include clientAuth EKU, got %v", cert.ExtKeyUsage)
	}
}

func TestGetMTLSClientTLSConfigBuilds(t *testing.T) {
	cfg, err := GetMTLSClientTLSConfig(true, "ai-gateway.aliang.one")
	if err != nil {
		t.Fatalf("GetMTLSClientTLSConfig returned error: %v", err)
	}
	if cfg == nil {
		t.Fatal("expected non-nil TLS config")
	}
	if len(cfg.Certificates) != 1 {
		t.Fatalf("expected exactly one client certificate, got %d", len(cfg.Certificates))
	}
	if cfg.ServerName != "ai-gateway.aliang.one" {
		t.Fatalf("unexpected server name: %q", cfg.ServerName)
	}
}
