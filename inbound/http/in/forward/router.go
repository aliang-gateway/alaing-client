package forward

import (
	"bufio"
	"bytes"
	"context"
	"crypto/tls"
	"io"
	"net/http"
	"sync"
	"time"

	"nursor.org/nursorgate/inbound/http/out"
	"nursor.org/nursorgate/runner/utils"

	"nursor.org/nursorgate/common/logger"
	"nursor.org/nursorgate/common/model"
)

var sslClient = &http.Client{
	Timeout: 30 * time.Second,
	Transport: &http.Transport{
		TLSClientConfig:       &tls.Config{InsecureSkipVerify: true}, // 仅用于调试
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: 10 * time.Second,
	},
}
var plainClient = &http.Client{
	Timeout: 30 * time.Second,
}

func HandleHttp2(clientConn *tls.Conn, req *http.Request) error {
	allowDomain := model.NewAllowProxyDomain()
	remoteHost := req.Host
	if allowDomain.IsAllowToCursor(remoteHost) {
		var isHttp2 = true
		alpnVersion := clientConn.ConnectionState().NegotiatedProtocol
		if alpnVersion != "h2" {
			isHttp2 = false
		}
		outBoundClient, err := out.NewHttp2ProxyClient(utils.GetServerHost(), req.Host, isHttp2)
		if err != nil {
			logger.Error(err.Error())
			return err
		}
		err = outBoundClient.Forward(clientConn)
		if err != nil {
			return err
		}
		return nil
	} else {
		directOutboundClient, err := out.NewDirectHttp2Client(remoteHost)
		if err != nil {
			logger.Error(err.Error())
			return err
		}

		err = directOutboundClient.ForwardDirect(clientConn)
		if err != nil {
			return err
		}
		return nil
	}
}

func HandleHttp1(w http.ResponseWriter, r *http.Request, isSSL bool) error {
	allowDomain := model.NewAllowProxyDomain()
	if allowDomain.IsAllowToCursor(r.Host) {
		proxyCursorRequest(w, r)
	} else {
		proxyNoCursorRequest(w, r, isSSL)
	}

	return nil
}

func proxyNoCursorRequest(w http.ResponseWriter, r *http.Request, isSSL bool) {
	if err := r.Context().Err(); err != nil {
		http.Error(w, "Request canceled", http.StatusServiceUnavailable)
		return
	}
	// 读取请求体，确保可重复使用
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusInternalServerError)
		return
	}
	r.Body.Close()

	// 创建带超时的上下文
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 创建新请求，复制更多字段
	newReq, err := http.NewRequestWithContext(
		ctx,
		r.Method,
		r.URL.String(),
		bytes.NewReader(body),
	)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// 复制原始请求的关键字段
	newReq.Header = r.Header.Clone() // 使用 Clone 避免引用原始 Header
	newReq.Proto = r.Proto           // 复制协议版本（HTTP/1.1 或 HTTP/2）
	newReq.ProtoMajor = r.ProtoMajor
	newReq.ProtoMinor = r.ProtoMinor
	newReq.Host = r.Host
	// 创建上游客户端，用系统默认的证书
	var client *http.Client
	if isSSL {
		client = sslClient
	} else {
		client = plainClient
	}
	resp, err2 := client.Do(newReq)
	if err2 != nil {
		logger.Error(err2)
		http.Error(w, "Request failed", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()
	for key, values := range resp.Header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}

	// 写回客户端
	_, err = io.Copy(w, resp.Body)
	if err != nil {
		logger.Error(err.Error())
	}
}

func proxyCursorRequest(w http.ResponseWriter, r *http.Request) {
	outBoundClient, err := out.NewOutboundClient(utils.GetServerHost(), r.Host)
	if err != nil {
		handleAuthError(w, r, err)
		logger.Error(err.Error())
		return
	}
	defer func(outBoundClient *out.OutboundClient) {
		err := outBoundClient.Close()
		if err != nil {
			logger.Error("failure to close outbound")
		}
	}(outBoundClient) // 确保连接关闭

	var wg sync.WaitGroup
	wg.Add(1)

	err = r.Write(outBoundClient)
	if err != nil {
		logger.Error(err)
	}
	resp, err := http.ReadResponse(bufio.NewReader(outBoundClient), r)
	if err != nil {
		logger.Error("Failed to read response:", err)
		http.Error(w, "Failed to read response", http.StatusInternalServerError)
		return
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {

		}
	}(resp.Body)
	// 复制响应头部
	for k, v := range resp.Header {
		for _, val := range v {
			w.Header().Add(k, val)
		}
	}
	// 设置状态码并复制 Body
	// w.WriteHeader(resp.StatusCode)
	_, err = io.Copy(w, resp.Body)
	if err != nil {
		logger.Error(err.Error())
	}

}

func handleAuthError(w http.ResponseWriter, r *http.Request, err error) {
	w.Header().Set("Content-Type", "text/plain")
	w.Header().Set("WWW-Authenticate", `Basic realm="Nursor Gate"`)
	w.Header().Set("Connection", "close")
	w.Header().Set("Date", time.Now().Format(time.RFC1123))
	w.Header().Set("Server", "Nursor Gate")
	w.Header().Set("Host", r.Host)
	w.WriteHeader(http.StatusUnauthorized)
	w.Write([]byte(err.Error()))
}
