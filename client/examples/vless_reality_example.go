package main

import (
	"context"
	"fmt"
	"io"
	"net/netip"
	"time"

	"nursor.org/nursorgate/client/server/tun/metadata"
	"nursor.org/nursorgate/client/server/tun/proxy"
)

func main() {
	fmt.Println("=== VLESS + REALITY 示例程序 ===")

	// 创建带 REALITY 的 VLESS 客户端
	vless, err := proxy.NewVLESSWithReality(
		"103.255.209.43:443",
		"c15c1096-752b-415c-ff54-f560e2e4ea85",
		"www.microsoft.com",
		"h1h7T-tqXyGaI0teh7i7kHu1qRLTT5HibTZcu30YtSs",
		"335fad66be5a",
	)
	if err != nil {
		fmt.Printf("创建 VLESS 客户端失败: %v\n", err)
		return
	}

	fmt.Printf("✅ VLESS + REALITY 客户端创建成功\n")
	fmt.Printf("配置: %s\n", vless.String())

	// 测试访问 Google
	fmt.Println("\n=== 测试访问 Google ===")
	testAccess(vless, "Google", [4]byte{142, 250, 197, 206}, "www.google.com")

	// 测试访问 Cloudflare
	fmt.Println("\n=== 测试访问 Cloudflare ===")
	testAccess(vless, "Cloudflare", [4]byte{104, 16, 124, 96}, "www.cloudflare.com")

	// 测试访问 GitHub
	fmt.Println("\n=== 测试访问 GitHub ===")
	testAccess(vless, "GitHub", [4]byte{140, 82, 112, 4}, "www.github.com")
}

func testAccess(vless *proxy.VLESS, name string, ip [4]byte, host string) {
	fmt.Printf("开始访问 %s (%s)...\n", name, host)

	// 创建测试元数据
	md := &metadata.Metadata{
		Network: metadata.TCP,
		DstIP:   netip.AddrFrom4(ip),
		DstPort: 443,
	}

	// 设置超时上下文
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// 建立连接
	conn, err := vless.DialContext(ctx, md)
	if err != nil {
		fmt.Printf("❌ 连接失败: %v\n", err)
		return
	}
	defer conn.Close()

	fmt.Printf("✅ 连接建立成功: %s -> %s\n", conn.LocalAddr(), conn.RemoteAddr())

	// 发送 HTTP 请求
	req := fmt.Sprintf("GET / HTTP/1.1\r\nHost: %s\r\nUser-Agent: VLESS-REALITY-Example\r\nConnection: close\r\n\r\n", host)
	_, err = conn.Write([]byte(req))
	if err != nil {
		fmt.Printf("❌ 写入请求失败: %v\n", err)
		return
	}

	fmt.Printf("✅ 成功发送 HTTP 请求到 %s\n", host)

	// 读取响应
	buf := make([]byte, 2048)
	n, err := conn.Read(buf)
	if err != nil && err != io.EOF {
		fmt.Printf("❌ 读取响应失败: %v\n", err)
		return
	}

	if n > 0 {
		response := string(buf[:n])
		fmt.Printf("✅ 收到响应 (%d 字节)\n", n)

		// 显示响应的前几行
		lines := 0
		for i, char := range response {
			if char == '\n' {
				lines++
				if lines >= 5 {
					fmt.Printf("响应内容 (前5行):\n%s...\n", response[:i])
					break
				}
			}
		}
		if lines < 5 {
			fmt.Printf("响应内容:\n%s\n", response)
		}
	} else {
		fmt.Printf("⚠️ 未收到响应\n")
	}
}
