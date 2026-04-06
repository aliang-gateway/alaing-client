package client

import (
	"crypto/x509"
	"net"
	"testing"
	"time"
)

func TestBuildHostLeafTemplateForDNSHost(t *testing.T) {
	before := time.Now().Add(-6 * time.Minute)
	template := buildHostLeafTemplate("api.openai.com")
	after := time.Now().Add(-4 * time.Minute)

	if template.IsCA {
		t.Fatal("expected MITM leaf certificate to not be a CA")
	}
	if len(template.ExtKeyUsage) != 1 || template.ExtKeyUsage[0] != x509.ExtKeyUsageServerAuth {
		t.Fatalf("expected serverAuth EKU, got %v", template.ExtKeyUsage)
	}
	if len(template.DNSNames) != 1 || template.DNSNames[0] != "api.openai.com" {
		t.Fatalf("unexpected DNS SANs: %v", template.DNSNames)
	}
	if len(template.IPAddresses) != 0 {
		t.Fatalf("expected no IP SANs for DNS host, got %v", template.IPAddresses)
	}
	if template.NotBefore.Before(before) || template.NotBefore.After(after) {
		t.Fatalf("expected NotBefore to be backdated by about 5 minutes, got %v", template.NotBefore)
	}
}

func TestBuildHostLeafTemplateForIPHost(t *testing.T) {
	template := buildHostLeafTemplate("127.0.0.1")

	if template.IsCA {
		t.Fatal("expected MITM leaf certificate to not be a CA")
	}
	if len(template.DNSNames) != 0 {
		t.Fatalf("expected no DNS SANs for IP host, got %v", template.DNSNames)
	}
	if len(template.IPAddresses) != 1 || !template.IPAddresses[0].Equal(net.ParseIP("127.0.0.1")) {
		t.Fatalf("unexpected IP SANs: %v", template.IPAddresses)
	}
}
