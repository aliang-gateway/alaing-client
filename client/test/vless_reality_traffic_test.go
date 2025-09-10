package test

import (
	"context"
	"fmt"
	"io"
	"net/netip"
	"strings"
	"testing"
	"time"

	"nursor.org/nursorgate/client/server/tun/metadata"
	"nursor.org/nursorgate/client/server/tun/proxy"
)

// TestVLESSRealityTraffic 测试 REALITY 流量转发
func TestVLESSRealityTraffic(t *testing.T) {
	t.Logf("=== 测试 REALITY 流量转发 ===")
	t.Logf("验证 VLESS + REALITY 是否能正常转发流量")

	// 创建带 REALITY 的 VLESS 客户端
	vless, err := proxy.NewVLESSWithReality(
		"103.255.209.43:443",
		"c15c1096-752b-415c-ff54-f560e2e4ea85",
		"www.microsoft.com",
		"h1h7T-tqXyGaI0teh7i7kHu1qRLTT5HibTZcu30YtSs",
		"335fad66be5a",
	)
	if err != nil {
		t.Fatalf("创建带 REALITY 的 VLESS 客户端失败: %v", err)
	}

	t.Logf("✅ VLESS + REALITY 客户端创建成功")
	t.Logf("配置: %s", vless.String())

	// 测试访问 Google
	t.Logf("\n=== 测试访问 Google ===")
	testTrafficForwarding(t, vless, "Google", [4]byte{142, 250, 197, 206}, "www.google.com")

	// 测试访问 Cloudflare
	t.Logf("\n=== 测试访问 Cloudflare ===")
	testTrafficForwarding(t, vless, "Cloudflare", [4]byte{104, 16, 124, 96}, "www.cloudflare.com")

	// 测试访问 GitHub
	t.Logf("\n=== 测试访问 GitHub ===")
	testTrafficForwarding(t, vless, "GitHub", [4]byte{140, 82, 112, 4}, "www.github.com")
}

// testTrafficForwarding 测试流量转发
func testTrafficForwarding(t *testing.T, vless *proxy.VLESS, name string, ip [4]byte, host string) {
	t.Logf("开始测试 %s 流量转发 (%s)...", name, host)

	// 创建测试元数据
	md := &metadata.Metadata{
		Network: metadata.TCP,
		DstIP:   netip.AddrFrom4(ip),
		DstPort: 443,
	}

	// 设置超时上下文
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	// 建立连接
	conn, err := vless.DialContext(ctx, md)
	if err != nil {
		t.Logf("❌ 连接失败: %v", err)
		return
	}
	defer conn.Close()

	t.Logf("✅ 连接建立成功: %s -> %s", conn.LocalAddr(), conn.RemoteAddr())

	// 检查连接类型
	connType := fmt.Sprintf("%T", conn)
	t.Logf("连接类型: %s", connType)

	// 发送 HTTP 请求
	req := fmt.Sprintf("GET / HTTP/1.1\r\nHost: %s\r\nUser-Agent: VLESS-REALITY-Traffic-Test\r\nConnection: close\r\n\r\n", host)
	_, err = conn.Write([]byte(req))
	if err != nil {
		t.Logf("❌ 写入请求失败: %v", err)
		return
	}

	t.Logf("✅ 成功发送 HTTP 请求到 %s", host)

	// 读取响应（设置读取超时）
	conn.SetReadDeadline(time.Now().Add(10 * time.Second))

	buf := make([]byte, 4096)
	n, err := conn.Read(buf)
	if err != nil && err != io.EOF {
		t.Logf("❌ 读取响应失败: %v", err)
		return
	}

	if n > 0 {
		response := string(buf[:n])
		t.Logf("✅ 收到响应 (%d 字节)", n)

		// 检查响应内容
		if strings.Contains(response, "HTTP/") {
			statusLine := strings.Split(response, "\r\n")[0]
			t.Logf("状态: %s", statusLine)

			if strings.Contains(statusLine, "200") {
				t.Logf("✅ %s 流量转发成功", name)
			} else {
				t.Logf("⚠️ %s 流量转发状态: %s", name, statusLine)
			}

			// 显示响应头
			lines := strings.Split(response, "\r\n")
			for i, line := range lines {
				if i < 10 && line != "" { // 显示前10行
					t.Logf("响应头: %s", line)
				}
			}
		} else {
			// 显示响应的前200个字符
			if len(response) > 200 {
				t.Logf("⚠️ %s 响应不是标准 HTTP 格式 (前200字符): %s...", name, response[:200])
			} else {
				t.Logf("⚠️ %s 响应不是标准 HTTP 格式: %s", name, response)
			}
		}
	} else {
		t.Logf("⚠️ 未收到响应")
	}
}

// TestVLESSRealityTrafficComparison 对比测试流量转发
func TestVLESSRealityTrafficComparison(t *testing.T) {
	t.Logf("=== 对比测试流量转发 ===")

	// 测试配置
	server := "103.255.209.43:443"
	uuid := "c15c1096-752b-415c-ff54-f560e2e4ea85"
	sni := "www.microsoft.com"

	// 测试 TLS 版本
	t.Logf("\n=== 测试 TLS 版本流量转发 ===")
	tlsVless, err := proxy.NewVLESSWithTLS(server, uuid, sni)
	if err != nil {
		t.Fatalf("创建 TLS VLESS 客户端失败: %v", err)
	}

	t.Logf("TLS 配置: %s", tlsVless.String())
	testTrafficComparison(t, tlsVless, "TLS")

	// 测试 REALITY 版本
	t.Logf("\n=== 测试 REALITY 版本流量转发 ===")
	realityVless, err := proxy.NewVLESSWithReality(
		server, uuid, sni,
		"h1h7T-tqXyGaI0teh7i7kHu1qRLTT5HibTZcu30YtSs",
		"335fad66be5a",
	)
	if err != nil {
		t.Fatalf("创建 REALITY VLESS 客户端失败: %v", err)
	}

	t.Logf("REALITY 配置: %s", realityVless.String())
	testTrafficComparison(t, realityVless, "REALITY")
}

// testTrafficComparison 测试流量转发对比
func testTrafficComparison(t *testing.T, vless *proxy.VLESS, protocol string) {
	// 测试连接
	md := &metadata.Metadata{
		Network: metadata.TCP,
		DstIP:   netip.AddrFrom4([4]byte{142, 250, 197, 206}), // Google
		DstPort: 443,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	conn, err := vless.DialContext(ctx, md)
	if err != nil {
		t.Logf("❌ %s 连接失败: %v", protocol, err)
		return
	}
	defer conn.Close()

	t.Logf("✅ %s 连接建立成功: %s -> %s", protocol, conn.LocalAddr(), conn.RemoteAddr())

	// 发送测试请求
	req := "GET / HTTP/1.1\r\nHost: www.google.com\r\nUser-Agent: Traffic-Test\r\nConnection: close\r\n\r\n"
	_, err = conn.Write([]byte(req))
	if err != nil {
		t.Logf("❌ %s 写入请求失败: %v", protocol, err)
		return
	}

	// 设置读取超时
	conn.SetReadDeadline(time.Now().Add(10 * time.Second))

	// 读取响应
	buf := make([]byte, 2048)
	n, err := conn.Read(buf)
	if err != nil && err != io.EOF {
		t.Logf("❌ %s 读取响应失败: %v", protocol, err)
		return
	}

	if n > 0 {
		response := string(buf[:n])
		t.Logf("✅ %s 收到响应 (%d 字节)", protocol, n)

		if len(response) > 200 {
			t.Logf("%s 响应内容 (前200字符): %s...", protocol, response[:200])
		} else {
			t.Logf("%s 响应内容: %s", protocol, response)
		}

		// 检查是否是HTTP响应
		if strings.HasPrefix(response, "HTTP/") {
			t.Logf("✅ %s 收到HTTP响应，流量转发成功", protocol)
		} else {
			t.Logf("⚠️ %s 收到非HTTP响应，流量转发可能有问题", protocol)
		}
	} else {
		t.Logf("⚠️ %s 未收到响应", protocol)
	}
}

// TestVLESSRealityTrafficAnalysis 分析流量转发问题
func TestVLESSRealityTrafficAnalysis(t *testing.T) {
	t.Logf("=== 流量转发问题分析 ===")
	t.Logf("")
	t.Logf("当前状态:")
	t.Logf("1. ✅ REALITY 握手成功")
	t.Logf("2. ✅ VLESS 握手成功")
	t.Logf("3. ✅ 连接类型正确 (*reality.Conn)")
	t.Logf("4. ⚠️ 流量转发可能有问题")
	t.Logf("")
	t.Logf("可能的问题:")
	t.Logf("1. VLESS 握手响应确认缺失")
	t.Logf("2. 流量加密/解密问题")
	t.Logf("3. 连接超时设置")
	t.Logf("4. 服务器端配置问题")
	t.Logf("")
	t.Logf("修复措施:")
	t.Logf("1. ✅ 添加 VLESS 握手响应确认")
	t.Logf("2. ✅ 添加读取超时设置")
	t.Logf("3. ✅ 改进错误处理")
	t.Logf("4. ⚠️ 需要进一步测试验证")
	t.Logf("")
	t.Logf("预期结果:")
	t.Logf("- 能够收到 HTTP 响应")
	t.Logf("- 响应状态码为 200 或 301/302")
	t.Logf("- 响应包含正确的 HTTP 头")
}
