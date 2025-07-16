package test

import (
	"context"
	"github.com/stretchr/testify/assert"
	"io"
	"net"
	"net/http"
	"net/netip"
	http2 "nursor.org/nursorgate/client/inbound/hysteria_forwarding/http"
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

func TestTCPTunnel(t *testing.T) {
	// Start the tunnel
	//l, err := net.Listen("tcp", "127.0.0.1:34567")
	//assert.NoError(t, err)
	//defer l.Close()
	//hysteriaDialer, err := proxy.NewHysteriaDialer("lisi", "IW6gUxtuG46FURELO08p9L9I3GtHtfh1")
	//if err != nil {
	//	t.Fatal(err)
	//}
	//tunnel := &hysteria_forwarding.TCPTunnel{
	//	HyClient: hysteriaDialer.Client,
	//}
	//tunnel.Serve(l)

	//for i := 0; i < 10; i++ {
	//	conn, err := net.Dial("tcp", "127.0.0.1:34567")
	//	assert.NoError(t, err)
	//
	//	data := make([]byte, 1024)
	//	_, _ = rand.Read(data)
	//	_, err = conn.Write(data)
	//	assert.NoError(t, err)
	//
	//	recv := make([]byte, 1024)
	//	_, err = conn.Read(recv)
	//	assert.NoError(t, err)
	//
	//	assert.Equal(t, data, recv)
	//	_ = conn.Close()
	//}
}

func TestHttpServer(t *testing.T) {
	// Start the server
	l, err := net.Listen("tcp", "127.0.0.1:18080")
	assert.NoError(t, err)
	defer l.Close()
	hysteriaDialer, err := proxy.NewHysteriaDialer("lisi", "IW6gUxtuG46FURELO08p9L9I3GtHtfh1")
	if err != nil {
		t.Fatal(err)
	}
	s := &http2.Server{
		HyClient: hysteriaDialer.Client,
	}
	go s.Serve(l)

	// Start a test HTTP & HTTPS server
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("control is an illusion"))
	})
	http.ListenAndServe("127.0.0.1:18081", nil)
	//go http.ListenAndServeTLS("127.0.0.1:18082", testCertFile, testKeyFile, nil)

	// Run the Python test script
	//cmd := exec.Command("python", "server_test.py")
	//// Suppress HTTPS warning text from Python
	//cmd.Env = append(cmd.Env, "PYTHONWARNINGS=ignore:Unverified HTTPS request")
	//out, err := cmd.CombinedOutput()
	//assert.NoError(t, err)
	//assert.Equal(t, "OK", strings.TrimSpace(string(out)))
}
