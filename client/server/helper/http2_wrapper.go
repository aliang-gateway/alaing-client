package helper

import (
	"bytes"
	"encoding/binary"
	"fmt"

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

func (w *WatcherWrapConn) processHttp2RequestFrame(frame []byte) {
	// length := binary.BigEndian.Uint32(append([]byte{0}, frame[0:3]...))
	ftype := frame[3]
	flags := frame[4]
	streamID := binary.BigEndian.Uint32(frame[5:9]) & 0x7FFFFFFF
	payload := frame[frameHeaderLen:]

	if _, ok := w.streams[streamID]; !ok {
		w.streams[streamID] = &http2Stream{}
	}

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
		headers, err := w.decodeHeaderBlock(headerBlock, true)
		if err != nil {
			logger.Error(fmt.Sprintf("Error decoding HTTP/2 headers for Stream %d: %v", streamID, err))
		} else {
			w.streams[streamID].ReqHeaders = headers
			logger.Debug(fmt.Sprintf("HTTP/2 Headers for Stream %d: %+v", streamID, headers))
		}

	case frameTypeData:
		// 处理请求体（DATA帧）
		w.streamsMu.Lock()

		stream := w.streams[streamID]
		stream.ReqBody.Write(payload)
		if flags&flagEndStream != 0 {
			stream.RespEndStream = true
		}
		w.streamsMu.Unlock()

	case frameTypeGoaway:
		logger.Debug("收到完整相应")
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
				contFrame, ok := w.tryParseFrameFromBuf(&w.respBuf) // 仍然从请求缓冲区读取
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
		headers, err := w.decodeHeaderBlock(headerBlock, false)
		if err != nil {
			logger.Error(fmt.Sprintf("Error decoding HTTP/2 headers for Stream %d: %v", streamID, err))
		} else {
			w.streams[streamID].RespHeaders = headers
			logger.Debug(fmt.Sprintf("HTTP/2 Headers for Stream %d: %+v", streamID, headers))
		}

	case frameTypeData:
		// 处理 DATA 帧 (请求体)
		stream.RespBody.Write(payload)

		if flags&flagEndStream != 0 {
			w.streamsMu.Lock()
			stream.ReqEndStream = true
			trimBody := func(buf *bytes.Buffer, n int) string {
				data := buf.Bytes()
				if len(data) > n {
					return string(data[:n]) + "..."
				}
				return string(data)
			}
			logger.HttpInfo(fmt.Sprintf(
				"-----------starth2----------------\n"+
					"ReqHeaders: %+v\n"+
					"RespHeaders: %+v\n"+
					"ReqBody: %s\n"+
					"RespBody: %s\n"+
					"-------------------------endh2------------------\n\n",
				stream.ReqHeaders,
				stream.RespHeaders,
				trimBody(&stream.ReqBody, 512),
				trimBody(&stream.RespBody, 512),
			))
			delete(w.streams, streamID)
			w.streamsMu.Unlock()

		}

	case frameTypeRstStream, frameTypeGoaway:
		// 流重置或连接关闭，清除流信息

		logger.Info(fmt.Sprintf("HTTP/2 Stream %d reset or GoAway, removing.", streamID))
	case frameTypeSettings, frameTypePing, frameTypeWindowUpdate, frameTypePriority, frameTypePushPromise:
		// 忽略其他帧类型或进行相应的协议处理
		logger.Debug(fmt.Sprintf("ℹ️ HTTP/2 Request Unhandled/Ignored frame type: %d (stream=%d)", ftype, streamID))
	default:
		logger.Debug(fmt.Sprintf("⚠️ HTTP/2 Request Unknown frame type: %d (stream=%d)", ftype, streamID))
	}

}
