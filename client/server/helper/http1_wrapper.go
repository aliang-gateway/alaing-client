package helper

import (
	"bufio"
	"bytes"
	"fmt"
	"net/textproto"
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

func (w *WatcherWrapConn) parseHttp1() {

}

func (w *WatcherWrapConn) processH1ReqHeaders() error {
	// 获取完整的请求头（HTTP/1.x）
	data := w.reqBuf.Bytes()
	headersEndIdx := bytes.Index(data, []byte("\r\n\r\n"))
	if headersEndIdx == -1 {
		return nil // 请求头还没有完全接收，等待更多数据
	}

	headersData := data[:headersEndIdx+4] // 包含头部和结束的 "\r\n\r\n"
	w.reqBuf.Next(headersEndIdx + 4)      // 从缓冲区移除已解析的头部数据

	// 解析 HTTP/1.x 头部
	headers := w.parseHttp1Headers(headersData)
	if authHeader, ok := headers["authorization"]; ok {
		w.isTokenFound = true
		user.SetAccessToken(authHeader)
		logger.Debug(fmt.Sprintf("✅ HTTP/1.x Authorization token found: %s", authHeader))
	}

	return nil
}

func (w *WatcherWrapConn) processHttp1Response() error {
	data := w.respBuf.Bytes()
	headersEndIdx := bytes.Index(data, []byte("\r\n\r\n"))

	if headersEndIdx == -1 {
		// 响应头部还没有完全接收，等待更多数据
		return nil
	}

	// 提取完整的响应头部（包括状态行和所有头部，以及最后的双 CRLF）
	rawHeaders := data[:headersEndIdx+4]
	// 从缓冲区中移除已解析的头部数据
	w.respBuf.Next(headersEndIdx + 4)

	// 使用 textproto.Reader 来解析 HTTP/1.x 头部，它能更好地处理各种头部格式

	tpReader := textproto.NewReader(bufio.NewReader(bytes.NewReader(rawHeaders)))

	// 解析状态行 (e.g., "HTTP/1.1 200 OK")
	statusLine, err := tpReader.ReadLine()
	if err != nil {
		logger.Error("Error reading HTTP/1.x status line: %v", err)
		return fmt.Errorf("failed to read status line: %w", err)
	}

	// 从状态行中提取状态码
	statusCode := 0
	statusParts := strings.SplitN(statusLine, " ", 3) // 至少包含 "HTTP/X.Y", "CODE", "STATUS_TEXT"
	if len(statusParts) >= 2 {
		statusCode, err = strconv.Atoi(statusParts[1])
		if err != nil {
			logger.Error("Error parsing HTTP/1.x status code '%s': %v", statusParts[1], err)
			// 即使解析失败，我们仍然可以继续处理头部
		}
	}

	logger.Debug(fmt.Sprintf("⚡️ HTTP/1.x Response Status: %d (%s)", statusCode, statusLine))

	// 解析其余的头部
	mimeHeader, err := tpReader.ReadMIMEHeader()
	if err != nil {
		logger.Error("Error reading HTTP/1.x response MIME headers: %v", err)
		return fmt.Errorf("failed to read MIME headers: %w", err)
	}

	// 遍历并记录一些重要的响应头部
	for key, values := range mimeHeader {
		logger.Debug(fmt.Sprintf("📄 HTTP/1.x Response Header: %s: %s", key, strings.Join(values, ", ")))
		// 你可以在这里根据需要处理特定的响应头，例如：
		// if strings.ToLower(key) == "content-type" {
		//     logger.Debug("Response Content-Type:", values[0])
		// }
		// if strings.ToLower(key) == "set-cookie" {
		//     logger.Debug("Response Set-Cookie:", values[0])
		// }
	}

	// 此时，响应头部已经完全处理并从缓冲区移除了。
	// 缓冲区中剩余的将是响应体数据。
	// 你可以在这里决定如何处理响应体，例如将其传递给一个通用的处理函数。
	// 例如：w.processResponseBody(w.respBuf.Bytes())
	// 注意：这里只是对头部进行了解析。响应体可能需要单独的、持续的读取和处理。
	// 在 HTTP/1.x 中，响应体通常会紧跟在头部之后，或者根据 Content-Length 或 Transfer-Encoding: chunked 来读取。
	// 这个方法只负责头部的解析，后续的 body 读取应该在 Write 方法的循环中继续进行。

	return nil
}
