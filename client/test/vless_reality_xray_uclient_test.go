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

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// TestVLESSRealityXrayUClient 测试基于 Xray-core UClient 方法的 REALITY 实现
func TestVLESSRealityXrayUClient(t *testing.T) {
	t.Logf("=== 测试基于 Xray-core UClient 方法的 REALITY 实现 ===")
	t.Logf("验证直接使用 Xray-core 的 UClient 方法")

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

	t.Logf("✅ VLESS + REALITY + Xray UClient 客户端创建成功")
	t.Logf("配置: %s", vless.String())

	// 测试访问 Google
	t.Logf("\n=== 测试访问 Google ===")
	testRealityXrayUClient(t, vless, "Google", [4]byte{142, 250, 197, 206}, "www.google.com")
}

// testRealityXrayUClient 测试基于 Xray-core UClient 方法的 REALITY
func testRealityXrayUClient(t *testing.T, vless *proxy.VLESS, name string, ip [4]byte, host string) {
	t.Logf("开始测试 %s 流量转发 (%s)...", name, host)

	// 创建测试元数据
	md := &metadata.Metadata{
		Network: metadata.TCP,
		DstIP:   netip.AddrFrom4(ip),
		DstPort: 443,
		// 添加目标域名信息
		// 注意：这里需要扩展 metadata 结构来支持目标域名
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
	t.Logf("连接类型: %T", conn)

	// 先测试连接是否真的建立成功
	t.Logf("🔍 测试连接状态...")

	// 发送一个简单的测试请求
	testReq := "GET /generate_204 HTTP/1.1\r\nHost: www.google.com\r\nUser-Agent: Test\r\nConnection: close\r\n\r\n"
	_, err = conn.Write([]byte(testReq))
	if err != nil {
		t.Logf("❌ 测试请求写入失败: %v", err)
		return
	}
	t.Logf("✅ 测试请求发送成功")

	// 设置较短的超时来快速测试
	conn.SetReadDeadline(time.Now().Add(5 * time.Second))

	// 尝试读取响应
	testBuf := make([]byte, 1024)
	n, err := conn.Read(testBuf)
	if err != nil {
		t.Logf("❌ 测试读取失败: %v", err)
	} else {
		t.Logf("✅ 测试读取成功: %d 字节", n)
		if n > 0 {
			t.Logf("测试响应前100字符: %s", string(testBuf[:min(n, 100)]))
		}
	}

	// 重置连接状态，准备正式测试
	conn.SetReadDeadline(time.Time{}) // 清除超时

	// 直接发送 HTTP 请求（VLESS+REALITY+Vision 已经处理了 TLS 层）
	req := fmt.Sprintf("GET / HTTP/1.1\r\nHost: %s\r\nUser-Agent: VLESS-REALITY-Xray-UClient-Test\r\nConnection: close\r\n\r\n", host)
	_, err = conn.Write([]byte(req))
	if err != nil {
		t.Logf("❌ 写入请求失败: %v", err)
		return
	}

	t.Logf("✅ 成功发送 HTTP 请求到 %s", host)

	// 读取响应（设置读取超时）
	conn.SetReadDeadline(time.Now().Add(20 * time.Second))

	buf := make([]byte, 4096)
	total := 0
	for {
		n, err := conn.Read(buf)
		if n > 0 {
			total += n
			t.Logf("✅ 读取到数据块 (%d 字节)", n)
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Logf("❌ 读取响应失败: %v", err)
			return
		}
	}

	if total > 0 {
		response := string(buf[:total])
		t.Logf("✅ 总共收到响应 (%d 字节)", total)

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
		t.Logf("⚠️ 未收到任何响应数据")
	}
}
