package listener

import (
	"crypto/tls"
	"log"
	"net"

	"nursor.org/nursorgate/client/inbound/cert"
)

func StartInbound() {
	listener, err := net.Listen("tcp", "0.0.0.0:8088")
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}
	defer listener.Close()

	log.Println("Starting MITM proxy on :8082")
	tlsConfig := &tls.Config{
		// 动态生成证书，基于客户端 SNI
		GetCertificate: func(info *tls.ClientHelloInfo) (*tls.Certificate, error) {
			log.Printf("Client SNI: %s", info.ServerName)
			return cert.GetNursorCertificate(), nil
		},
		// 支持 HTTP/2
		NextProtos: []string{"h2", "http/1.1"},
	}
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Accept error: %v", err)
			continue
		}
		tlsConn := tls.Server(conn, tlsConfig)
		if err := tlsConn.Handshake(); err != nil {
			log.Printf("TLS handshake error: %v", err)
			conn.Close()
			continue
		}
		log.Printf("TLS handshake successful for %s", conn.RemoteAddr())
		go func() {
			HandleTLSConnection(tlsConn)
		}()

	}
}
