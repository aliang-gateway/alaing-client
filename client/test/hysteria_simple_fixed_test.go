package test

import (
	"context"
	"net/netip"
	"testing"
	"time"

	"nursor.org/nursorgate/client/server/tun/metadata"
	"nursor.org/nursorgate/client/server/tun/proxy"
)

// TestHysteriaSimpleFixed 修复版本的 Hysteria2 连接测试
func TestHysteriaSimpleFixed(t *testing.T) {
	t.Logf("=== 修复版 Hysteria2 连接测试 ===")

	// 创建 Hysteria2 客户端
	hysteria, err := proxy.NewHysteriaDialer("", "Y2QuH3NUCv")
	if err != nil {
		t.Fatalf("创建 Hysteria2 客户端失败: %v", err)
	}

	t.Logf("✅ Hysteria2 客户端创建成功")

	// 测试1: 连接到HTTP服务器 (端口80)
	t.Logf("\n=== 测试1: HTTP连接 (端口80) ===")
	mdHTTP := &metadata.Metadata{
		Network: metadata.TCP,
		DstIP:   netip.AddrFrom4([4]byte{142, 250, 197, 206}), // Google.com IP
		DstPort: 80,                                           // HTTP 端口
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	hysteriaConn, err := hysteria.DialContext(ctx, mdHTTP)
	if err != nil {
		t.Logf("❌ Hysteria2 HTTP连接失败: %v", err)
		return
	}
	defer hysteriaConn.Close()

	t.Logf("✅ Hysteria2 HTTP连接成功: %s -> %s", hysteriaConn.LocalAddr(), hysteriaConn.RemoteAddr())

	// 发送HTTP请求
	httpReq := "GET / HTTP/1.1\r\nHost: 142.250.197.206\r\nUser-Agent: Hysteria2-Test\r\nConnection: close\r\n\r\n"
	n, err := hysteriaConn.Write([]byte(httpReq))
	if err != nil {
		t.Logf("❌ 写入HTTP请求失败: %v", err)
		return
	}
	t.Logf("✅ 成功写入 %d 字节HTTP请求", n)

	// 读取响应
	hysteriaConn.SetReadDeadline(time.Now().Add(10 * time.Second))
	buf := make([]byte, 4096)
	n, err = hysteriaConn.Read(buf)
	if err != nil {
		t.Logf("❌ 读取HTTP响应失败: %v", err)
	} else {
		t.Logf("✅ 成功读取 %d 字节HTTP响应", n)
		if n > 0 {
			response := string(buf[:n])
			t.Logf("HTTP响应前200字符: %s", response[:min(len(response), 200)])
		}
	}

	// 注意：HTTPS测试需要TLS握手，请使用 TestHysteriaHTTPS 或 TestHysteriaHTTPSInsecure
	t.Logf("\n=== 注意 ===")
	t.Logf("HTTPS测试需要正确的TLS握手，请运行 TestHysteriaHTTPS 或 TestHysteriaHTTPSInsecure")
}
