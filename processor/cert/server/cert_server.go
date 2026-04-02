package server

import (
	"crypto/tls"
	"crypto/x509"
	_ "embed"

	"aliang.one/nursorgate/common/logger"
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
	cert, err := tls.X509KeyPair(ClientCert, ClientKey)
	if err != nil {
		return nil
	}

	caCertPool := x509.NewCertPool()
	if !caCertPool.AppendCertsFromPEM(CaCert) {
		return nil
	}

	var tlsConfig = &tls.Config{
		RootCAs:      caCertPool,
		Certificates: []tls.Certificate{cert},
		ServerName:   SNIName, // 这里用于发送 SNI (伪装/路由)

		// 1. 必须设为 true，否则标准库会因为域名不匹配而报错
		InsecureSkipVerify: true,

		// 2. 【关键】使用这个回调手动验证证书
		VerifyConnection: func(cs tls.ConnectionState) error {
			// 如果没有证书，直接报错
			if len(cs.PeerCertificates) == 0 {
				return x509.CertificateInvalidError{Reason: x509.NotAuthorizedToSign}
			}

			// 3. 配置验证选项
			opts := x509.VerifyOptions{
				Roots:         caCertPool, // 指定只信任我们的 CA
				Intermediates: x509.NewCertPool(),
			}
			// 将中间证书加入验证池
			for _, cert := range cs.PeerCertificates[1:] {
				opts.Intermediates.AddCert(cert)
			}

			// 4. 【核心魔法】只验证签名，不验证域名
			// 我们故意不设置 opts.DNSName，这样 Verify 就只检查“是不是亲生的”，不检查“名字对不对”
			_, err := cs.PeerCertificates[0].Verify(opts)
			if err != nil {
				logger.Error("mTLS 验证失败: 服务端证书不是由指定 CA 签发的")
				return err
			}

			logger.Debug("mTLS 验证成功: 确认是自家服务端")
			return nil
		},
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
