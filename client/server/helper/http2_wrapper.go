package helper

import (
	"bytes"
	"encoding/binary"
	"fmt"

	"golang.org/x/net/http2/hpack"
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
	frameTypeContinuation = 0x9

	flagEndStream  = 0x1
	flagAck        = 0x1
	flagEndHeaders = 0x4
	flagPadded     = 0x8
	flagPriority   = 0x20

	// HTTP/2 SETTINGS 参数类型
	SETTINGS_HEADER_TABLE_SIZE      = 0x1
	SETTINGS_ENABLE_PUSH            = 0x2
	SETTINGS_MAX_CONCURRENT_STREAMS = 0x3
	SETTINGS_INITIAL_WINDOW_SIZE    = 0x4
	SETTINGS_MAX_FRAME_SIZE         = 0x5
	SETTINGS_MAX_HEADER_LIST_SIZE   = 0x6
)

// ParseSettingsFrame 解析 HTTP/2 SETTINGS 帧
func (w *WatcherWrapConn) ParseSettingsFrame(payload []byte) {
	if len(payload)%6 != 0 {
		logger.Error("Invalid SETTINGS frame payload length")
		return
	}

	w.settingsMu.Lock()
	defer w.settingsMu.Unlock()

	for i := 0; i < len(payload); i += 6 {
		if i+6 > len(payload) {
			break
		}
		identifier := binary.BigEndian.Uint16(payload[i : i+2])
		value := binary.BigEndian.Uint32(payload[i+2 : i+6])

		w.http2Settings[identifier] = value

		// 记录解析到的 SETTINGS
		var settingName string
		switch identifier {
		case SETTINGS_HEADER_TABLE_SIZE:
			settingName = "HEADER_TABLE_SIZE"
		case SETTINGS_ENABLE_PUSH:
			settingName = "ENABLE_PUSH"
		case SETTINGS_MAX_CONCURRENT_STREAMS:
			settingName = "MAX_CONCURRENT_STREAMS"
		case SETTINGS_INITIAL_WINDOW_SIZE:
			settingName = "INITIAL_WINDOW_SIZE"
		case SETTINGS_MAX_FRAME_SIZE:
			settingName = "MAX_FRAME_SIZE"
		case SETTINGS_MAX_HEADER_LIST_SIZE:
			settingName = "MAX_HEADER_LIST_SIZE"
		default:
			settingName = fmt.Sprintf("UNKNOWN_%d", identifier)
		}

		logger.Debug(fmt.Sprintf("HTTP/2 SETTINGS: %s = %d", settingName, value))
	}
}

// GetHttp2Setting 获取指定的 HTTP/2 SETTINGS 值
func (w *WatcherWrapConn) GetHttp2Setting(identifier uint16) (uint32, bool) {
	w.settingsMu.Lock()
	defer w.settingsMu.Unlock()
	value, exists := w.http2Settings[identifier]
	return value, exists
}

// GetAllHttp2Settings 获取所有 HTTP/2 SETTINGS
func (w *WatcherWrapConn) GetAllHttp2Settings() map[uint16]uint32 {
	w.settingsMu.Lock()
	defer w.settingsMu.Unlock()

	// 返回一个副本，避免并发访问问题
	settings := make(map[uint16]uint32)
	for k, v := range w.http2Settings {
		settings[k] = v
	}
	return settings
}

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

func (w *WatcherWrapConn) processHttp2RequestFrame(preBuff *bytes.Buffer) error {
	for {
		frame, ok := w.tryExtractFrameFromBuf(&w.reqBuf, true)
		if !ok {
			return nil
		}
		ftype := frame[3]
		flags := frame[4]
		streamID := binary.BigEndian.Uint32(frame[5:9]) & 0x7FFFFFFF
		payload := frame[frameHeaderLen:]

		w.getOrCreateStream(streamID)

		switch ftype {
		case frameTypeHeaders:
			_, priorityPayload, _ := extraceHeaderBlockPriority(frame, flags)

			frameHeaderPayload, err := extractHeaderBlockFromHeaderFrame(frame, flags)
			if err != nil {
				return err
			}

			headerBlock := append([]byte{}, frameHeaderPayload...)

			if flags&flagEndHeaders == 0 {
				// Continue collecting CONTINUATION frames
				for {
					contFrame, ok := w.tryExtractFrameFromBuf(&w.reqBuf, false)
					if !ok {
						break
					}
					ctype := contFrame[3]
					cflags := contFrame[4]
					cstreamID := binary.BigEndian.Uint32(contFrame[5:9]) & 0x7FFFFFFF
					if ctype != frameTypeContinuation || cstreamID != streamID {
						// 这里break，可以跳出这次stream的处理，出去再进来，就是处理另外一个stream的逻辑了
						break
					}
					// 这里来move，避免上边的break丢去包的问题
					w.reqBuf.Next(len(contFrame))
					frameHeaderPayload, err := extractHeaderBlockFromHeaderFrame(contFrame, cflags)
					if err != nil {
						// 这里可能有bug
						break
					}
					headerBlock = append(headerBlock, frameHeaderPayload...)
					if cflags&flagEndHeaders != 0 {
						break
					}
				}
			}
			newHeader, err := w.rebuildReqHeadersWithInjectedField(headerBlock, streamID, flags, priorityPayload, "nursor-token", user.GetInnerToken())

			if err != nil {
				logger.Error(fmt.Sprintf("Error rebuilding HTTP/2 Request headers for Stream %d: %v", streamID, err))
				return err
			}
			newHeader[4] = flags
			preBuff.Write(newHeader)
			return nil

		case frameTypeSettings:
			// 解析并保存 SETTINGS 帧
			w.ParseSettingsFrame(payload)
			logger.Debug("HTTP/2 SETTINGS frame processed and saved")
			preBuff.Write(frame)

		case frameTypeData:
			// 处理请求体（DATA帧）
			w.streamsMu.Lock()
			stream := w.streams[streamID]
			stream.ReqBody.Write(payload)
			if flags&flagEndStream != 0 {
				stream.ReqEndStream = true
			}
			w.streamsMu.Unlock()
			preBuff.Write(frame)

		case frameTypeGoaway:
			logger.Debug("收到完整相应")
			preBuff.Write(frame)
		default:
			preBuff.Write(frame)

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
				contFrame, ok := w.tryExtractFrameFromBuf(&w.respBuf, false) // 仍然从请求缓冲区读取
				if !ok {
					break
				}
				ctype := contFrame[3]
				cflags := contFrame[4]
				cstreamID := binary.BigEndian.Uint32(contFrame[5:9]) & 0x7FFFFFFF
				if ctype != frameTypeContinuation || cstreamID != streamID {
					break // 不是当前流的 CONTINUATION 帧，或帧类型不对
				}
				w.respBuf.Next(len(contFrame))
				headerBlock = append(headerBlock, contFrame[frameHeaderLen:]...)
				if cflags&flagEndHeaders != 0 {
					break // 收到 END_HEADERS 标志
				}
			}
		}

		headers, err := w.decodeHeaderBlock(headerBlock, false)
		if err != nil {
			logger.Error(fmt.Sprintf("Error decoding HTTP/2 response headers for Stream %d: %v", streamID, err))
		} else {
			w.streams[streamID].RespHeaders = headers
			logger.Debug(fmt.Sprintf("HTTP/2 Response Headers for Stream %d: %+v", streamID, headers))
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
			if IsWatcherAllowed {
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
			}
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

// 重新组装http2的header，并注入新的字段
func (w *WatcherWrapConn) rebuildReqHeadersWithInjectedField(
	headerBlock []byte,
	streamID uint32,
	originFlags byte,
	priorityPayload []byte,
	keyToInject string,
	valueToInject string,
) ([]byte, error) {
	// 1. 解码原始 header block
	headers := make(map[string]string)
	w.hpackDecoderReq.SetEmitFunc(func(f hpack.HeaderField) {
		headers[f.Name] = f.Value
	})
	if _, err := w.hpackDecoderReq.Write(headerBlock); err != nil {
		return nil, fmt.Errorf("decode error: %w", err)
	}

	// 2. 注入新字段
	headers[keyToInject] = valueToInject

	w.streams[streamID].RespHeaders = headers
	logger.Debug(fmt.Sprintf("HTTP/2 Request Headers for Stream %d: %+v", streamID, headers))

	// 3. HPACK 编码新的 header block
	w.toServerBuffer.Reset()
	for k, v := range headers {
		if err := w.hpackEncoderToServer.WriteField(hpack.HeaderField{Name: k, Value: v}); err != nil {
			return nil, fmt.Errorf("hpack encode error: %w", err)
		}
	}
	newHeaderBlock := w.toServerBuffer.Bytes()

	// 4. 拆分帧
	maxFrameSize := 16384                                          // 默认值
	if val, ok := w.GetHttp2Setting(SETTINGS_MAX_FRAME_SIZE); ok { // 0x5 是 SETTINGS_MAX_FRAME_SIZE
		maxFrameSize = int(val)
	}

	var frames [][]byte
	if originFlags&flagPriority != 0 {
		newHeaderBlock = append(priorityPayload, newHeaderBlock...)
	}
	remaining := newHeaderBlock
	first := true
	for len(remaining) > 0 {
		chunkSize := len(remaining)
		if chunkSize > maxFrameSize {
			chunkSize = maxFrameSize
		}
		chunk := remaining[:chunkSize]
		remaining = remaining[chunkSize:]

		var frameType byte
		var flags byte
		if first {
			frameType = frameTypeHeaders
			flags = 0
			if len(remaining) == 0 {
				flags |= flagEndHeaders
			}
			if originFlags&flagPriority != 0 {
				flags |= flagPriority
			}

		} else {
			frameType = frameTypeContinuation
			flags = 0
			if len(remaining) == 0 {
				flags |= flagEndHeaders
			}
		}

		// 构造 HTTP/2 frame header
		length := uint32(len(chunk))
		frame := make([]byte, frameHeaderLen+len(chunk))
		frame[0] = byte(length >> 16)
		frame[1] = byte(length >> 8)
		frame[2] = byte(length)
		frame[3] = frameType
		frame[4] = flags
		binary.BigEndian.PutUint32(frame[5:9], streamID&0x7FFFFFFF)
		copy(frame[frameHeaderLen:], chunk)
		frames = append(frames, frame)
		first = false
	}

	// 5. 合并帧+剩余原始数据
	var fullBuf bytes.Buffer
	for _, f := range frames {
		fullBuf.Write(f)
	}
	return fullBuf.Bytes(), nil
}

func (w *WatcherWrapConn) decodeHeaderBlock(block []byte, isRequest bool) (map[string]string, error) {
	headers := make(map[string]string)

	var decoder *hpack.Decoder
	if isRequest {
		decoder = w.hpackDecoderReq
	} else {
		decoder = w.hpackDecoderResp
	}

	// 每次重新设置 emitFunc（因为 headers 是临时的）
	decoder.SetEmitFunc(func(f hpack.HeaderField) {
		headers[f.Name] = f.Value
	})

	_, err := decoder.Write(block)
	return headers, err
}

func extractHeaderBlockFromHeaderFrame(frame []byte, flags byte) ([]byte, error) {
	payload := frame[frameHeaderLen:]

	offset := 0

	// 如果有 PADDED，先读取 padding 长度
	if flags&flagPadded != 0 {
		if len(payload) < 1 {
			return nil, fmt.Errorf("PADDED flag set but payload too short")
		}
		padLength := int(payload[0])
		offset += 1

		// 确保长度合法
		if len(payload) < offset+padLength {
			return nil, fmt.Errorf("padding length invalid: payload=%d pad=%d", len(payload), padLength)
		}

		payload = payload[offset : len(payload)-padLength]
		offset = 0
	} else {
		payload = payload[offset:]
	}

	// 如果有 PRIORITY，跳过 5 字节
	if flags&flagPriority != 0 {
		if len(payload) < 5 {
			return nil, fmt.Errorf("PRIORITY flag set but payload too short")
		}
		payload = payload[5:]
	}

	return payload, nil
}

func extraceHeaderBlockPriority(frame []byte, flags byte) (bool, []byte, error) {
	payload := frame[frameHeaderLen:]
	if flags&flagPriority != 0 {
		if len(payload) < 5 {
			return false, nil, fmt.Errorf("PRIORITY flag set but payload too short")
		}
		priorityPayload := payload[:5]
		return true, priorityPayload, nil

	}
	return false, nil, nil
}
