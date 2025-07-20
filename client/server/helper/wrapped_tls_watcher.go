package helper

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"net"

	"golang.org/x/net/http2/hpack"
	"nursor.org/nursorgate/client/user"
	"nursor.org/nursorgate/common/logger"
)

const (
	frameHeaderLen   = 9
	frameTypeHeaders = 0x1
	frameTypeCont    = 0x9
	flagEndHeaders   = 0x4
)

type WatcherWrapConn struct {
	net.Conn
	reqBuf       bytes.Buffer
	respBuf      bytes.Buffer
	prefetched   bool
	isTokenFound bool
	isHttp1      bool
}

func (w *WatcherWrapConn) Read(p []byte) (int, error) {
	n, err := w.Conn.Read(p)
	if n > 0 && !w.isTokenFound {
		// step 1: prefetch the client preface (24 bytes)
		if !w.isTokenFound {
			//  !w.prefetched && !w.isHttp1
			w.reqBuf.Write(p[:n])
			preface := w.reqBuf.Next(24)
			if w.reqBuf.Len() < 24 || string(preface) != "PRI * HTTP/2.0\r\n\r\nSM\r\n\r\n" {
				logger.Debug("❌ Not a valid HTTP/2 connection preface")
				w.isHttp1 = true // 标记为已完成预取（虽然这不是 HTTP/2 请求）
				// 不是http2,就是http1
				if !w.isTokenFound && w.isHttp1 {
					if err := w.processHttp1Headers(); err != nil {
						return n, err
					}
				}
				// return n, err
			} else {
				w.prefetched = true
				logger.Debug("✅ HTTP/2 connection preface detected")
				if w.reqBuf.Len() == 0 {
					return n, err // preface done, no other data
				}
				for {
					frame, ok := w.tryParseFrameFromBuf(w.reqBuf)
					if !ok {
						break
					}
					w.processHttp2Frame(frame)
				}
			}

		} else {
			w.reqBuf.Write(p[:n])
		}

	}
	return n, err
}

func (w *WatcherWrapConn) Write(p []byte) (n int, err error) {
	// 先调用原本的Write逻辑
	n, err = w.Conn.Write(p)
	if err != nil {
		return n, err
	}

	// 假设是写入 HTTP/2 的 DATA 帧
	if !w.isHttp1 {
		w.respBuf.Write(p[:n])
		frame, ok := w.tryParseFrameFromBuf(w.respBuf)
		if ok {
			w.processHttp2ResponseFrame(frame)
		}
	}

	return n, err
}

func (w *WatcherWrapConn) processResponseBody(payload []byte) {
	// 这里只是简单地记录日志，可以根据需要对响应体进行更复杂的处理
	logger.Debug("Response body:", string(payload))

	// 如果是JSON或其他可解析的格式，你可以尝试解析
	// 假设是JSON格式
	if len(payload) > 0 && payload[0] == '{' {
		logger.Debug("JSON response detected")
		// 执行JSON解析操作
		// json.Unmarshal(payload, &yourStruct)
	}
}

func (w *WatcherWrapConn) processHttp2ResponseFrame(frame []byte) {
	length := binary.BigEndian.Uint32(append([]byte{0}, frame[0:3]...))
	ftype := frame[3]
	flags := frame[4]
	streamID := binary.BigEndian.Uint32(frame[5:9]) & 0x7FFFFFFF
	logger.Debug(fmt.Sprintf("Response Frame len=%d type=%d stream=%d", length, ftype, streamID))
	payload := frame[frameHeaderLen:]

	if ftype == frameTypeData {
		// DATA帧，处理响应体
		logger.Debug("Processing HTTP/2 DATA frame for stream ID:", streamID)

		// 判断是否是最后一个数据帧
		if flags&flagEndStream != 0 {
			logger.Debug("This is the last DATA frame for stream:", streamID)
		}

		// 这里你可以根据需要来处理响应体数据
		// 比如保存到文件，记录日志等
		w.processResponseBody(payload)
	}
}

func (w *WatcherWrapConn) processHttp1Headers() error {
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

func (w *WatcherWrapConn) tryParseFrameFromBuf(buf bytes.Buffer) ([]byte, bool) {
	data := buf.Bytes()
	if len(data) < frameHeaderLen {
		return nil, false
	}
	length := binary.BigEndian.Uint32(append([]byte{0}, data[0:3]...))
	totalLen := frameHeaderLen + int(length)
	if len(data) < totalLen {
		return nil, false
	}
	frame := make([]byte, totalLen)
	copy(frame, data[:totalLen])
	buf.Next(totalLen)
	return frame, true
}

func (w *WatcherWrapConn) processHttp2Frame(frame []byte) {
	length := binary.BigEndian.Uint32(append([]byte{0}, frame[0:3]...))
	ftype := frame[3]
	flags := frame[4]
	streamID := binary.BigEndian.Uint32(frame[5:9]) & 0x7FFFFFFF
	logger.Debug(fmt.Sprintf("Frame len=%d type=%d stream=%d", length, ftype, streamID))
	payload := frame[frameHeaderLen:]

	if ftype == frameTypeHeaders {
		headerBlock := append([]byte{}, payload...)
		if flags&flagEndHeaders == 0 {
			// Continue collecting CONTINUATION frames
			for {
				contFrame, ok := w.tryParseFrame()
				if !ok {
					break
				}
				ctype := contFrame[3]
				cflags := contFrame[4]
				cstreamID := binary.BigEndian.Uint32(contFrame[5:9]) & 0x7FFFFFFF
				if ctype != frameTypeCont || cstreamID != streamID {
					break
				}
				headerBlock = append(headerBlock, contFrame[frameHeaderLen:]...)
				if cflags&flagEndHeaders != 0 {
					break
				}
			}
		}
		headers, err := decodeHeaderBlock(headerBlock)
		if err == nil {
			if val, ok := headers["authorization"]; ok {
				w.isTokenFound = true
				user.SetAccessToken(val)
				logger.Debug("\u2705 Authorization:", val)
			}
		}
	}
}

func decodeHeaderBlock(block []byte) (map[string]string, error) {
	headers := make(map[string]string)
	decoder := hpack.NewDecoder(4096, func(hf hpack.HeaderField) {
		headers[hf.Name] = hf.Value
	})
	_, err := decoder.Write(block)
	return headers, err
}
