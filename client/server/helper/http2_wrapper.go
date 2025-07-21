package helper

import (
	"bytes"
	"encoding/binary"
	"fmt"

	"nursor.org/nursorgate/client/user"
	"nursor.org/nursorgate/common/logger"
)

const (
	frameHeaderLen        = 9
	frameTypeData         = 0x0 // 补充 DATA 帧类型定义
	frameTypeHeaders      = 0x1
	frameTypePriority     = 0x2 // 常用帧类型，增加以便 switch default 不再打印
	frameTypeRstStream    = 0x3
	frameTypeSettings     = 0x4 // SETTINGS 帧
	frameTypePushPromise  = 0x5
	frameTypePing         = 0x6
	frameTypeGoaway       = 0x7
	frameTypeWindowUpdate = 0x8
	frameTypeCont         = 0x9

	flagEndHeaders = 0x4
	flagEndStream  = 0x1 // 补充 END_STREAM 标志
)

// http2Stream 结构体用于存储单个 HTTP/2 流（请求及其对应的响应）的所有信息
type http2Stream struct {
	// 请求信息
	ReqHeaders   map[string]string
	ReqBody      bytes.Buffer
	ReqEndStream bool // 标记请求体是否已结束 (收到 END_STREAM)

	// 响应信息
	RespHeaders   map[string]string
	RespBody      bytes.Buffer
	RespEndStream bool // 标记响应体是否已结束 (收到 END_STREAM)
}

func (w *WatcherWrapConn) processHttp2Frame(frame []byte) {
	// length := binary.BigEndian.Uint32(append([]byte{0}, frame[0:3]...))
	ftype := frame[3]
	flags := frame[4]
	streamID := binary.BigEndian.Uint32(frame[5:9]) & 0x7FFFFFFF
	payload := frame[frameHeaderLen:]

	switch ftype {
	case frameTypeHeaders:
		headerBlock := append([]byte{}, payload...)
		if flags&flagEndHeaders == 0 {
			// Continue collecting CONTINUATION frames
			for {
				contFrame, ok := w.tryParseFrameFromBuf(&w.reqBuf)
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
				logger.Debug("✅ Authorization:", val)
			}
		}

	case frameTypeData:
		// 处理请求体（DATA帧）
		logger.Debug(fmt.Sprintf("📦 HTTP/2 DATA frame stream=%d len=%d", streamID, len(payload)))
		if len(payload) > 0 {
			// 你可以加点智能逻辑，比如判断是不是 JSON、form-data 等
			logger.Debug("📨 HTTP/2 Request body:", string(payload))
		}
		if flags&flagEndStream != 0 {
			logger.Debug("✅ End of request body stream.")
		}
	}
}

func (w *WatcherWrapConn) processHttp2ResponseFrame(frame []byte) {
	ftype := frame[3]
	flags := frame[4]
	streamID := binary.BigEndian.Uint32(frame[5:9]) & 0x7FFFFFFF

	w.streamsMu.Lock()
	stream, ok := w.streams[streamID]
	if !ok {
		stream = &http2Stream{}
		w.streams[streamID] = stream
	}
	w.streamsMu.Unlock()

	payload := frame[frameHeaderLen:]
	switch ftype {
	case frameTypeHeaders:
		// 处理 HEADERS 帧 (请求头部)
		headerBlock := append([]byte{}, payload...)
		if flags&flagEndHeaders == 0 {
			// 继续收集 CONTINUATION 帧
			for {
				contFrame, ok := w.tryParseFrameFromBuf(&w.reqBuf) // 仍然从请求缓冲区读取
				if !ok {
					break
				}
				ctype := contFrame[3]
				cflags := contFrame[4]
				cstreamID := binary.BigEndian.Uint32(contFrame[5:9]) & 0x7FFFFFFF
				if ctype != frameTypeCont || cstreamID != streamID {
					break // 不是当前流的 CONTINUATION 帧，或帧类型不对
				}
				headerBlock = append(headerBlock, contFrame[frameHeaderLen:]...)
				if cflags&flagEndHeaders != 0 {
					break // 收到 END_HEADERS 标志
				}
			}
		}
		headers, err := decodeHeaderBlock(headerBlock)
		if err == nil {
			stream.ReqHeaders = headers
			if authHeader, ok := headers[":authorization"]; ok { // HTTP/2 伪头部
				w.isTokenFound = true // 令牌通常只在请求中找到一次
				user.SetAccessToken(authHeader)
				logger.Debug("✅ HTTP/2 Request Authorization token found.")
			}

		} else {
			logger.Error(fmt.Sprintf("Error decoding HTTP/2 request headers for Stream %d: %v", streamID, err))
		}
		if flags&flagEndStream != 0 {
			stream.ReqEndStream = true
		}

	case frameTypeData:
		// 处理 DATA 帧 (请求体)
		stream.ReqBody.Write(payload)

		if flags&flagEndStream != 0 {
			stream.ReqEndStream = true
			logger.Debug(fmt.Sprintf("✅ HTTP/2 Request body EndStream for Stream %d.", streamID))
		}

	case frameTypeRstStream, frameTypeGoaway:
		// 流重置或连接关闭，清除流信息
		w.streamsMu.Lock()
		delete(w.streams, streamID)
		w.streamsMu.Unlock()
		logger.Info(fmt.Sprintf("HTTP/2 Stream %d reset or GoAway, removing.", streamID))
	case frameTypeSettings, frameTypePing, frameTypeWindowUpdate, frameTypePriority, frameTypePushPromise:
		// 忽略其他帧类型或进行相应的协议处理
		logger.Debug(fmt.Sprintf("ℹ️ HTTP/2 Request Unhandled/Ignored frame type: %d (stream=%d)", ftype, streamID))
	default:
		logger.Debug(fmt.Sprintf("⚠️ HTTP/2 Request Unknown frame type: %d (stream=%d)", ftype, streamID))
	}

}

func (w *WatcherWrapConn) processH2ResponseBody(payload []byte) {
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
