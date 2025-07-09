package server

import (
	"bufio"
	"crypto/tls"
	"log"
	"net"
	"net/http"

	"nursor.org/nursorgate/client/inbound"
	"nursor.org/nursorgate/client/inbound/cert"
	"nursor.org/nursorgate/client/outbound"
	"nursor.org/nursorgate/common/logger"
	"nursor.org/nursorgate/common/model"
)

// 全局 CA 证书
func StartMitmHttpSimple() {
	// 加载或生成 CA 证书
	// 启动代理服务器
	listener, err := net.Listen("tcp", "127.0.0.1:56432")
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}
	defer listener.Close()

	log.Println("Starting MITM proxy on 127.0.0.1:56432")
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Accept error: %v", err)
			continue
		}
		go handleRawConnectionSimple(conn)
	}

}

// 处理客户端连接
func handleRawConnectionSimple(conn net.Conn) {

	// 读取客户端初始数据，检查是否为 CONNECT 请求
	reader := bufio.NewReader(conn)
	req, err := http.ReadRequest(reader)
	if err != nil {
		log.Printf("Failed to read initial request: %v", err)
		return
	}

	if req.Method == "CONNECT" {
		log.Printf("Received CONNECT request for %s", req.Host)
		// resp := http.Response{
		// 	Status:        "200 Connection Established",
		// 	StatusCode:    http.StatusOK,
		// 	Proto:         "HTTP/1.1",
		// 	ProtoMajor:    1,
		// 	ProtoMinor:    1,
		// 	Body:          io.NopCloser(strings.NewReader("")),
		// 	ContentLength: 0,
		// }
		// if err := resp.Write(conn); err != nil {
		// 	log.Printf("Failed to send 200 OK: %v", err)
		// 	return
		// }
		_, err := conn.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n"))
		if err != nil {
			log.Printf("Failed to send 200 OK: %v", err)
			return
		}

		allowDomain := model.NewAllowProxyDomain()
		if allowDomain.IsAllowToCursor(req.Host) {
			tlsConf := cert.CreateTlsConfigForHost(req.Host)
			tlsConn := tls.Server(conn, tlsConf)
			if err := tlsConn.Handshake(); err != nil {
				logger.Error("TLS handshake with client failed:", req.Host, err)
				// return
			}
			inbound.HandleTLSConnectionSimple(tlsConn, req)
			// outbound.Direct(tlsConn, req)
		} else {
			outbound.Direct(conn, req)
		}
		// outbound.Direct(conn, req)

	} else {
		outbound.Direct(conn, req)
	}
}
