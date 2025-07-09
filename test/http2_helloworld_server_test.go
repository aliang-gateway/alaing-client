package test

import (
	"crypto/tls"
	"fmt"
	"log"
	"net/http"
	"testing"
	"time"

	"golang.org/x/net/http2"
)

func TestRunServer(t *testing.T) {
	// 创建一个简单的 HTTP 处理函数
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// 检查是否支持 Flusher
		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "Streaming unsupported!", http.StatusInternalServerError)
			return
		}

		// 设置头部
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")

		// 持续写入数据
		for i := 0; i < 10; i++ {
			fmt.Fprintf(w, "Message %d\n", i)
			flusher.Flush()             // 立即发送数据到客户端
			time.Sleep(1 * time.Second) // 模拟处理时间
		}
	})

	// 创建一个 TLS 配置，禁用 HTTP/1.1
	tlsConfig := &tls.Config{
		NextProtos: []string{http2.NextProtoTLS}, // 仅启用 HTTP/2
	}

	// 创建一个 HTTP/2 服务器
	server := &http.Server{
		Addr:      ":7788",
		TLSConfig: tlsConfig,
	}

	// 启用 HTTP/2 支持
	if err := http2.ConfigureServer(server, nil); err != nil {
		log.Fatalf("Failed to configure HTTP/2 server: %v", err)
	}

	// 启动 HTTPS 服务器
	log.Println("Starting HTTPS server on :7788")
	if err := server.ListenAndServeTLS("../proxy_cert.pem", "../proxy_key.pem"); err != nil {
		log.Fatalf("Failed to start HTTPS server: %v", err)
	}
}
