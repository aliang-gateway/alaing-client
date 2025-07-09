package outbound

import (
	"fmt"
	"log"
	"net"
	"strings"
)

func SendConnect(conn net.Conn, reqHost string) error {
	host, port, err := net.SplitHostPort(reqHost)
	connectReq := fmt.Sprintf("CONNECT %s:%s HTTP/1.1\r\nHost: %s\r\n\r\n", host, port, host)
	_, err = conn.Write([]byte(connectReq))
	if err != nil {
		log.Printf("Failed to send CONNECT to mitmproxy: %v", err)
		return err
	}

	// 读取 mitmproxy 的响应
	buf := make([]byte, 1024)
	n, err := conn.Read(buf)
	if err != nil {
		log.Printf("Failed to read CONNECT response: %v", err)
		return err
	}
	response := string(buf[:n])
	if !strings.Contains(response, "200 Connection") {
		log.Printf("Unexpected CONNECT response: %s", response)
		return fmt.Errorf("CONNECT failed: %s", response)
	}
	return nil
}
