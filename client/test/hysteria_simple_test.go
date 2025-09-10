package test

import (
	"context"
	"net/netip"
	"testing"
	"time"

	"nursor.org/nursorgate/client/server/tun/metadata"
	"nursor.org/nursorgate/client/server/tun/proxy"
)

// TestHysteriaSimple 简单测试 Hysteria2 连接
func TestHysteriaSimple(t *testing.T) {
	t.Logf("=== 简单 Hysteria2 连接测试 ===")

	// 创建 Hysteria2 客户端
	// 根据你的配置：password="Y2QuH3NUCv", obfs.password="2hKDWT79uWNIJuRMS5jqFNyOtSIf05Oc"
	hysteria, err := proxy.NewHysteriaDialer("", "Y2QuH3NUCv") // 用户名留空，只使用密码
	if err != nil {
		t.Fatalf("创建 Hysteria2 客户端失败: %v", err)
	}

	t.Logf("✅ Hysteria2 客户端创建成功")

	// 测试 Hysteria2 连接
	t.Logf("\n=== 测试 Hysteria2 连接 ===")
	md := &metadata.Metadata{
		Network: metadata.TCP,
		DstIP:   netip.AddrFrom4([4]byte{142, 250, 197, 206}), // Google.com IP
		DstPort: 443,                                          // HTTPS 端口
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	hysteriaConn, err := hysteria.DialContext(ctx, md)
	if err != nil {
		t.Logf("❌ Hysteria2 连接失败: %v", err)
		return
	}
	defer hysteriaConn.Close()

	t.Logf("✅ Hysteria2 连接成功: %s -> %s", hysteriaConn.LocalAddr(), hysteriaConn.RemoteAddr())
	t.Logf("连接类型: %T", hysteriaConn)

	// 测试发送 HTTP 请求
	t.Logf("\n=== 测试发送 HTTP 请求 ===")
	httpReq := "GET / HTTP/1.1\r\nHost: 142.250.197.206\r\nUser-Agent: Hysteria2-Test\r\nConnection: close\r\n\r\n"
	n, err := hysteriaConn.Write([]byte(httpReq))
	if err != nil {
		t.Logf("❌ 写入 HTTP 请求失败: %v", err)
		return
	}
	t.Logf("✅ 成功写入 %d 字节 HTTP 请求", n)

	// 测试读取数据
	t.Logf("\n=== 测试读取数据 ===")
	hysteriaConn.SetReadDeadline(time.Now().Add(10 * time.Second))
	buf := make([]byte, 4096)
	n, err = hysteriaConn.Read(buf)
	if err != nil {
		t.Logf("❌ 读取数据失败: %v", err)
		return
	}
	t.Logf("✅ 成功读取 %d 字节数据", n)
	if n > 0 {
		response := string(buf[:n])
		t.Logf("响应前200字符: %s", response[:min(len(response), 200)])
	}
}
