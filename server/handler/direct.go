package handler

import (
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"net"
	"strings"
	"sync"

	"nursor.org/nursorgate/client/inbound/cert"
	"nursor.org/nursorgate/common/logger"
)

func ForwardHttpDirect(conn net.Conn, host string) {
	targetAddr := host
	port := 443
	if !strings.Contains(host, ":") {
		targetAddr = fmt.Sprintf("%s:%d", targetAddr, port)
	} else {
		host := strings.Split(targetAddr, ":")
		if len(host) > 0 {
			targetAddr = fmt.Sprintf("%s:%d", host[0], port)
		}
	}

	// 连接到目标服务器
	targetConn, err := net.Dial("tcp", targetAddr)
	if err != nil {
		log.Printf("Failed to connect to target %s: %v", targetAddr, err)
		conn.Close()
		return
	}
	tlsConf := cert.CreateTlsConfigForHost(host)
	tlsConn := tls.Client(targetConn, tlsConf)
	if err := tlsConn.Handshake(); err != nil {
		logger.Error("TLS handshake with client failed:", host, err)
		return
	}

	// 确保两个连接在函数退出时关闭
	defer conn.Close()
	defer targetConn.Close()

	// 使用 WaitGroup 等待双向转发完成
	var wg sync.WaitGroup
	wg.Add(2)

	// 从客户端到目标服务器的转发
	go func() {
		defer wg.Done()
		_, err := io.Copy(targetConn, conn)
		if err != nil {
			log.Printf("Error copying from client to target: %v", err)
		}
		// 关闭目标连接的写入方向，通知对端
		if tc, ok := targetConn.(*net.TCPConn); ok {
			tc.CloseWrite()
		}
		// targetConn.CloseWrite()
	}()

	// 从目标服务器到客户端的转发
	go func() {
		defer wg.Done()
		_, err := io.Copy(conn, targetConn)
		if err != nil {
			log.Printf("Error copying from target to client: %v", err)
		}
		// 关闭客户端连接的写入方向
		if cc, ok := conn.(*net.TCPConn); ok {
			cc.CloseWrite()
		}
	}()

	// 等待两个转发完成
	wg.Wait()

}
