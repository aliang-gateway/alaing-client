package http

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"strings"

	"nursor.org/nursorgate/common/logger"
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
		logger.Debug("Received CONNECT request for " + req.Host)

		// Extract metadata from CONNECT request
		metadata, err := ExtractMetadataFromCONNECT(req, conn)
		if err != nil {
			logger.Error("Failed to extract metadata from CONNECT: " + err.Error())
			resp := &http.Response{
				Status:        "400 Bad Request",
				StatusCode:    http.StatusBadRequest,
				Proto:         "HTTP/1.1",
				ProtoMajor:    1,
				ProtoMinor:    1,
				Body:          io.NopCloser(strings.NewReader("")),
				ContentLength: 0,
			}
			resp.Write(conn)
			return
		}

		logger.Debug(fmt.Sprintf("CONNECT metadata: host=%s, port=%d, srcIP=%s, dstIP=%s",
			metadata.HostName, metadata.DstPort, metadata.SrcIP, metadata.DstIP))

		// Send 200 Connection Established
		// IMPORTANT: Use raw write instead of http.Response for proper formatting
		response200 := "HTTP/1.1 200 Connection Established\r\n\r\n"
		if _, err := conn.Write([]byte(response200)); err != nil {
			logger.Error("Failed to send 200 OK: " + err.Error())
			return
		}
		logger.Debug("Sent 200 Connection Established")

		// Handle CONNECT tunnel - establishes direct tunnel between client and target
		logger.Debug(fmt.Sprintf("Starting CONNECT tunnel for %s:%d", metadata.HostName, metadata.DstPort))
		if err := HandleCONNECTTunnel(conn, metadata); err != nil {
			logger.Error(fmt.Sprintf("CONNECT tunnel error for %s: %v", metadata.HostName, err))
		}
		logger.Debug("CONNECT tunnel closed for " + req.Host)
	} else {
		// 透明代理，基本不存在
		HandleHttpConnection(conn, req)
	}
}
