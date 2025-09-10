package test

import (
	"context"
	"io"
	"net/netip"
	"strings"
	"testing"
	"time"

	"nursor.org/nursorgate/client/server/tun/metadata"
	"nursor.org/nursorgate/client/server/tun/proxy"
)

// TestVLESSRealityDebug 调试 REALITY 响应
func TestVLESSRealityDebug(t *testing.T) {
	t.Logf("=== 调试 REALITY 响应 ===")
	t.Logf("详细分析 16 字节响应的内容")

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

	// 测试连接
	md := &metadata.Metadata{
		Network: metadata.TCP,
		DstIP:   netip.AddrFrom4([4]byte{142, 250, 197, 206}), // Google
		DstPort: 443,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	// 建立连接
	conn, err := vless.DialContext(ctx, md)
	if err != nil {
		t.Fatalf("连接失败: %v", err)
	}
	defer conn.Close()

	t.Logf("✅ 连接建立成功: %s -> %s", conn.LocalAddr(), conn.RemoteAddr())
	t.Logf("连接类型: %T", conn)

	// 发送 HTTP 请求
	req := "GET / HTTP/1.1\r\nHost: www.google.com\r\nUser-Agent: Debug-Test\r\nConnection: close\r\n\r\n"
	_, err = conn.Write([]byte(req))
	if err != nil {
		t.Fatalf("写入请求失败: %v", err)
	}

	t.Logf("✅ 成功发送 HTTP 请求")

	// 读取响应
	conn.SetReadDeadline(time.Now().Add(10 * time.Second))

	buf := make([]byte, 1024)
	n, err := conn.Read(buf)
	if err != nil && err != io.EOF {
		t.Fatalf("读取响应失败: %v", err)
	}

	if n > 0 {
		responseBytes := buf[:n]
		responseStr := string(responseBytes)

		t.Logf("✅ 收到响应 (%d 字节)", n)
		t.Logf("响应内容 (字符串): %q", responseStr)
		t.Logf("响应内容 (十六进制): %x", responseBytes)
		t.Logf("响应内容 (字节): %v", responseBytes)

		// 分析响应
		if n == 16 {
			t.Logf("🔍 分析 16 字节响应:")
			t.Logf("  前 4 字节: %x", responseBytes[:4])
			t.Logf("  后 4 字节: %x", responseBytes[12:])
			t.Logf("  中间 8 字节: %x", responseBytes[4:12])

			// 检查是否是 TLS 相关
			if responseBytes[0] == 0x15 || responseBytes[0] == 0x16 || responseBytes[0] == 0x17 {
				t.Logf("  可能是 TLS 警报消息")
			}

			// 检查是否是 HTTP 相关
			if strings.HasPrefix(responseStr, "HTTP/") {
				t.Logf("  是 HTTP 响应")
			} else if strings.Contains(responseStr, "error") || strings.Contains(responseStr, "Error") {
				t.Logf("  包含错误信息")
			}
		}

		// 尝试读取更多数据
		t.Logf("\n=== 尝试读取更多数据 ===")
		conn.SetReadDeadline(time.Now().Add(5 * time.Second))

		moreBuf := make([]byte, 1024)
		moreN, moreErr := conn.Read(moreBuf)
		if moreErr != nil && moreErr != io.EOF {
			t.Logf("读取更多数据失败: %v", moreErr)
		} else if moreN > 0 {
			t.Logf("✅ 收到更多数据 (%d 字节): %q", moreN, string(moreBuf[:moreN]))
		} else {
			t.Logf("没有更多数据")
		}
	} else {
		t.Logf("⚠️ 未收到响应")
	}
}

// TestVLESSRealityAnalysis 分析 REALITY 问题
func TestVLESSRealityAnalysis(t *testing.T) {
	t.Logf("=== REALITY 问题分析 ===")
	t.Logf("")
	t.Logf("当前状态:")
	t.Logf("1. ✅ REALITY 握手成功")
	t.Logf("2. ✅ VLESS 握手成功")
	t.Logf("3. ✅ 连接类型正确 (*reality.Conn)")
	t.Logf("4. ✅ 能收到 16 字节响应")
	t.Logf("5. ⚠️ 响应不是 HTTP 格式")
	t.Logf("")
	t.Logf("16 字节响应分析:")
	t.Logf("- 可能是连接关闭信号")
	t.Logf("- 可能是错误消息")
	t.Logf("- 可能是 TLS 警报")
	t.Logf("- 需要分析具体内容")
	t.Logf("")
	t.Logf("可能的原因:")
	t.Logf("1. 服务器端配置问题")
	t.Logf("2. REALITY 协议版本不匹配")
	t.Logf("3. 流量转发逻辑问题")
	t.Logf("4. 需要特定的请求格式")
	t.Logf("")
	t.Logf("建议解决方案:")
	t.Logf("1. 分析 16 字节响应的具体内容")
	t.Logf("2. 检查服务器端日志")
	t.Logf("3. 尝试不同的请求格式")
	t.Logf("4. 参考 sing-box 的行为")
}
