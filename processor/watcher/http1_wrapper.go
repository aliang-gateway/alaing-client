package tls

import (
	"bufio"
	"bytes"
	"fmt"
	"net/http"
	"strings"

	"aliang.one/nursorgate/common/logger"
	"aliang.one/nursorgate/common/version"
	user "aliang.one/nursorgate/processor/auth"
)

func serializeHTTPRequestHead(req *http.Request) ([]byte, error) {
	var rebuilt bytes.Buffer

	requestURI := req.RequestURI
	if requestURI == "" && req.URL != nil {
		requestURI = req.URL.RequestURI()
	}
	if requestURI == "" {
		return nil, fmt.Errorf("invalid HTTP/1 request: empty request URI")
	}

	if _, err := fmt.Fprintf(&rebuilt, "%s %s %s\r\n", req.Method, requestURI, req.Proto); err != nil {
		return nil, err
	}

	headers := req.Header.Clone()
	headers.Del("Host")
	if req.Host != "" {
		if _, err := fmt.Fprintf(&rebuilt, "Host: %s\r\n", req.Host); err != nil {
			return nil, err
		}
	}
	if err := headers.Write(&rebuilt); err != nil {
		return nil, err
	}
	if _, err := rebuilt.WriteString("\r\n"); err != nil {
		return nil, err
	}

	return rebuilt.Bytes(), nil
}

func (w *WatcherWrapConn) processH1ReqHeaders() ([]byte, bool, error) {
	dataOrigin := append([]byte(nil), w.reqBuf.Bytes()...)
	headersEndIdx := bytes.Index(dataOrigin, []byte("\r\n\r\n"))
	if headersEndIdx == -1 {
		return nil, false, nil
	}

	headersData := dataOrigin[:headersEndIdx+4]
	bodyData := dataOrigin[headersEndIdx+4:]
	w.http1ReqContent = string(dataOrigin)

	req, err := http.ReadRequest(bufio.NewReader(bytes.NewReader(headersData)))
	if err != nil {
		return nil, false, fmt.Errorf("invalid HTTP/1 request: %w", err)
	}
	// 将localhost:56432上监听到的，别家的host，改成openai.com等，改完后继续往下走正常的流程，最终发给aliang，不然后端要考虑处理各种第三方的host
	rewriteAliangHTTPRequestHost(req)

	if authHeader := strings.TrimSpace(user.GetCurrentAuthorizationHeader()); authHeader != "" {
		req.Header.Set("Authorization-Inner", authHeader)
	}

	requestLine := fmt.Sprintf("%s %s %s", req.Method, req.RequestURI, req.Proto)
	if req.Header.Get("Authorization-Inner") == "" {
		logger.Warn(fmt.Sprintf(
			"WatcherWrapConn: missing authorization-inner after HTTP/1 header rewrite request=%q host=%q",
			requestLine,
			req.Host,
		))
	} else if !version.IsProdBuild() {
		logger.Debug(fmt.Sprintf(
			"WatcherWrapConn: added authorization-inner for HTTP/1 request=%q host=%q",
			requestLine,
			req.Host,
		))
	}

	headBytes, err := serializeHTTPRequestHead(req)
	if err != nil {
		return nil, false, err
	}

	var rebuilt bytes.Buffer
	rebuilt.Write(headBytes)
	rebuilt.Write(bodyData)
	w.http1HeaderDone = true
	w.reqBuf.Reset()

	logger.Debug(fmt.Sprintf("new http1 content is : %s", rebuilt.String()))
	return rebuilt.Bytes(), true, nil
}
