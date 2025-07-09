package server

import (
	"crypto/tls"
	"log"
	"net"
	"net/http"
	"strconv"

	"nursor.org/nursorgate/common/logger"

	"nursor.org/nursorgate/client/inbound"
	"nursor.org/nursorgate/client/inbound/cert"
	"nursor.org/nursorgate/client/server/helper"
)

// 全局 CA 证书
func StartTproxyHttps() {
	// 加载或生成 CA 证书
	// 启动代理服务器
	port := 56433
	listener, err := net.Listen("tcp", "127.0.0.1:"+strconv.Itoa(port))
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}
	defer listener.Close()

	log.Println("Starting MITM proxy on :" + strconv.Itoa(port))
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Accept error: %v", err)
			continue
		}
		logger.Info("new connect coming")
		go HandleTProxyHttpsConnection(conn)
	}

}

// 处理客户端连接
func HandleTProxyHttpsConnection(conn net.Conn) {
	defer conn.Close()
	serverName, _, err := helper.ExtractSNI(conn)
	if err != nil {
		log.Printf("Failed to extract SNI: %v", err)
		return
	}
	var req = &http.Request{
		Host: serverName,
	}
	tlsConf := cert.CreateTlsConfigForHost(serverName)
	tlsConn := tls.Server(conn, tlsConf)
	if err := tlsConn.Handshake(); err != nil {
		log.Printf("TLS handshake with client failed: %v", err)
		return
	}
	// 处理 TLS 后的请求
	inbound.HandleTLSConnection(tlsConn, req)

}
