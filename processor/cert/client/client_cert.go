package client

import (
	crand "crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"math/rand"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"aliang.one/nursorgate/common/cache"
	"aliang.one/nursorgate/common/logger"
	"aliang.one/nursorgate/processor/cert/generator"

	"golang.org/x/net/http2"

	cert_config "aliang.one/nursorgate/processor/cert"
)

var defaultCertificate *tls.Certificate
var caCertPool *x509.CertPool
var certCache = sync.Map{}
var mu sync.RWMutex

// GetCertDir returns the certificate directory path
func GetCertDir() (string, error) {
	certDir, err := cache.GetCacheDir()
	if err != nil {
		return "", err
	}
	return certDir, nil
}

// LoadMitmCACertificate loads the MITM CA certificate from filesystem
func LoadMitmCACertificate() (*tls.Certificate, error) {
	mu.RLock()
	if defaultCertificate != nil {
		mu.RUnlock()
		return defaultCertificate, nil
	}
	mu.RUnlock()

	certDir, err := GetCertDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get cert dir: %w", err)
	}

	certPath := filepath.Join(certDir, "mitm-ca.pem")
	keyPath := filepath.Join(certDir, "mitm-ca.pem.key")

	// Check if certificate files exist, if not generate them
	if _, err := os.Stat(certPath); os.IsNotExist(err) {
		logger.Warn("MITM CA certificate not found, generating new one")
		config := cert_config.GetCertConfig("mitm-ca")
		if config == nil {
			return nil, fmt.Errorf("MITM CA configuration not found")
		}
		if err := generator.GenerateCertificateFromConfig(config, certPath); err != nil {
			return nil, fmt.Errorf("failed to generate MITM CA certificate: %w", err)
		}
	}

	// Load the certificate
	cert, err := tls.LoadX509KeyPair(certPath, keyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load MITM CA certificate: %w", err)
	}

	mu.Lock()
	defaultCertificate = &cert
	mu.Unlock()

	return defaultCertificate, nil
}

// LoadRootCACertificate loads the Root CA certificate from filesystem
func LoadRootCACertificate() ([]byte, error) {
	certDir, err := GetCertDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get cert dir: %w", err)
	}

	certPath := filepath.Join(certDir, "root-ca.pem")

	// Check if certificate file exists, if not generate it
	if _, err := os.Stat(certPath); os.IsNotExist(err) {
		logger.Warn("Root CA certificate not found, generating new one")
		config := cert_config.GetCertConfig("root-ca")
		if config == nil {
			return nil, fmt.Errorf("Root CA configuration not found")
		}
		if err := generator.GenerateCertificateFromConfig(config, certPath); err != nil {
			return nil, fmt.Errorf("failed to generate root CA certificate: %w", err)
		}
	}

	return os.ReadFile(certPath)
}

// GetRootCertBytes returns the Root CA certificate bytes
func GetRootCertBytes() []byte {
	certBytes, err := LoadRootCACertificate()
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to load root CA certificate: %v", err))
		return nil
	}

	block, _ := pem.Decode(certBytes)
	if block == nil {
		logger.Error("Failed to decode root CA certificate PEM")
		return nil
	}

	return block.Bytes
}

// GetCaCertPool returns the CA certificate pool
func GetCaCertPool() *x509.CertPool {
	mu.RLock()
	if caCertPool != nil {
		mu.RUnlock()
		return caCertPool
	}
	mu.RUnlock()

	certBytes, err := LoadRootCACertificate()
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to load root CA certificate: %v", err))
		return nil
	}

	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(certBytes) {
		logger.Error("Failed to add root CA certificate to pool")
		return nil
	}

	mu.Lock()
	caCertPool = pool
	mu.Unlock()

	return pool
}

// creatCertForHost creates a TLS certificate for the specified host signed by MITM CA
func creatCertForHost(host string) (tls.Certificate, error) {
	var err error
	if strings.Contains(host, ":") {
		host, _, err = net.SplitHostPort(host)
		if err != nil {
			return tls.Certificate{}, err
		}
	}

	if cert, ok := certCache.Load(host); ok {
		return cert.(tls.Certificate), nil
	}

	// Load the MITM CA certificate
	caCert, err := LoadMitmCACertificate()
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("failed to load MITM CA certificate: %w", err)
	}

	// Generate private key for the host
	priv, err := rsa.GenerateKey(crand.Reader, 2048)
	if err != nil {
		return tls.Certificate{}, err
	}

	// Create certificate template
	template := x509.Certificate{
		SerialNumber: big.NewInt(rand.Int63n(1 << 62)),
		Subject: pkix.Name{
			CommonName: host,
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(365 * 24 * time.Hour),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		DNSNames:              []string{host},
		IsCA:                  true,
		BasicConstraintsValid: true,
	}

	if net.ParseIP(host) != nil {
		template.IPAddresses = append(template.IPAddresses, net.ParseIP(host))
	}

	// Parse CA certificate
	// caCert.Certificate[0] is already DER-encoded bytes (not PEM),
	// so we can parse it directly without PEM decoding
	// tls.LoadX509KeyPair returns Certificate field as DER-encoded bytes
	if len(caCert.Certificate) == 0 {
		return tls.Certificate{}, fmt.Errorf("CA certificate has no certificate chain")
	}

	ca, err := x509.ParseCertificate(caCert.Certificate[0])
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("failed to parse CA certificate: %w", err)
	}

	// Sign the certificate
	derBytes, err := x509.CreateCertificate(crand.Reader, &template, ca, &priv.PublicKey, caCert.PrivateKey)
	if err != nil {
		return tls.Certificate{}, err
	}

	// Build certificate chain:
	// 1. Server certificate (signed by MITM CA)
	// 2. MITM CA certificate (intermediate CA)
	//
	// IMPORTANT: Do NOT include Root CA in the chain if it's already in system trust store.
	// Including Root CA in the chain can cause "unknown certificate authority" errors because:
	// - Root CA certificates should be self-signed and trusted by the system
	// - Clients will use the Root CA from system trust store to verify the chain
	// - Including Root CA in the chain may confuse clients or cause validation failures
	//
	// The certificate chain should be: Server Cert -> MITM CA Cert
	// The client will use the Root CA from system trust store to verify MITM CA
	certChain := [][]byte{derBytes, caCert.Certificate[0]}

	cert := tls.Certificate{
		Certificate: certChain,
		PrivateKey:  priv,
	}

	certCache.Store(host, cert)
	return cert, nil
}

// CreateTlsConfigForHost creates a TLS configuration for the specified host
func CreateTlsConfigForHost(host string) *tls.Config {
	cert, err := creatCertForHost(host)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to create certificate for host %s: %v", host, err))
		return nil
	}

	certs := []tls.Certificate{
		cert,
	}

	return &tls.Config{
		Certificates:       certs,
		NextProtos:         []string{http2.NextProtoTLS, "http/1.1"},
		InsecureSkipVerify: true,
		MaxVersion:         tls.VersionTLS13,
		MinVersion:         tls.VersionTLS12,
	}
}

// Note: CertConfig is imported from processor/cert/config.go via cert_config alias
// The certificate type constants are also defined in processor/cert/config.go
