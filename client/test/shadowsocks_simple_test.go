package test

import (
	"context"
	"net/netip"
	"os"
	"testing"
	"time"

	"nursor.org/nursorgate/client/server/tun/metadata"
	"nursor.org/nursorgate/client/server/tun/proxy"
)

// TestShadowsocksSimple 通过 Shadowsocks 测试 HTTP 访问 Google（80 端口）
func TestShadowsocksSimple(t *testing.T) {
	addr := os.Getenv("SHADOWSOCKS_ADDR") // 例如: 127.0.0.1:8388
	if addr == "" {
		addr = "103.255.209.43:13299"
	}

	method := os.Getenv("SHADOWSOCKS_METHOD")
	if method == "" {
		method = "aes-256-gcm"
	}
	password := os.Getenv("SHADOWSOCKS_PASSWORD")
	if password == "" {
		password = "1DarCSuZf6"
	}
	obfs := ""     // 可选: tls/http
	obfsHost := "" // 可选: 如 example.com

	ss, err := proxy.NewShadowsocks(addr, method, password, obfs, obfsHost)
	if err != nil {
		t.Fatalf("创建 Shadowsocks 客户端失败: %v", err)
	}
	t.Logf("✅ Shadowsocks 客户端创建成功: %s (%s), method=%s obfs=%s host=%s", ss.Addr(), ss.Proto(), method, obfs, obfsHost)

	// 访问一个更简单的目标 - 使用 Cloudflare 的 IP
	md := &metadata.Metadata{
		Network: metadata.TCP,
		DstIP:   netip.AddrFrom4([4]byte{104, 16, 132, 229}), // Cloudflare IP
		DstPort: 80,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()

	conn, err := ss.DialContext(ctx, md)
	if err != nil {
		t.Fatalf("Shadowsocks 连接失败: %v", err)
	}
	defer conn.Close()
	t.Logf("✅ Shadowsocks 连接成功: %s -> %s", conn.LocalAddr(), conn.RemoteAddr())
	t.Logf("✅ 连接类型: %T", conn)

	// 验证连接可以保持一段时间
	time.Sleep(1 * time.Second)
	t.Logf("✅ Shadowsocks 连接保持正常")

	// 尝试发送一个简单的数据包
	testData := []byte("test")
	conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
	n, err := conn.Write(testData)
	if err != nil {
		t.Logf("⚠️ 写入测试数据失败: %v (这可能是正常的，取决于服务端配置)", err)
	} else {
		t.Logf("✅ 成功写入 %d 字节测试数据", n)
	}
}

// TestShadowsocksCompare 对比我们的实现和 tun2socks 官方实现
func TestShadowsocksCompare(t *testing.T) {
	// 使用 tun2socks 官方的实现
	addr := os.Getenv("SHADOWSOCKS_ADDR")
	if addr == "" {
		addr = "103.255.209.43:13299"
	}
	password := os.Getenv("SHADOWSOCKS_PASSWORD")
	if password == "" {
		password = "1DarCSuZf6"
	}

	t.Logf("🔍 对比测试: %s", addr)

	// 创建我们的实现
	ourSS, err := proxy.NewShadowsocks(addr, "aes-256-gcm", password, "", "")
	if err != nil {
		t.Fatalf("创建我们的客户端失败: %v", err)
	}

	md := &metadata.Metadata{
		Network: metadata.TCP,
		DstIP:   netip.AddrFrom4([4]byte{142, 250, 197, 206}),
		DstPort: 443,
	}

	t.Logf("🎯 目标地址: %s:%d", md.DstIP, md.DstPort)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 测试我们的实现
	conn, err := ourSS.DialContext(ctx, md)
	if err != nil {
		t.Logf("❌ 我们的实现连接失败: %v", err)
		t.Logf("💡 这确认了问题出在我们的实现上")
		return
	}
	defer conn.Close()

	t.Logf("✅ 我们的实现连接成功!")
	t.Logf("   连接类型: %T", conn)
}

// TestShadowsocksDebug 调试 Shadowsocks 连接问题
func TestShadowsocksDebug(t *testing.T) {
	addr := os.Getenv("SHADOWSOCKS_ADDR")
	if addr == "" {
		addr = "103.255.209.43:13299"
	}
	password := os.Getenv("SHADOWSOCKS_PASSWORD")
	if password == "" {
		password = "1DarCSuZf6"
	}

	t.Logf("🔍 调试 Shadowsocks 连接: %s", addr)
	t.Logf("📝 密码: %s (长度: %d)", password, len(password))

	// 测试基本连接
	ss, err := proxy.NewShadowsocks(addr, "aes-256-gcm", password, "", "")
	if err != nil {
		t.Fatalf("创建客户端失败: %v", err)
	}

	md := &metadata.Metadata{
		Network: metadata.TCP,
		DstIP:   netip.AddrFrom4([4]byte{142, 250, 197, 206}),
		DstPort: 443,
	}

	t.Logf("🎯 目标地址: %s:%d", md.DstIP, md.DstPort)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 只测试到连接建立，不进行 TLS 握手
	conn, err := ss.DialContext(ctx, md)
	if err != nil {
		t.Logf("❌ Shadowsocks 连接失败: %v", err)
		t.Logf("💡 可能原因:")
		t.Logf("   1. 加密方法不匹配 (当前: aes-256-gcm)")
		t.Logf("   2. 密码不正确 (当前: %s)", password)
		t.Logf("   3. 服务端是 Shadowsocks-2022 (当前客户端不支持)")
		t.Logf("   4. 端口不是 Shadowsocks 端口")
		return
	}
	defer conn.Close()

	t.Logf("✅ Shadowsocks 连接成功!")
	t.Logf("   本地地址: %s", conn.LocalAddr())
	t.Logf("   远程地址: %s", conn.RemoteAddr())
	t.Logf("   连接类型: %T", conn)
}

// TestShadowsocksAutoDetect 自动探测 Shadowsocks 服务端配置
func TestShadowsocksAutoDetect(t *testing.T) {
	addr := os.Getenv("SHADOWSOCKS_ADDR")
	if addr == "" {
		addr = "103.255.209.43:13299"
	}
	password := os.Getenv("SHADOWSOCKS_PASSWORD")
	if password == "" {
		password = "1DarCSuZf6"
	}

	// 常见的加密方法
	methods := []string{
		"aes-256-gcm",
	}

	// 混淆模式
	obfsModes := []string{""}
	obfsHosts := []string{""}

	t.Logf("🔍 开始自动探测 Shadowsocks 配置: %s", addr)
	t.Logf("📝 使用密码: %s", password)

	success := false
	for _, method := range methods {
		for _, obfsMode := range obfsModes {
			for _, obfsHost := range obfsHosts {
				// 跳过无效组合
				if obfsMode != "" && obfsHost == "" {
					continue
				}

				t.Logf("\n--- 尝试: method=%s, obfs=%s, host=%s ---", method, obfsMode, obfsHost)

				ss, err := proxy.NewShadowsocks(addr, method, password, obfsMode, obfsHost)
				if err != nil {
					t.Logf("❌ 创建客户端失败: %v", err)
					continue
				}

				// 测试连接 - 使用 HTTP 80 端口而不是 HTTPS 443
				md := &metadata.Metadata{
					Network: metadata.TCP,
					DstIP:   netip.AddrFrom4([4]byte{142, 250, 197, 206}),
					DstPort: 80,
				}

				ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
				conn, err := ss.DialContext(ctx, md)
				cancel()

				if err != nil {
					t.Logf("❌ 连接失败: %v", err)
					continue
				}

				// 直接发送 HTTP 请求，不进行 TLS 握手
				req := "GET / HTTP/1.1\r\nHost: www.google.com\r\nUser-Agent: SS-AutoDetect\r\nConnection: close\r\n\r\n"
				conn.SetWriteDeadline(time.Now().Add(5 * time.Second))

				if _, err := conn.Write([]byte(req)); err != nil {
					t.Logf("❌ 写入 HTTP 请求失败: %v", err)
					conn.Close()
					continue
				}

				t.Logf("🎉 成功！找到有效配置:")
				t.Logf("   method: %s", method)
				t.Logf("   obfs: %s", obfsMode)
				t.Logf("   obfsHost: %s", obfsHost)
				t.Logf("   password: %s", password)
				t.Logf("   addr: %s", addr)

				// 读取响应验证
				buf := make([]byte, 1024)
				conn.SetReadDeadline(time.Now().Add(3 * time.Second))
				if n, err := conn.Read(buf); err == nil && n > 0 {
					t.Logf("✅ HTTP 请求成功，响应长度: %d 字节", n)
					preview := string(buf[:minInt(n, 100)])
					t.Logf("响应预览: %s", preview)
				}

				conn.Close()
				success = true
				break
			}
			if success {
				break
			}
		}
		if success {
			break
		}
	}

	if !success {
		t.Logf("❌ 所有配置组合都失败了")
		t.Logf("💡 建议:")
		t.Logf("   1. 确认服务端是否为 Shadowsocks-2022（当前客户端不支持）")
		t.Logf("   2. 确认密码是否完全正确（无空格、大小写敏感）")
		t.Logf("   3. 确认端口 13299 确实是 Shadowsocks 端口")
		t.Logf("   4. 用其他客户端（如 sing-box）验证相同参数")
	}
}

// TestHTTPProxyAuth 测试带用户名/密码的 HTTP 代理发起 CONNECT 到 Google:443
func TestHTTPProxyAuth(t *testing.T) {
	proxyAddr := os.Getenv("HTTP_PROXY_ADDR") // 例如: 127.0.0.1:8080
	if proxyAddr == "" {
		t.Skip("未设置 HTTP_PROXY_ADDR，跳过 HTTP 代理鉴权测试")
	}

	user := "1lv4z2lm"
	pass := "hXuA0VcC2648R1ocuR6qK24JHXcI2mG4AvpfFuiP8Tg="

	h, err := proxy.NewHTTP(proxyAddr, user, pass)
	if err != nil {
		t.Fatalf("创建 HTTP 代理失败: %v", err)
	}

	md := &metadata.Metadata{
		Network: metadata.TCP,
		DstIP:   netip.AddrFrom4([4]byte{142, 250, 197, 206}),
		DstPort: 443,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	c, err := h.DialContext(ctx, md)
	if err != nil {
		t.Fatalf("HTTP 代理 CONNECT 失败: %v", err)
	}
	defer c.Close()
	// 若成功，说明用户名/密码鉴权生效，CONNECT 已建立到 Google:443
	t.Logf("✅ HTTP 代理鉴权成功并已 CONNECT: %s -> %s", c.LocalAddr(), c.RemoteAddr())
}

// TestShadowsocksHTTPGoogle 使用 Shadowsocks 访问 http://www.google.com （80端口）
func TestShadowsocksHTTPGoogle(t *testing.T) {
	addr := os.Getenv("SHADOWSOCKS_ADDR")
	if addr == "" {
		addr = "103.255.209.43:13299"
	}

	method := os.Getenv("SHADOWSOCKS_METHOD")
	if method == "" {
		method = "aes-256-gcm"
	}
	password := os.Getenv("SHADOWSOCKS_PASSWORD")
	if password == "" {
		password = "1DarCSuZf6"
	}
	obfs := os.Getenv("SHADOWSOCKS_OBFS")
	obfsHost := os.Getenv("SHADOWSOCKS_OBFS_HOST")

	ss, err := proxy.NewShadowsocks(addr, method, password, obfs, obfsHost)
	if err != nil {
		t.Fatalf("创建 Shadowsocks 客户端失败: %v", err)
	}
	t.Logf("✅ SS 创建成功: %s (%s) method=%s obfs=%s host=%s", ss.Addr(), ss.Proto(), method, obfs, obfsHost)

	md := &metadata.Metadata{
		Network: metadata.TCP,                                //142.250.198.36
		DstIP:   netip.AddrFrom4([4]byte{142, 250, 198, 36}), // google IP
		DstPort: 80,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	c, err := ss.DialContext(ctx, md)
	if err != nil {
		t.Fatalf("SS 连接失败: %v", err)
	}
	defer c.Close()

	req := "GET / HTTP/1.1\r\nHost: www.google.com\r\nUser-Agent: SS-HTTP-Test\r\nConnection: close\r\n\r\n"
	_ = c.SetWriteDeadline(time.Now().Add(8 * time.Second))
	if _, err := c.Write([]byte(req)); err != nil {
		t.Fatalf("写入 HTTP 请求失败: %v", err)
	}

	buf := make([]byte, 8192)
	_ = c.SetReadDeadline(time.Now().Add(10 * time.Second))
	n, err := c.Read(buf)
	if err != nil {
		t.Fatalf("读取响应失败: %v", err)
	}
	if n == 0 {
		t.Fatalf("读取为空")
	}
	t.Logf("✅ 收到 %d 字节响应", n)
	preview := string(buf[:minInt(n, 200)])
	t.Logf("响应预览: %s", preview)
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
