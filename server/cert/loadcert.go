package cert

import (
	"crypto/tls"
	_ "embed"
)

//go:embed mitm.server.pem
var ServerCaCert []byte

//go:embed mitm.server.key.pem
var ServerCaKey []byte

var defaultCertificate *tls.Certificate

func GetNursorCertificate() *tls.Certificate {
	if defaultCertificate == nil {
		newCertfile, err := tls.X509KeyPair(ServerCaCert, ServerCaKey)
		if err != nil {
			return nil
		}
		defaultCertificate = &newCertfile
	}
	return defaultCertificate
}
