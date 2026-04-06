package tls

import (
	"bytes"
	"fmt"
	"strings"

	"aliang.one/nursorgate/common/logger"
	"aliang.one/nursorgate/common/version"
	user "aliang.one/nursorgate/processor/auth"
)

func getHeaderCaseInsensitive(headers map[string]string, target string) (string, bool) {
	for key, value := range headers {
		if strings.EqualFold(key, target) {
			return value, true
		}
	}
	return "", false
}

func (w *WatcherWrapConn) parseHttp1Headers(data []byte) map[string]string {
	headers := make(map[string]string)
	lines := bytes.Split(data, []byte("\r\n"))
	for _, line := range lines {
		if len(line) == 0 {
			continue
		}
		parts := bytes.SplitN(line, []byte(": "), 2)
		if len(parts) == 2 {
			headers[string(parts[0])] = string(parts[1])
		}
	}
	return headers
}

func (w *WatcherWrapConn) processH1ReqHeaders() ([]byte, error) {
	dataOrigin := w.reqBuf.Bytes()
	headersEndIdx := bytes.Index(dataOrigin, []byte("\r\n\r\n"))
	if headersEndIdx == -1 {
		return dataOrigin, nil // 请求头还没接收完整
	}

	headersData := dataOrigin[:headersEndIdx+4]
	bodyData := dataOrigin[headersEndIdx+4:]

	w.http1ReqContent = string(dataOrigin)

	// 解析 headers
	headers := w.parseHttp1Headers(headersData)

	// 获取首行（request line）
	lines := bytes.SplitN(headersData, []byte("\r\n"), 2)
	if len(lines) < 2 {
		return dataOrigin, fmt.Errorf("invalid HTTP/1 request: missing request line")
	}
	requestLine := string(lines[0]) // 比如：GET /abc HTTP/1.1

	// 注入登录态 Authorization header，替代历史 inner-token 机制
	if authHeader := strings.TrimSpace(user.GetCurrentAuthorizationHeader()); authHeader != "" {
		headers["authorization-inner"] = authHeader
	}
	if _, ok := getHeaderCaseInsensitive(headers, "authorization-inner"); !ok {
		host, _ := getHeaderCaseInsensitive(headers, "Host")
		logger.Warn(fmt.Sprintf(
			"WatcherWrapConn: missing authorization-inner after HTTP/1 header rewrite request=%q host=%q",
			requestLine,
			host,
		))
	} else if !version.IsProdBuild() {
		host, _ := getHeaderCaseInsensitive(headers, "Host")
		logger.Info(fmt.Sprintf(
			"WatcherWrapConn: added authorization-inner for HTTP/1 request=%q host=%q",
			requestLine,
			host,
		))
	}

	// 重建 HTTP/1 请求头字符串
	var rebuilt bytes.Buffer
	rebuilt.WriteString(requestLine + "\r\n")
	for k, v := range headers {
		rebuilt.WriteString(fmt.Sprintf("%s: %s\r\n", k, v))
	}
	rebuilt.WriteString("\r\n") // headers 结束
	rebuilt.Write(bodyData)     // 添加 body（如果有）
	logger.Debug(fmt.Sprintf("new http1 content is : %s", rebuilt.String()))

	return rebuilt.Bytes(), nil
}

func deleteHeaderCaseInsensitive(headers map[string]string, target string) {
	for key := range headers {
		if strings.EqualFold(key, target) {
			delete(headers, key)
		}
	}
}
