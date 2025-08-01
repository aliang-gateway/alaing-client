package helper

import (
	"bytes"
	"fmt"
	"strings"

	"nursor.org/nursorgate/client/user"
	"nursor.org/nursorgate/common/logger"
)

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

	// 注入自定义 header
	headers["nursor-token"] = "1a12dfa3456"
	if authHeader, ok := headers["authorization"]; ok {
		w.isTokenFound = true
		user.SetAccessToken(authHeader)
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

// parseRequestLine parses the HTTP/1.x request line (e.g., "GET /path HTTP/1.1").
// (Keep this function as previously defined)
func parseRequestLine(line string) (method, path, protocol string) {
	parts := strings.SplitN(line, " ", 3)
	if len(parts) >= 1 {
		method = parts[0]
	}
	if len(parts) >= 2 {
		path = parts[1]
	}
	if len(parts) >= 3 {
		protocol = parts[2]
	}
	return
}
