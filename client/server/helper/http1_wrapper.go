package helper

import (
	"bytes"
	"fmt"
	"strconv"
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

func (w *WatcherWrapConn) processH1ReqHeaders() error {
	// 获取完整的请求头（HTTP/1.x）
	data := w.reqBuf.Bytes()
	headersEndIdx := bytes.Index(data, []byte("\r\n\r\n"))
	if headersEndIdx == -1 {
		return nil // 请求头还没有完全接收，等待更多数据
	}

	w.http1ReqContent = string(data)

	headersData := data[:headersEndIdx+4] // 包含头部和结束的 "\r\n\r\n"
	w.reqBuf.Next(headersEndIdx + 4)      // 从缓冲区移除已解析的头部数据

	// 解析 HTTP/1.x 头部
	headers := w.parseHttp1Headers(headersData)
	if authHeader, ok := headers["authorization"]; ok {
		w.isTokenFound = true
		user.SetAccessToken(authHeader)
		// logger.Debug(fmt.Sprintf("✅ HTTP/1.x Authorization token found: %s", authHeader))
	}

	if contentLengthStr, ok := headers["content-length"]; ok {
		contentLength, err := strconv.Atoi(contentLengthStr)
		if err != nil {
			logger.Error(fmt.Sprintf("Failed to parse Content-Length: %v", err))
			return fmt.Errorf("invalid Content-Length: %w", err)
		}
		// Ensure we have enough data in the buffer for the entire body
		if w.reqBuf.Len() < contentLength {
			// Body not fully received yet. We need to wait for more data.
			// This function will be called again by Read() when more data arrives.
			// For simplicity, we return here and rely on subsequent Read() calls.
			logger.Debug(fmt.Sprintf("HTTP/1.x body not fully received yet. Expected %d, got %d", contentLength, w.reqBuf.Len()))
			return nil
		}

		// Extract the body
		// You can add more specific body processing here if needed,
		// e.g., JSON parsing based on Content-Type header.
		// w.processResponseBody(body) // If you have a generic body processor
		// logger.Info(body)
	} else {
		// If Content-Length is not present, for requests like POST/PUT,
		// it might mean there's no body, or it's chunked encoding.
		// For simplicity here, we assume no body if no Content-Length.
		// Handling chunked encoding would require more complex parsing.
		logger.Debug("No Content-Length header found for HTTP/1.x request. Assuming no body or chunked encoding (not handled here).")
	}

	return nil
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
