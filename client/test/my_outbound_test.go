package test

import (
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"testing"

	"golang.org/x/net/http2"
	"nursor.org/nursorgate/client/outbound"
)

func TestOutbound(t *testing.T) {
	outbound.SetOutboundToken("123321")
	outbound, err := outbound.NewHttp2ProxyClient("localhost:8082", "", true)
	if err != nil {
		t.Fatal(err)
	}

	defer outbound.Close()

	outbound.Write([]byte("GET / HTTP/2.0\r\nHost: www.baidu.com\r\n\r\n"))
	buf := make([]byte, 1024)
	n, err := outbound.Read(buf)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(string(buf[:n]))
}

func TestOutbound2(t *testing.T) {
	proxyURL, err := url.Parse("http://localhost:5643") // 替换为你的代理地址和端口
	if err != nil {
		log.Fatal("Invalid proxy URL:", err)
	}

	// 创建自定义的 Transport
	transport := &http.Transport{
		Proxy:           http.ProxyURL(proxyURL),
		TLSClientConfig: &tls.Config{
			// 如果需要跳过证书验证，可以设置为 true
			// InsecureSkipVerify: true,
		},
	}

	// 配置 HTTP/2 支持
	err = http2.ConfigureTransport(transport)
	if err != nil {
		log.Fatal("Failed to configure HTTP/2:", err)
	}

	// 创建 HTTP 客户端
	client := &http.Client{
		Transport: transport,
	}

	// 目标 URL
	targetURL := "https://www.zhihu.com" // 替换为你要请求的地址

	// 创建请求
	req, err := http.NewRequest("GET", targetURL, nil)
	if err != nil {
		log.Fatal("Failed to create request:", err)
	}

	// 设置请求头（可选）
	req.Header.Set("User-Agent", "Golang-HTTP2-Client/1.0")

	// 发送请求
	resp, err := client.Do(req)
	if err != nil {
		log.Fatal("Failed to send request:", err)
	}
	defer resp.Body.Close()

	// 检查是否使用 HTTP/2
	if resp.ProtoMajor == 2 {
		fmt.Println("Using HTTP/2")
	} else {
		fmt.Println("Not using HTTP/2, got:", resp.Proto)
	}

	// 读取响应内容
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatal("Failed to read response:", err)
	}

	// 输出响应状态和内容
	fmt.Printf("Status: %s\n", resp.Status)
	fmt.Printf("Body: %s\n", string(body))
}
