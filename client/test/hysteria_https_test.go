package test

import (
	"context"
	"crypto/tls"
	"net/netip"
	"testing"
	"time"

	"nursor.org/nursorgate/client/server/tun/metadata"
	"nursor.org/nursorgate/client/server/tun/proxy"
)

// TestHysteriaHTTPS 正确的HTTPS连接测试
func TestHysteriaHTTPS(t *testing.T) {
	t.Logf("=== Hysteria2 HTTPS连接测试 ===")

	// 创建 Hysteria2 客户端
	hysteria, err := proxy.NewHysteriaDialer("", "Y2QuH3NUCv")
	if err != nil {
		t.Fatalf("创建 Hysteria2 客户端失败: %v", err)
	}

	t.Logf("✅ Hysteria2 客户端创建成功")

	// 连接到HTTPS服务器
	md := &metadata.Metadata{
		Network: metadata.TCP,
		DstIP:   netip.AddrFrom4([4]byte{142, 250, 198, 36}), // Google.com IP
		DstPort: 443,                                         // HTTPS 端口
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	hysteriaConn, err := hysteria.DialContext(ctx, md)
	if err != nil {
		t.Fatalf("❌ Hysteria2 HTTPS连接失败: %v", err)
	}
	defer hysteriaConn.Close()

	t.Logf("✅ Hysteria2 HTTPS连接成功: %s -> %s", hysteriaConn.LocalAddr(), hysteriaConn.RemoteAddr())

	// 创建TLS连接
	tlsConn := tls.Client(hysteriaConn, &tls.Config{
		ServerName: "www.google.com", // 使用正确的SNI
		// 不跳过证书验证，这样可以测试完整的TLS握手
	})

	// 执行TLS握手
	t.Logf("=== 执行TLS握手 ===")
	err = tlsConn.Handshake()
	if err != nil {
		t.Logf("❌ TLS握手失败: %v", err)
		return
	}
	t.Logf("✅ TLS握手成功")

	// 发送HTTPS请求
	t.Logf("=== 发送HTTPS请求 ===")
	httpsReq := "GET / HTTP/1.1\r\nHost: www.google.com\r\nUser-Agent: Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/140.0.0.0 Safari/537.36 \r\nConnection: close\r\n\r\n"
	n, err := tlsConn.Write([]byte(httpsReq))
	if err != nil {
		t.Logf("❌ 写入HTTPS请求失败: %v", err)
		return
	}
	t.Logf("✅ 成功写入 %d 字节HTTPS请求", n)

	// 读取响应
	t.Logf("=== 读取HTTPS响应 ===")
	tlsConn.SetReadDeadline(time.Now().Add(10 * time.Second))
	buf := make([]byte, 4096)
	n, err = tlsConn.Read(buf)
	if err != nil {
		t.Logf("❌ 读取HTTPS响应失败: %v", err)
		return
	}
	t.Logf("✅ 成功读取 %d 字节HTTPS响应", n)
	if n > 0 {
		response := string(buf[:n])
		t.Logf("HTTPS响应前200字符: %s", response)
	}
}

// TestHysteriaHTTPSInsecure 跳过证书验证的HTTPS测试
func TestHysteriaHTTPSInsecure(t *testing.T) {
	t.Logf("=== Hysteria2 HTTPS连接测试 (跳过证书验证) ===")

	// 创建 Hysteria2 客户端
	hysteria, err := proxy.NewHysteriaDialer("", "Y2QuH3NUCv")
	if err != nil {
		t.Fatalf("创建 Hysteria2 客户端失败: %v", err)
	}

	t.Logf("✅ Hysteria2 客户端创建成功")

	// 连接到HTTPS服务器
	md := &metadata.Metadata{
		Network: metadata.TCP,
		DstIP:   netip.AddrFrom4([4]byte{142, 250, 197, 206}), // Google.com IP
		DstPort: 443,                                          // HTTPS 端口
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	hysteriaConn, err := hysteria.DialContext(ctx, md)
	if err != nil {
		t.Fatalf("❌ Hysteria2 HTTPS连接失败: %v", err)
	}
	defer hysteriaConn.Close()

	t.Logf("✅ Hysteria2 HTTPS连接成功: %s -> %s", hysteriaConn.LocalAddr(), hysteriaConn.RemoteAddr())

	// 创建TLS连接，跳过证书验证
	tlsConn := tls.Client(hysteriaConn, &tls.Config{
		ServerName:         "www.google.com",
		InsecureSkipVerify: true, // 跳过证书验证
	})

	// 执行TLS握手
	t.Logf("=== 执行TLS握手 (跳过证书验证) ===")
	err = tlsConn.Handshake()
	if err != nil {
		t.Logf("❌ TLS握手失败: %v", err)
		return
	}
	t.Logf("✅ TLS握手成功")

	// 发送HTTPS请求
	t.Logf("=== 发送HTTPS请求 ===")
	httpsReq := "GET / HTTP/1.1\r\nHost: www.google.com\r\nUser-Agent: Hysteria2-Test\r\nConnection: close\r\n\r\n"
	n, err := tlsConn.Write([]byte(httpsReq))
	if err != nil {
		t.Logf("❌ 写入HTTPS请求失败: %v", err)
		return
	}
	t.Logf("✅ 成功写入 %d 字节HTTPS请求", n)

	// 读取响应
	t.Logf("=== 读取HTTPS响应 ===")
	tlsConn.SetReadDeadline(time.Now().Add(10 * time.Second))
	buf := make([]byte, 4096)
	n, err = tlsConn.Read(buf)
	if err != nil {
		t.Logf("❌ 读取HTTPS响应失败: %v", err)
		return
	}
	t.Logf("✅ 成功读取 %d 字节HTTPS响应", n)
	if n > 0 {
		response := string(buf[:n])
		t.Logf("HTTPS响应前200字符: %s", response[:min(len(response), 200)])
	}
}
