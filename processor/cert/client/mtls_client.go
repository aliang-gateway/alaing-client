package client

import (
	"crypto/tls"
	"crypto/x509"
	_ "embed"
	"fmt"

	"aliang.one/nursorgate/common/logger"
)

//go:embed client.crt
var mtlsClientCertPEM []byte

//go:embed client.key
var mtlsClientKeyPEM []byte

//go:embed ca.pem
var mtlsCACertPEM []byte

// GetMTLSClientTLSConfig returns the outbound TLS config used by the aliang mTLS connector.
// The certificate material is embedded from processor/cert/client so the runtime consistently
// uses the dedicated client-auth certificate rather than the MITM server certificate.
//
// When isHTTP2 is false, the config intentionally does not advertise any ALPN.
// This keeps the mTLS channel as a raw encrypted tunnel that can carry arbitrary
// upper-layer payloads, including both HTTP/1.1 and HTTP/2 bytes.
func GetMTLSClientTLSConfig(isHTTP2 bool, serverName string) (*tls.Config, error) {
	cert, err := tls.X509KeyPair(mtlsClientCertPEM, mtlsClientKeyPEM)
	if err != nil {
		return nil, fmt.Errorf("load embedded mTLS client key pair: %w", err)
	}

	caCertPool := x509.NewCertPool()
	if !caCertPool.AppendCertsFromPEM(mtlsCACertPEM) {
		return nil, fmt.Errorf("load embedded mTLS CA certificate: append failed")
	}

	tlsConfig := &tls.Config{
		RootCAs:      caCertPool,
		Certificates: []tls.Certificate{cert},
		ServerName:   serverName,

		// We intentionally disable standard hostname verification because the upstream
		// uses a routing/SNI domain that may not match the certificate SANs. The actual
		// trust decision is enforced by VerifyConnection against our pinned CA.
		InsecureSkipVerify: true,
		VerifyConnection: func(cs tls.ConnectionState) error {
			if len(cs.PeerCertificates) == 0 {
				return x509.CertificateInvalidError{Reason: x509.NotAuthorizedToSign}
			}

			opts := x509.VerifyOptions{
				Roots:         caCertPool,
				Intermediates: x509.NewCertPool(),
			}

			for _, cert := range cs.PeerCertificates[1:] {
				opts.Intermediates.AddCert(cert)
			}

			if _, err := cs.PeerCertificates[0].Verify(opts); err != nil {
				logger.Error(fmt.Sprintf("mTLS verification failed for server %s: %v", serverName, err))
				return err
			}

			logger.Debug(fmt.Sprintf("mTLS verification succeeded for server %s", serverName))
			return nil
		},
	}

	if isHTTP2 {
		tlsConfig.NextProtos = []string{"h2", "http/1.1"}
	}

	return tlsConfig, nil
}
