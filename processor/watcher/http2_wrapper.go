package tls

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"

	"golang.org/x/net/http2/hpack"
	"nursor.org/nursorgate/common/logger"
	user "nursor.org/nursorgate/processor/auth"
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
			w.hpackDecoderReq.SetMaxDynamicTableSize(value)
			// w.hpackEncoderToServer.SetMaxDynamicTableSize(value)
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
		cacheFrame := []byte{}
		switch ftype {
		case frameTypeHeaders:
			cacheFrame = append(cacheFrame, frame...)
			_, priorityPayload, _ := extraceHeaderBlockPriority(frame, flags)

			frameHeaderPayload, err := extractHeaderBlockFromHeaderFrame(frame, flags)
			if err != nil {
				return err
			}

			headerBlock := append([]byte{}, frameHeaderPayload...)

			if flags&flagEndHeaders == 0 {
				// Continue collecting CONTINUATION frames
				for {
					continueFrame, ok := w.tryExtractFrameFromBuf(&w.reqBuf, false)
					if !ok {
						break
					}
					ctype := continueFrame[3]
					cflags := continueFrame[4]
					cstreamID := binary.BigEndian.Uint32(continueFrame[5:9]) & 0x7FFFFFFF
					// HTTP的帧有时候是乱序的
					if ctype != frameTypeContinuation || cstreamID != streamID {
						// 这里break，可以跳出这次stream的处理，出去再进来，就是处理另外一个stream的逻辑了
						break
					}
					// 这里来move，避免上边的break丢去包的问题
					w.reqBuf.Next(len(continueFrame))
					cacheFrame = append(cacheFrame, continueFrame...)
					frameHeaderPayload, err := extractHeaderBlockFromHeaderFrame(continueFrame, cflags)
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
			newHeaderFrames, err := w.rebuildReqHeadersWithInjectedField(headerBlock, streamID, flags, priorityPayload, "inner-token", user.GetInnerToken())

			if err != nil {
				logger.Error(fmt.Sprintf("❌❌Error rebuilding HTTP/2 Request headers for Stream %d: %v", streamID, err))
				preBuff.Write(cacheFrame)
				return nil
			}
			// newHeader[4] = flags
			preBuff.Write(newHeaderFrames)
			// return nil

		case frameTypeSettings:
			// 解析并保存 SETTINGS 帧
			w.ParseSettingsFrame(payload)
			logger.Debug(fmt.Sprintf("HTTP/2 SETTINGS frame processed and saved %v", payload))
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

func decodeHPACKInteger(r *bytes.Reader, prefixBits uint8) (uint64, error) {
	firstByte, err := r.ReadByte()
	if err != nil {
		return 0, err
	}
	prefixMask := uint8((1 << prefixBits) - 1)
	value := uint64(firstByte & prefixMask)

	if value < uint64(prefixMask) {
		return value, nil
	}

	// 多字节表示
	multiplier := uint64(1)
	for {
		b, err := r.ReadByte()
		if err != nil {
			return 0, err
		}
		value += uint64(b&127) * multiplier
		if b&128 == 0 {
			break
		}
		multiplier *= 128
	}
	return value, nil
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
	// headers := make(map[string]string)
	var headerFields []hpack.HeaderField
	w.hpackDecoderReq.SetEmitFunc(func(f hpack.HeaderField) {
		headerFields = append(headerFields, f)
	})
	r := bytes.NewReader(headerBlock)
	peek, err := r.ReadByte()
	if err != nil {
		return nil, err
	}
	r.UnreadByte()
	if peek&0b11100000 == 0b00100000 { // 001xxxxx => 动态表大小更新
		// ✅ 解析 table size
		size, err := decodeHPACKInteger(r, 5)
		if err != nil {
			return nil, fmt.Errorf("failed to parse dynamic table size: %w", err)
		}
		fmt.Printf("parsed dynamic table size: %d\n", size)

		// ✅ 手动设置 decoder 的 max size
		w.hpackDecoderReq.SetMaxDynamicTableSize(uint32(size))
	}
	oldRemaining, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("failed to read remaining header block: %w", err)
	}

	if _, err := w.hpackDecoderReq.Write(oldRemaining); err != nil {
		fmt.Printf("%d", headerBlock)
		return nil, fmt.Errorf("decode error: %w", err)
		// tmpDecoder := hpack.NewDecoder(4096, nil)
		// tmpDecoder.SetEmitFunc(func(f hpack.HeaderField) {
		// 	headerFields = append(headerFields, f)
		// })
		// if _, err2 := tmpDecoder.Write(headerBlock); err2 != nil {
		// 	fmt.Printf("%v", headerBlock)
		// 	return nil, fmt.Errorf("decode error: %w", err2)
		// }
	}

	// 2. 注入新字段
	// headers[keyToInject] = valueToInject
	headerFields = append(headerFields, hpack.HeaderField{Name: keyToInject, Value: valueToInject})
	// w.streams[streamID].RespHeaders = headers
	// logger.Debug(fmt.Sprintf("HTTP/2 Request Headers for Stream %d: %+v", streamID, headers))

	// 3. HPACK 编码新的 header block
	w.toServerBuffer.Reset()
	for _, v := range headerFields {
		// if err := w.hpackEncoderToServer.WriteField(hpack.HeaderField{Name: v.Name, Value: v.Value, Sensitive: true}); err != nil {
		// 	return nil, fmt.Errorf("hpack encode error: %w", err)
		// }
		v.Sensitive = true
		w.hpackEncoderToServer.WriteField(v)
	}
	newHeaderBlock := w.toServerBuffer.Bytes()
	// newHeaderBlock := headerBlock
	// 4. 拆分帧
	maxFrameSize := 16384
	if val, ok := w.GetHttp2Setting(SETTINGS_MAX_FRAME_SIZE); ok {
		maxFrameSize = int(val)
	}

	if originFlags&flagPriority != 0 {
		newHeaderBlock = append(priorityPayload, newHeaderBlock...)
	}

	// 👇 判断原始 HEADERS 是否有 END_STREAM
	hasEndStream := originFlags&flagEndStream != 0

	var frames [][]byte
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
			if len(remaining) == 0 {
				flags |= flagEndHeaders
				if hasEndStream {
					flags |= flagEndStream // ✅ 添加 END_STREAM
				}
			}
			if originFlags&flagPriority != 0 {
				flags |= flagPriority
			}
		} else {
			frameType = frameTypeContinuation
			if len(remaining) == 0 {
				flags |= flagEndHeaders
				// if hasEndStream {
				// 	flags |= flagEndStream // ✅ CONTINUATION 上也可以打 END_STREAM
				// }
			}
		}

		// 构造 frame
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

	// 5. 合并
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

// extractHeaderBlockFromHeaderFrame 从 HTTP/2 HEADERS/CONTINUATION 帧中提取 Header Block Fragment
//
// HTTP/2 HEADERS 帧的 Payload 结构（按顺序）：
//  1. Pad Length (1 byte) - 可选，当 PADDED flag 设置时存在
//  2. Priority (5 bytes) - 可选，当 PRIORITY flag 设置时存在
//     - Stream Dependency (31 bits, 4 bytes)
//     - E flag (1 bit) + Weight (1 byte)
//  3. Header Block Fragment (可变长度) - 必需，HPACK 编码的头部数据
//  4. Padding (Pad Length 字节) - 可选，当 PADDED flag 设置时存在
//
// 返回值：Header Block Fragment（去除 Padding 和 Priority 后的纯净 HPACK 数据）
func extractHeaderBlockFromHeaderFrame(frame []byte, flags byte) ([]byte, error) {
	// 提取 payload（跳过 9 字节的 frame header）
	payload := frame[frameHeaderLen:]
	offset := 0
	var padLength int

	// Step 1: 如果设置了 PADDED flag，读取 Pad Length（1 字节）
	if flags&flagPadded != 0 {
		if len(payload) < 1 {
			return nil, fmt.Errorf("PADDED flag set but payload too short to read pad length")
		}
		padLength = int(payload[offset])
		offset++

		// 验证：payload 必须至少包含 padLength + header block + padding
		if len(payload) < offset+padLength {
			return nil, fmt.Errorf("invalid padding: declared pad length %d exceeds remaining payload %d",
				padLength, len(payload)-offset)
		}
	}

	// Step 2: 如果设置了 PRIORITY flag，跳过 Priority 字段（5 字节）
	if flags&flagPriority != 0 {
		if len(payload) < offset+5 {
			return nil, fmt.Errorf("PRIORITY flag set but payload too short (need 5 bytes, have %d)",
				len(payload)-offset)
		}
		offset += 5
	}

	// Step 3: 提取 Header Block Fragment
	// Header Block Fragment 位于 offset 之后，padding 之前
	if flags&flagPadded != 0 {
		// 有 padding：从 offset 开始，到 (len(payload) - padLength) 结束
		if len(payload) < offset+padLength {
			return nil, fmt.Errorf("payload too short: need %d bytes for header block, have %d",
				offset+padLength, len(payload))
		}
		return payload[offset : len(payload)-padLength], nil
	}

	// 无 padding：从 offset 开始到 payload 末尾
	return payload[offset:], nil
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
