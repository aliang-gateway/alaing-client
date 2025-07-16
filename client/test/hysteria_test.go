package test

import (
	"context"
	"io"
	"net/netip"
	"nursor.org/nursorgate/client/server/tun/metadata"
	"nursor.org/nursorgate/client/server/tun/proxy"
	"testing"
	"time"
)

func TestHysteriaToGoogle(t *testing.T) {
	dialer, err := proxy.NewHysteriaDialer("lisi", "IW6gUxtuG46FURELO08p9L9I3GtHtfh1")
	if err != nil {
		t.Fatal("dialer 初始化失败：", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 1000*time.Second)
	defer cancel()

	// 目标地址，标准 HTTP 请求用 80 端口
	metadata := &metadata.Metadata{
		//59.24.3.174
		//142.250.196.196
		//142.250.198.78
		DstIP:   netip.AddrFrom4([4]byte([]byte{142, 250, 198, 78})), // google.com 的其中一个 IP
		DstPort: 80,
	}

	conn, err := dialer.DialContext(ctx, metadata)
	if err != nil {
		t.Fatal("连接失败：", err)
	}
	defer conn.Close()

	// 发起 HTTP 请求
	req := "GET / HTTP/1.1\r\n" +
		"Host: www.youtube.com\r\n" +
		"User-Agent: Hysteria-Test\r\n" +
		"Connection: close\r\n\r\n"

	_, err = conn.Write([]byte(req))
	if err != nil {
		t.Fatal("写入请求失败：", err)
	}

	// 读取响应
	buf := make([]byte, 4096)
	n, err := conn.Read(buf)
	if err != nil && err != io.EOF {
		t.Fatal("读取响应失败：", err)
	}

	t.Logf("收到响应 (%d 字节)：\n%s", n, string(buf[:n]))
}

func NewHysteriaConfig() {

}
