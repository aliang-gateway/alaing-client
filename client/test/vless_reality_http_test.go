package test

import (
	"bufio"
	"log"
	"net"
	"net/http"
	"testing"

	"nursor.org/nursorgate/client/inbound"
)

func TestVLESSRealityHTTP(t *testing.T) {
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
		_, err := conn.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n"))
		if err != nil {
			log.Printf("Failed to send 200 OK: %v", err)
			return
		}
	}

	inbound.HandleTLSConnectionSimpleWithoutDecrypt(conn, req.Host, req.Host, req)
}

func handleRawConnectionSimple(conn net.Conn) {
	defer conn.Close()
	// 读取客户端初始数据，检查是否为 CONNECT 请求

	inbound.HandleTLSConnectionSimpleWithoutDecrypt(conn, "httpforever.com", "httpforever.com", nil)
}
