package cert

import (
	"crypto/tls"
	"crypto/x509"
	_ "embed"
)

//go:embed client.pem
var ClientCert []byte

//go:embed client.key.pem
var ClientKey []byte

//go:embed ca.pem
var CaCert []byte

//var outboundCert *OutboundCert

type OutboundCert struct {
	cert      *tls.Certificate
	ca        *x509.CertPool
	tlsConfig *tls.Config
	token     string
}

func GetOutboundCert(isHttp2 bool, SNIName string) *OutboundCert {
	//if outboundCert == nil {
	cert, err := tls.X509KeyPair(ClientCert, ClientKey)
	if err != nil {
		return nil
	}

	caCertPool := x509.NewCertPool()
	if !caCertPool.AppendCertsFromPEM(CaCert) {
		return nil
	}
	println("OutboundCert: CA cert loaded successfully", SNIName)
	var tlsConfig = &tls.Config{
		RootCAs:            caCertPool,
		Certificates:       []tls.Certificate{cert},
		ServerName:         SNIName,
		InsecureSkipVerify: true,
	}
	if isHttp2 {
		tlsConfig.NextProtos = []string{"h2", "http/1.1"}
	}

	outboundCert := &OutboundCert{
		cert:      &cert,
		ca:        caCertPool,
		tlsConfig: tlsConfig,
		token:     "",
	}
	//}
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
