package helper

import (
	"encoding/binary"
	"fmt"

	"nursor.org/nursorgate/client/user"
	"nursor.org/nursorgate/common/logger"
)

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
		w.processH2ResponseBody(payload)
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
