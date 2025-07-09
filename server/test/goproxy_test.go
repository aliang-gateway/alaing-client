package test

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"io"
	"log"
	"math/big"
	"net"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"golang.org/x/net/http2"
)

func TestGoproxy(t *testing.T) {
	StartMitmHttp2()
}

// 全局 CA 证书
var caCert tls.Certificate

func StartMitmHttp2() {
	// 加载或生成 CA 证书
	certPath := "proxy_cert.pem"
	keyPath := "proxy_key.pem"
	if _, err := os.Stat(certPath); os.IsNotExist(err) {
		generateCACert(certPath, keyPath)
	}
	certPEM, err := os.ReadFile(certPath)
	if err != nil {
		log.Fatalf("Failed to read cert file: %v", err)
	}
	keyPEM, err := os.ReadFile(keyPath)
	if err != nil {
		log.Fatalf("Failed to read key file: %v", err)
	}
	caCert, err = tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		log.Fatalf("Failed to parse CA certificate: %v", err)
	}

	// 启动代理服务器
	listener, err := net.Listen("tcp", ":56432")
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}
	defer listener.Close()

	log.Println("Starting MITM proxy on :56432")
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Accept error: %v", err)
			continue
		}
		go handleConnection(conn)
	}

}

type connListener struct {
	conn net.Conn
}

func (l *connListener) Accept() (net.Conn, error) {
	return l.conn, nil
}

func (l *connListener) Close() error {
	return l.conn.Close()
}

func (l *connListener) Addr() net.Addr {
	return l.conn.LocalAddr()
}

// 处理客户端连接
func handleConnection(conn net.Conn) {
	defer conn.Close()

	// 创建 TLS 连接，支持 HTTP/1.x 和 HTTP/2
	tlsConn := tls.Server(conn, &tls.Config{
		GetCertificate: func(info *tls.ClientHelloInfo) (*tls.Certificate, error) {
			return &caCert, nil // 使用 CA 证书与客户端握手
		},
		NextProtos: []string{"h2", "http/1.1"}, // 支持 HTTP/2 和 HTTP/1.1
	})
	if err := tlsConn.Handshake(); err != nil {
		log.Printf("TLS handshake failed: %v", err)
		return
	}

	// 检查协议并处理
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "CONNECT" {
			handleConnect(w, r)
		} else {
			proxyRequest(w, r, nil) // 处理普通 HTTP 请求
		}
	})

	// 创建 HTTP/2 和 HTTP/1.x 服务
	srv := &http.Server{
		Handler: handler,
	}
	http2.ConfigureServer(srv, &http2.Server{})
	srv.Serve(&connListener{conn: tlsConn})
}

// 处理 CONNECT 请求
func handleConnect(w http.ResponseWriter, r *http.Request) {
	log.Printf("CONNECT to %s, Proto: %s", r.Host, r.Proto)

	// 连接上游服务器
	upstreamConn, err := net.DialTimeout("tcp", r.Host, 10*time.Second)
	if err != nil {
		log.Printf("Failed to connect to upstream: %v", err)
		http.Error(w, "Upstream connection failed", http.StatusBadGateway)
		return
	}
	defer upstreamConn.Close()

	// 劫持客户端连接
	hj, ok := w.(http.Hijacker)
	if !ok {
		log.Printf("Hijacking not supported")
		return
	}
	clientConn, _, err := hj.Hijack()
	if err != nil {
		log.Printf("Hijack failed: %v", err)
		return
	}
	defer clientConn.Close()

	// 返回 200 OK
	clientConn.Write([]byte("HTTP/1.1 200 OK\r\n\r\n"))

	// 生成伪造证书
	cert, err := generateCert(r.Host)
	if err != nil {
		log.Printf("Failed to generate cert: %v", err)
		return
	}

	// 创建客户端 TLS 连接
	clientTLS := tls.Server(clientConn, &tls.Config{
		Certificates: []tls.Certificate{cert},
		NextProtos:   []string{"h2", "http/1.1"},
	})
	if err := clientTLS.Handshake(); err != nil {
		log.Printf("Client TLS handshake failed: %v", err)
		return
	}

	// 创建上游 TLS 连接
	upstreamTLS := tls.Client(upstreamConn, &tls.Config{
		InsecureSkipVerify: true, // 跳过上游证书验证
		NextProtos:         []string{"h2", "http/1.1"},
	})
	if err := upstreamTLS.Handshake(); err != nil {
		log.Printf("Upstream TLS handshake failed: %v", err)
		return
	}

	// 双向代理
	go proxyTraffic(clientTLS, upstreamTLS)
	proxyTraffic(upstreamTLS, clientTLS)
}

// 代理请求（普通 HTTP）
func proxyRequest(w http.ResponseWriter, r *http.Request, upstreamConn net.Conn) {
	log.Printf("Proxying %s %s, Proto: %s", r.Method, r.URL.String(), r.Proto)

	// 修改请求（例如 Authorization）
	if strings.Contains(r.URL.String(), "cursor.") {
		if auth := r.Header.Get("Authorization"); auth != "" {
			newAuth := "Bearer modified_" + auth
			r.Header.Set("Authorization", newAuth)
			log.Printf("Modified Authorization header to: %s", newAuth)
		}
	}

	// 创建上游客户端
	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}
	resp, err := client.Do(r)
	if err != nil {
		log.Printf("Failed to forward request: %v", err)
		http.Error(w, "Request failed", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	// 复制响应头
	for k, v := range resp.Header {
		w.Header()[k] = v
	}
	w.WriteHeader(resp.StatusCode)

	// 读取并记录响应体
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Failed to read response body: %v", err)
	}
	if strings.Contains(r.URL.String(), "api2.cursor.sh") {
		log.Printf("Response Body: %s", string(body))
	}

	// 写回客户端
	w.Write(body)
}

// 双向代理流量（支持 HTTP/2 和 HTTP/1.x）
func proxyTraffic(clientConn, upstreamConn net.Conn) {
	defer clientConn.Close()
	defer upstreamConn.Close()

	// 创建双向通道
	go io.Copy(upstreamConn, clientConn)
	io.Copy(clientConn, upstreamConn)
}

// 生成 CA 证书
func generateCACert(certPath, keyPath string) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		log.Fatalf("Failed to generate private key: %v", err)
	}
	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization: []string{"My MITM Proxy"},
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(10 * 365 * 24 * time.Hour),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature,
		BasicConstraintsValid: true,
		IsCA:                  true,
	}
	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	if err != nil {
		log.Fatalf("Failed to create CA certificate: %v", err)
	}

	certOut, err := os.Create(certPath)
	if err != nil {
		log.Fatalf("Failed to write cert: %v", err)
	}
	pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes})
	certOut.Close()

	keyOut, err := os.Create(keyPath)
	if err != nil {
		log.Fatalf("Failed to write key: %v", err)
	}
	pem.Encode(keyOut, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(priv)})
	keyOut.Close()

	log.Printf("Generated CA certificate at %s and key at %s", certPath, keyPath)
}

// 生成伪造证书
func generateCert(host string) (tls.Certificate, error) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return tls.Certificate{}, err
	}
	template := x509.Certificate{
		SerialNumber: big.NewInt(2),
		Subject: pkix.Name{
			CommonName: host,
		},
		NotBefore:   time.Now(),
		NotAfter:    time.Now().Add(24 * time.Hour),
		KeyUsage:    x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		DNSNames:    []string{host},
	}
	ca, _ := x509.ParseCertificate(caCert.Certificate[0])
	derBytes, err := x509.CreateCertificate(rand.Reader, &template, ca, &priv.PublicKey, caCert.PrivateKey)
	if err != nil {
		return tls.Certificate{}, err
	}
	return tls.Certificate{
		Certificate: [][]byte{derBytes},
		PrivateKey:  priv,
	}, nil
}
