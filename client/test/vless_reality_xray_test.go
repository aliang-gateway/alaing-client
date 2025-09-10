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

// TestVLESSRealityXray 测试基于 Xray-core 思路的 REALITY 实现
func TestVLESSRealityXray(t *testing.T) {
	t.Logf("=== 测试基于 Xray-core 思路的 REALITY 实现 ===")
	t.Logf("验证是否能避免 'processed invalid connection' 错误")

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

	t.Logf("✅ VLESS + REALITY + Xray 客户端创建成功")
	t.Logf("配置: %s", vless.String())

	// 测试访问 Google
	t.Logf("\n=== 测试访问 Google ===")
	testRealityXray(t, vless, "Google", [4]byte{142, 250, 197, 206}, "www.google.com")
}

// testRealityXray 测试基于 Xray-core 思路的 REALITY
func testRealityXray(t *testing.T, vless *proxy.VLESS, name string, ip [4]byte, host string) {
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
	t.Logf("连接类型: %T", conn)

	// 发送 HTTP 请求
	req := fmt.Sprintf("GET / HTTP/1.1\r\nHost: %s\r\nUser-Agent: VLESS-REALITY-Xray-Test\r\nConnection: close\r\n\r\n", host)
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

// TestVLESSRealityXrayComparison 对比测试
func TestVLESSRealityXrayComparison(t *testing.T) {
	t.Logf("=== 对比测试：REALITY + Xray 思路 vs 标准实现 ===")
	t.Logf("")
	t.Logf("测试目标:")
	t.Logf("1. 基于 Xray-core 的 REALITY 实现思路")
	t.Logf("2. 使用 SessionTicketsDisabled: true")
	t.Logf("3. 避免 'processed invalid connection' 错误")
	t.Logf("4. 验证流量转发功能")
	t.Logf("")
	t.Logf("修改内容:")
	t.Logf("1. 添加 SessionTicketsDisabled: true")
	t.Logf("2. 基于 Xray-core 的配置思路")
	t.Logf("3. 保持相同的握手流程")
	t.Logf("")
	t.Logf("预期结果:")
	t.Logf("- 服务端不再报告 'processed invalid connection'")
	t.Logf("- 能够正常转发流量")
	t.Logf("- 收到 HTTP 响应")
	t.Logf("- 连接类型为 *tls.UConn")
}
