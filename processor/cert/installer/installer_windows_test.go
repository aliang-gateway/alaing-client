//go:build windows

package installer

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha1"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/hex"
	"encoding/pem"
	"math/big"
	"strings"
	"testing"
	"time"

	cert_config "aliang.one/nursorgate/processor/cert"
)

func TestGetWindowsStoreTargets(t *testing.T) {
	t.Setenv("ALIANG_DATA_DIR", "")
	t.Setenv("ALIANG_SOCKET_PATH", "")

	rootTargets := getWindowsStoreTargets(cert_config.CertTypeMitmCA)
	if len(rootTargets) != 2 {
		t.Fatalf("expected 2 root targets, got %d", len(rootTargets))
	}
	if rootTargets[0].PSPath != "Cert:\\CurrentUser\\Root" {
		t.Fatalf("unexpected primary root store: %s", rootTargets[0].PSPath)
	}
	if rootTargets[1].PSPath != "Cert:\\LocalMachine\\Root" {
		t.Fatalf("unexpected fallback root store: %s", rootTargets[1].PSPath)
	}

	mtlsTargets := getWindowsStoreTargets(cert_config.CertTypeMtlsClient)
	if len(mtlsTargets) != 2 {
		t.Fatalf("expected 2 mTLS targets, got %d", len(mtlsTargets))
	}
	if mtlsTargets[0].PSPath != "Cert:\\CurrentUser\\My" {
		t.Fatalf("unexpected primary mTLS store: %s", mtlsTargets[0].PSPath)
	}
	if mtlsTargets[1].PSPath != "Cert:\\LocalMachine\\My" {
		t.Fatalf("unexpected fallback mTLS store: %s", mtlsTargets[1].PSPath)
	}
}

func TestGetWindowsStoreTargetsPrefersLocalMachineInDaemonMode(t *testing.T) {
	t.Setenv("ALIANG_DATA_DIR", `C:\ProgramData\Aliang`)
	t.Setenv("ALIANG_SOCKET_PATH", `\\.\pipe\aliang-core`)

	rootTargets := getWindowsStoreTargets(cert_config.CertTypeMitmCA)
	if rootTargets[0].PSPath != "Cert:\\LocalMachine\\Root" {
		t.Fatalf("unexpected primary root store in daemon mode: %s", rootTargets[0].PSPath)
	}
	if rootTargets[1].PSPath != "Cert:\\CurrentUser\\Root" {
		t.Fatalf("unexpected fallback root store in daemon mode: %s", rootTargets[1].PSPath)
	}

	mtlsTargets := getWindowsStoreTargets(cert_config.CertTypeMtlsClient)
	if mtlsTargets[0].PSPath != "Cert:\\LocalMachine\\My" {
		t.Fatalf("unexpected primary mTLS store in daemon mode: %s", mtlsTargets[0].PSPath)
	}
	if mtlsTargets[1].PSPath != "Cert:\\CurrentUser\\My" {
		t.Fatalf("unexpected fallback mTLS store in daemon mode: %s", mtlsTargets[1].PSPath)
	}
}

func TestExtractCertThumbprint(t *testing.T) {
	pemBytes, rawBytes := mustCreateTestCertificate(t)

	got, err := extractCertThumbprint(pemBytes)
	if err != nil {
		t.Fatalf("extractCertThumbprint returned error: %v", err)
	}

	sum := sha1.Sum(rawBytes)
	want := strings.ToUpper(hex.EncodeToString(sum[:]))
	if got != want {
		t.Fatalf("unexpected thumbprint: got %s want %s", got, want)
	}
}

func TestEscapePowerShellSingleQuoted(t *testing.T) {
	got := escapePowerShellSingleQuoted("C:\\ProgramData\\Aliang\\it's.pem")
	want := "C:\\ProgramData\\Aliang\\it''s.pem"
	if got != want {
		t.Fatalf("unexpected escaped string: got %q want %q", got, want)
	}
}

func TestWindowsInstallerIsTrustedReturnsFalseForMtls(t *testing.T) {
	installer := &WindowsInstaller{}

	trusted, err := installer.IsTrusted(cert_config.CertTypeMtlsClient, nil)
	if err != nil {
		t.Fatalf("IsTrusted returned error: %v", err)
	}
	if trusted {
		t.Fatal("expected mTLS certificate to not be treated as root-trusted")
	}
}

func mustCreateTestCertificate(t *testing.T) ([]byte, []byte) {
	t.Helper()

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}

	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			CommonName:   "aliang-test",
			Organization: []string{"Aliang"},
		},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
		BasicConstraintsValid: true,
		IsCA:                  true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature,
	}

	rawBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &privateKey.PublicKey, privateKey)
	if err != nil {
		t.Fatalf("failed to create certificate: %v", err)
	}

	pemBytes := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: rawBytes,
	})

	return pemBytes, rawBytes
}
