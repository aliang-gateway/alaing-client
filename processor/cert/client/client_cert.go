package client

import (
	crand "crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"math/rand"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	"nursor.org/nursorgate/common/logger"

	_ "embed"

	"golang.org/x/net/http2"
)

//go:embed ca.pem
var caCert []byte

//go:embed mitm-ca.pem
var mitmCaCert []byte

//go:embed mitm-ca.key.pem
var mitmCaKey []byte

var defaultCertificate *tls.Certificate
var caCertPool *x509.CertPool

func ExportMitmCaCertToFile(certPath string) error {
	return os.WriteFile(certPath, mitmCaCert, 0644)
}

func ExportRootCaCertToFile(certPath string) error {
	return os.WriteFile(certPath, caCert, 0644)
}

func GetNursorCertificate() *tls.Certificate {
	if defaultCertificate == nil {
		newCertfile, err := tls.X509KeyPair(mitmCaCert, mitmCaKey)
		if err != nil {
			return nil
		}
		defaultCertificate = &newCertfile
	}
	return defaultCertificate
}

func GetRootCertBytes() []byte {
	block, _ := pem.Decode(caCert)
	return block.Bytes
}

func GetCaCertPool() *x509.CertPool {
	if caCertPool == nil {
		caCertPool = x509.NewCertPool()
		caCertPool.AppendCertsFromPEM(caCert)
	}
	return caCertPool
}

var certCache = sync.Map{}

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

	priv, err := rsa.GenerateKey(crand.Reader, 2048)
	if err != nil {
		return tls.Certificate{}, err
	}
	template := x509.Certificate{
		SerialNumber: big.NewInt(rand.Int63n(1 << 62)),
		Subject: pkix.Name{
			CommonName: host,
		},
		NotBefore:   time.Now(),
		NotAfter:    time.Now().Add(365 * 24 * time.Hour),
		KeyUsage:    x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		DNSNames:    []string{host},
	}

	if net.ParseIP(host) != nil {
		template.IPAddresses = append(template.IPAddresses, net.ParseIP(host))
	}

	caCert := GetNursorCertificate()
	ca, _ := x509.ParseCertificate(caCert.Certificate[0])

	derBytes, err := x509.CreateCertificate(crand.Reader, &template, ca, &priv.PublicKey, caCert.PrivateKey)
	if err != nil {
		return tls.Certificate{}, err
	}

	cert := tls.Certificate{
		Certificate: [][]byte{derBytes, caCert.Certificate[0], GetRootCertBytes()},
		PrivateKey:  priv,
	}

	certCache.Store(host, cert)
	return cert, nil
}

func CreateTlsConfigForHost(host string) *tls.Config {
	cert, err := creatCertForHost(host)
	if err != nil {
		logger.Error(err)
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
