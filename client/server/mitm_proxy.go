package server

import (
	"bufio"
	"crypto/tls"
	"io"
	"log"
	"net"
	"net/http"
	"strings"

	"nursor.org/nursorgate/common/logger"

	"nursor.org/nursorgate/client/inbound"

	"nursor.org/nursorgate/client/inbound/cert"
)

// 全局 CA 证书
func StartMitmHttp() {
	// 加载或生成 CA 证书
	// 启动代理服务器
	listener, err := net.Listen("tcp", "127.0.0.1:56432")
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}
	defer listener.Close()

	log.Println("Starting MITM proxy on :56432")
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Accept error: %v", err)
			continue
		}
		go handleRawConnection(conn)
	}

}

// 处理客户端连接
func handleRawConnection(conn net.Conn) {
	defer conn.Close()

	// 读取客户端初始数据，检查是否为 CONNECT 请求
	reader := bufio.NewReader(conn)
	req, err := http.ReadRequest(reader)
	if err != nil {
		log.Printf("Failed to read initial request: %v", err)
		return
	}

	if req.Method == "CONNECT" {
		log.Printf("Received CONNECT request for %s", req.Host)
		resp := http.Response{
			Status:        "200 Connection Established",
			StatusCode:    http.StatusOK,
			Proto:         "HTTP/1.1",
			ProtoMajor:    1,
			ProtoMinor:    1,
			Body:          io.NopCloser(strings.NewReader("")),
			ContentLength: 0,
		}
		if err := resp.Write(conn); err != nil {
			log.Printf("Failed to send 200 OK: %v", err)
			return
		}
		tlsConf := cert.CreateTlsConfigForHost(req.Host)
		tlsConn := tls.Server(conn, tlsConf)
		if err := tlsConn.Handshake(); err != nil {
			logger.Warn("TLS handshake with client failed:", req.Host, err)
			return
		}
		// 处理 TLS 后的请求
		if strings.Contains(req.Host, "api42.cursor") || strings.Contains(req.Host, "repo42.cursor") {
			state := tlsConn.ConnectionState()
			logger.Info("TLS handshake succeeded for", req.Host, "Version:", state.Version, "CipherSuite:", state.CipherSuite)
		}
		inbound.HandleTLSConnection(tlsConn, req)
	} else {
		inbound.HandleHttpConnection(conn, req)
	}
}
