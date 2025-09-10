package test

import (
	"context"
	"net"
	"net/netip"
	"testing"
	"time"

	"nursor.org/nursorgate/client/server/tun/metadata"
	"nursor.org/nursorgate/client/server/tun/proxy"
)

// TestVLESSSimple 简单测试 VLESS 连接
func TestVLESSSimple(t *testing.T) {
	t.Logf("=== 简单 VLESS 连接测试 ===")

	// 创建带 REALITY 的 VLESS 客户端
	vless, err := proxy.NewVLESSWithReality(
		"103.255.209.43:443",
		"c15c1096-752b-415c-ff54-f560e2e4ea85",
		"www.microsoft.com",
		"h1h7T-tqXyGaI0teh7i7kHu1qRLTT5HibTZcu30YtSs",
		"335fad66be5a",
	)
	if err != nil {
		t.Fatalf("创建 VLESS 客户端失败: %v", err)
	}

	t.Logf("✅ VLESS 客户端创建成功")

	// 测试基本 TCP 连接
	t.Logf("\n=== 测试基本 TCP 连接 ===")
	conn, err := net.DialTimeout("tcp", "103.255.209.43:443", 10*time.Second)
	if err != nil {
		t.Fatalf("TCP 连接失败: %v", err)
	}
	defer conn.Close()

	t.Logf("✅ TCP 连接成功: %s -> %s", conn.LocalAddr(), conn.RemoteAddr())

	// 测试 VLESS 连接
	t.Logf("\n=== 测试 VLESS 连接 ===")
	md := &metadata.Metadata{
		Network: metadata.TCP,
		DstIP:   netip.AddrFrom4([4]byte{142, 250, 197, 206}), // Google
		DstPort: 443,                                          // HTTPS 端口
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	vlessConn, err := vless.DialContext(ctx, md)
	if err != nil {
		t.Logf("❌ VLESS 连接失败: %v", err)
		return
	}
	defer vlessConn.Close()

	t.Logf("✅ VLESS 连接成功: %s -> %s", vlessConn.LocalAddr(), vlessConn.RemoteAddr())
	t.Logf("连接类型: %T", vlessConn)

	// 测试发送 HTTP 请求
	t.Logf("\n=== 测试发送 HTTP 请求 ===")
	httpReq := "GET / HTTP/1.1\r\nHost: www.google.com\r\nUser-Agent: VLESS-Test\r\nConnection: close\r\n\r\n"
	n, err := vlessConn.Write([]byte(httpReq))
	if err != nil {
		t.Logf("❌ 写入 HTTP 请求失败: %v", err)
		return
	}
	t.Logf("✅ 成功写入 %d 字节 HTTP 请求", n)

	// 测试读取数据
	t.Logf("\n=== 测试读取数据 ===")
	vlessConn.SetReadDeadline(time.Now().Add(5 * time.Second))
	buf := make([]byte, 1024)
	n, err = vlessConn.Read(buf)
	if err != nil {
		t.Logf("❌ 读取数据失败: %v", err)
		return
	}
	t.Logf("✅ 成功读取 %d 字节数据", n)
	if n > 0 {
		t.Logf("数据内容: %s", string(buf[:n]))
	}
}
