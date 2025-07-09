package test

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/binary"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strconv"
	"testing"
)

func TestConnectMitm(t *testing.T) {
	// TLS 配置
	clientCert, err := tls.LoadX509KeyPair("../outbound/client.pem", "../outbound/client.key.pem")
	if err != nil {
		t.Fatal(err)
	}
	caCert, err := os.ReadFile("../inbound/cert/ca.pem")
	if err != nil {
		t.Fatal(err)
	}
	caCertPool := x509.NewCertPool()
	if !caCertPool.AppendCertsFromPEM(caCert) {
		t.Fatal("failed to append CA certificate to pool")
	}

	// 创建 TLS 配置的 HTTP 客户端
	tlsConfig := &tls.Config{
		RootCAs:      caCertPool,
		Certificates: []tls.Certificate{clientCert},
		ServerName:   "mitmproxy.nursor.com",
	}
	transport := &http.Transport{
		TLSClientConfig: tlsConfig,
	}
	client := &http.Client{Transport: transport}

	// 构造身份验证请求
	magic := []byte("MAGIC:custom")
	tokenData := map[string]string{"token": "123321"}
	jsonData, err := json.Marshal(tokenData)
	if err != nil {
		t.Fatal(err)
	}
	length := uint32(len(jsonData))
	lengthBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(lengthBytes, length)
	payload := append(magic, lengthBytes...)
	payload = append(payload, jsonData...)

	authReq, err := http.NewRequest("POST", "http://mitmproxy.nursor.com:8082/auth", bytes.NewReader(payload))
	if err != nil {
		t.Fatal(err)
	}
	authReq.Header.Set("Content-Length", strconv.Itoa(len(payload)))

	// 发送身份验证请求
	authResp, err := client.Do(authReq)
	if err != nil {
		t.Fatal(err)
	}
	defer authResp.Body.Close()
	authBody, err := io.ReadAll(authResp.Body)
	if err != nil {
		t.Fatal(err)
	}
	t.Log("Auth response:", string(authBody))
	if string(authBody) != "success" {
		t.Fatalf("Authentication failed: %s", string(authBody))
	}

	// 发送实际 HTTP 请求
	httpReq, err := http.NewRequest("GET", "http://www.baidu.com", nil)
	if err != nil {
		t.Fatal(err)
	}
	httpReq.Header.Set("Host", "www.baidu.com")

	// 通过代理发送请求
	httpResp, err := client.Do(httpReq)
	if err != nil {
		t.Fatal(err)
	}
	defer httpResp.Body.Close()
	httpBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		t.Fatal(err)
	}
	t.Log("HTTP response:", string(httpBody))
}
