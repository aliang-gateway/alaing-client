package server

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"nursor.org/nursorgate/common/logger"
	client_cert "nursor.org/nursorgate/processor/cert/client"
)

var (
	// ClientCert and ClientKey will be loaded from filesystem
	// var ClientCert []byte
	// var ClientKey []byte

	// CaCert will be loaded from filesystem
	// var CaCert []byte

	cachedOutboundCert *OutboundCert
	certMutex          sync.RWMutex
)

type OutboundCert struct {
	cert      *tls.Certificate
	ca        *x509.CertPool
	tlsConfig *tls.Config
	token     string
}

// loadClientCertFromFilesystem loads the mTLS client certificate from filesystem
// Falls back to embedded certificate if filesystem certificate is not available
func loadClientCertFromFilesystem() (*tls.Certificate, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get user home directory: %w", err)
	}

	certPath := filepath.Join(homeDir, ".nonelane", "mtls-client.pem")
	keyPath := filepath.Join(homeDir, ".nonelane", "mtls-client.pem.key")

	// Check if files exist
	if _, err := os.Stat(certPath); err == nil {
		// Load certificate and key from filesystem
		cert, err := tls.LoadX509KeyPair(certPath, keyPath)
		if err == nil {
			logger.Info("Loaded mTLS client certificate from " + certPath)
			return &cert, nil
		}
		logger.Warn(fmt.Sprintf("Failed to load certificate from %s: %v", certPath, err))
	}

	// Fallback to embedded certificate if available
	logger.Warn("Falling back to embedded mTLS client certificate")
	return nil, fmt.Errorf("mTLS client certificate not found in filesystem and no embedded fallback available")
}

// loadRootCAFromFilesystem loads the Root CA certificate from filesystem
func loadRootCAFromFilesystem() (*x509.CertPool, error) {
	// Use the client_cert module's GetCaCertPool which handles filesystem loading
	caCertPool := client_cert.GetCaCertPool()
	if caCertPool == nil {
		return nil, fmt.Errorf("failed to load Root CA certificate")
	}
	return caCertPool, nil
}

func GetOutboundCert(isHttp2 bool, SNIName string) *OutboundCert {
	certMutex.RLock()
	if cachedOutboundCert != nil {
		certMutex.RUnlock()
		return cachedOutboundCert
	}
	certMutex.RUnlock()

	// Load client certificate from filesystem
	cert, err := loadClientCertFromFilesystem()
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to load client certificate: %v", err))
		return nil
	}

	// Load Root CA from filesystem
	caCertPool, err := loadRootCAFromFilesystem()
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to load Root CA: %v", err))
		return nil
	}

	logger.Debug("OutboundCert: CA cert loaded successfully for " + SNIName)
	var tlsConfig = &tls.Config{
		RootCAs:            caCertPool,
		Certificates:       []tls.Certificate{*cert},
		ServerName:         SNIName,
		InsecureSkipVerify: true,
	}
	if isHttp2 {
		tlsConfig.NextProtos = []string{"h2", "http/1.1"}
	}

	outboundCert := &OutboundCert{
		cert:      cert,
		ca:        caCertPool,
		tlsConfig: tlsConfig,
		token:     "",
	}

	// Cache the certificate
	certMutex.Lock()
	cachedOutboundCert = outboundCert
	certMutex.Unlock()

	return outboundCert
}

func (c *OutboundCert) SetToken(token string) {
	c.token = token
}

func (c *OutboundCert) GetToken() string {
	return c.token
}

func (c *OutboundCert) GetTLSConfig() *tls.Config {
	return c.tlsConfig
}

func (c *OutboundCert) GetCert() *tls.Certificate {
	return c.cert
}

func (c *OutboundCert) GetCA() *x509.CertPool {
	return c.ca
}
