package test

import (
	"net"
	"net/http"
	"nursor.org/nursorgate/client/inbound/hysteria_forwarding"
	http2 "nursor.org/nursorgate/client/inbound/hysteria_forwarding/http"
	"nursor.org/nursorgate/client/server/tun/proxy"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTCPTunnel(t *testing.T) {
	// Start the tunnel
	l, err := net.Listen("tcp", "127.0.0.1:34567")
	assert.NoError(t, err)
	defer l.Close()
	hysteriaDialer, err := proxy.NewHysteriaDialer("lisi", "IW6gUxtuG46FURELO08p9L9I3GtHtfh1")
	if err != nil {
		t.Fatal(err)
	}
	tunnel := &hysteria_forwarding.TCPTunnel{
		HyClient: hysteriaDialer.Client,
	}
	tunnel.Serve(l)

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

func TestServer(t *testing.T) {
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
