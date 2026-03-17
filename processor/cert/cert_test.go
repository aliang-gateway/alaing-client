package cert

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"testing"
	"time"
)

// TestGetCertConfig tests the GetCertConfig function
func TestGetCertConfig(t *testing.T) {
	testCases := []struct {
		name         string
		certType     string
		expectNil    bool
		expectCN     string
		expectIssuer string
	}{
		{
			name:         "MITM CA Config",
			certType:     CertTypeMitmCA,
			expectNil:    false,
			expectCN:     "aliang",
			expectIssuer: "aliang.com",
		},
		{
			name:         "Root CA Config",
			certType:     CertTypeRootCA,
			expectNil:    false,
			expectCN:     "aliang",
			expectIssuer: "aliang.com",
		},
		{
			name:         "mTLS Client Config",
			certType:     CertTypeMtlsClient,
			expectNil:    false,
			expectCN:     "aliang",
			expectIssuer: "aliang.com",
		},
		{
			name:      "Unknown Config",
			certType:  "unknown-type",
			expectNil: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			config := GetCertConfig(tc.certType)
			if tc.expectNil && config != nil {
				t.Errorf("Expected nil config for type %s, got %v", tc.certType, config)
			}
			if !tc.expectNil && config == nil {
				t.Errorf("Expected config for type %s, got nil", tc.certType)
			}
			if config != nil {
				if config.CN != tc.expectCN {
					t.Errorf("Expected CN %s, got %s", tc.expectCN, config.CN)
				}
				if config.Issuer != tc.expectIssuer {
					t.Errorf("Expected Issuer %s, got %s", tc.expectIssuer, config.Issuer)
				}
			}
		})
	}
}

// TestAllCertTypes tests the AllCertTypes function
func TestAllCertTypes(t *testing.T) {
	certTypes := AllCertTypes()

	if len(certTypes) != 3 {
		t.Errorf("Expected 3 certificate types, got %d", len(certTypes))
	}

	expectedTypes := map[string]bool{
		CertTypeMitmCA:     false,
		CertTypeRootCA:     false,
		CertTypeMtlsClient: false,
	}

	for _, certType := range certTypes {
		if _, exists := expectedTypes[certType]; !exists {
			t.Errorf("Unexpected certificate type: %s", certType)
		}
		expectedTypes[certType] = true
	}

	for certType, found := range expectedTypes {
		if !found {
			t.Errorf("Certificate type %s not found in AllCertTypes()", certType)
		}
	}
}

// TestCertConfigProperties tests certificate configuration properties
func TestCertConfigProperties(t *testing.T) {
	testCases := []struct {
		name       string
		config     *CertConfig
		expectKey  int
		expectYear int
	}{
		{
			name:       "MITM CA RSA 2048 10 years",
			config:     &MitmCAConfig,
			expectKey:  2048,
			expectYear: 10,
		},
		{
			name:       "Root CA RSA 2048 10 years",
			config:     &RootCAConfig,
			expectKey:  2048,
			expectYear: 10,
		},
		{
			name:       "mTLS Client RSA 2048 10 years",
			config:     &MtlsClientConfig,
			expectKey:  2048,
			expectYear: 10,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.config.KeySize != tc.expectKey {
				t.Errorf("Expected KeySize %d, got %d", tc.expectKey, tc.config.KeySize)
			}
			if tc.config.ValidityYears != tc.expectYear {
				t.Errorf("Expected ValidityYears %d, got %d", tc.expectYear, tc.config.ValidityYears)
			}
		})
	}
}

// TestCertConfigConsistency tests that all cert configs have consistent properties
func TestCertConfigConsistency(t *testing.T) {
	configs := []*CertConfig{
		&MitmCAConfig,
		&RootCAConfig,
		&MtlsClientConfig,
	}

	for _, config := range configs {
		if config.CN == "" {
			t.Errorf("Certificate config has empty CN")
		}
		if config.Issuer == "" {
			t.Errorf("Certificate config has empty Issuer")
		}
		if config.Country == "" {
			t.Errorf("Certificate config has empty Country")
		}
		if config.Organization == "" {
			t.Errorf("Certificate config has empty Organization")
		}
		if config.FileName == "" {
			t.Errorf("Certificate config has empty FileName")
		}
		if config.CertType == "" {
			t.Errorf("Certificate config has empty CertType")
		}
		if config.KeySize <= 0 {
			t.Errorf("Certificate config has invalid KeySize: %d", config.KeySize)
		}
		if config.ValidityYears <= 0 {
			t.Errorf("Certificate config has invalid ValidityYears: %d", config.ValidityYears)
		}
	}
}

// Helper function to create a test certificate for testing CN extraction
func createTestCertificate(cn string) []byte {
	privateKey, _ := rsa.GenerateKey(rand.Reader, 2048)

	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			CommonName: cn,
		},
		NotBefore: time.Now(),
		NotAfter:  time.Now().AddDate(1, 0, 0),
	}

	certBytes, _ := x509.CreateCertificate(rand.Reader, &template, &template, &privateKey.PublicKey, privateKey)

	return pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certBytes,
	})
}

// TestCertConfigAliangNaming tests that "aliang" is used consistently in configs
func TestCertConfigAliangNaming(t *testing.T) {
	configs := []*CertConfig{
		&MitmCAConfig,
		&RootCAConfig,
		&MtlsClientConfig,
	}

	for _, config := range configs {
		if config.CN != "aliang" {
			t.Errorf("Expected CN 'aliang' in %s, got '%s'", config.CertType, config.CN)
		}
		if config.Issuer != "aliang.com" {
			t.Errorf("Expected Issuer 'aliang.com' in %s, got '%s'", config.CertType, config.Issuer)
		}
	}
}
