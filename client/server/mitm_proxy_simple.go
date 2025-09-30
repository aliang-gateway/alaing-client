package server

import (
	"bufio"
	"log"
	"net"
	"net/http"

	"nursor.org/nursorgate/client/inbound"
	"nursor.org/nursorgate/client/outbound"
	"nursor.org/nursorgate/common/model"
)

// readerFirstConn 在 Read 时优先从已有的 bufio.Reader 中读取，
// 用于在 CONNECT 之后透传已经缓存在 reader 里的 TLS 首包字节。
type readerFirstConn struct {
	net.Conn
	r *bufio.Reader
}

func (c *readerFirstConn) Read(p []byte) (int, error) {
	if c.r != nil && c.r.Buffered() > 0 {
		return c.r.Read(p)
	}
	return c.Conn.Read(p)
}

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
		_, err := conn.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n"))
		if err != nil {
			log.Printf("Failed to send 200 OK: %v", err)
			return
		}

		allowDomain := model.NewAllowProxyDomain()
		if allowDomain.IsAllowToCursor(req.Host) || allowDomain.IsAllowToAnyDoor(req.Host) || allowDomain.IsAllowToGate(req.Host) {
			// tlsConf := cert.CreateTlsConfigForHost(req.Host)
			// tlsConn := tls.Server(conn, tlsConf)
			// if err := tlsConn.Handshake(); err != nil {
			// 	logger.Warn("TLS handshake with client failed:", req.Host, err)
			// 	return
			// }
			// inbound.HandleTLSConnectionSimple(tlsConn, req)
			// 使用包级 readerFirstConn，复用上面的 reader，避免丢失缓冲中的 TLS 首包

			// var newConn *readerFirstConn
			// if reader.Buffered() > 0 {
			// 	newConn = &readerFirstConn{Conn: conn, r: reader}
			// } else {
			// 	newConn = &readerFirstConn{Conn: conn, r: nil}
			// }
			inbound.HandleTLSConnectionSimpleWithoutDecrypt(conn, req.Host, req.Host, nil)
			// outbound.Direct(tlsConn, req)
		} else {
			outbound.Direct(conn, req)
		}
		// outbound.Direct(conn, req)

	} else {
		allowDomain := model.NewAllowProxyDomain()
		if allowDomain.IsAllowToCursor(req.Host) || allowDomain.IsAllowToAnyDoor(req.Host) || allowDomain.IsAllowToGate(req.Host) {
			inbound.HandleTLSConnectionSimpleWithoutDecrypt(conn, req.Host, req.Host, req)
		} else {
			outbound.Direct(conn, req)
		}
		// outbound.Direct(conn, req)
	}
}
