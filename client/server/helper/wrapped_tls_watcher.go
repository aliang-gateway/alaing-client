package helper

import (
	"bytes"
	"crypto/tls"
	"encoding/binary"
	"fmt"
	"net"
	"sync"

	"golang.org/x/net/http2"
	"golang.org/x/net/http2/hpack"
	"nursor.org/nursorgate/common/logger"
)

type WatcherWrapConn struct {
	net.Conn
	reqBuf           bytes.Buffer
	respBuf          bytes.Buffer
	prefetched       bool
	isTokenFound     bool
	isHttp1          bool
	http1ReqContent  string
	http1RespContent string
	hpackDecoderReq  *hpack.Decoder // 解码请求头
	hpackDecoderResp *hpack.Decoder // 解码响应头

	toServerBuffer       bytes.Buffer
	hpackEncoderToServer *hpack.Encoder
	// hpackDecoderToServer *hpack.Decoder

	streams   map[uint32]*http2Stream // 存储活跃的 HTTP/2 流
	streamsMu sync.Mutex              // 保护 streams map 的并发访问

	// HTTP/2 SETTINGS 存储
	http2Settings map[uint16]uint32 // 存储 HTTP/2 SETTINGS 参数
	settingsMu    sync.Mutex        // 保护 settings map 的并发访问

	pendingBuffer *bytes.Buffer
}

func NewWatcherWrapConn(conn1 *tls.Conn) *WatcherWrapConn {
	newBuffer := bytes.NewBuffer([]byte{})
	return &WatcherWrapConn{
		Conn:             conn1,
		streams:          map[uint32]*http2Stream{},
		http2Settings:    map[uint16]uint32{},
		hpackDecoderReq:  hpack.NewDecoder(4096, nil),
		hpackDecoderResp: hpack.NewDecoder(4096, nil),

		toServerBuffer:       *newBuffer,
		hpackEncoderToServer: hpack.NewEncoder(newBuffer),
		// hpackDecoderToServer: ,
	}
}

func (w *WatcherWrapConn) getOrCreateStream(id uint32) *http2Stream {
	w.streamsMu.Lock()
	defer w.streamsMu.Unlock()
	if s, ok := w.streams[id]; ok {
		return s
	}
	s := &http2Stream{}
	w.streams[id] = s
	return s
}

func (w *WatcherWrapConn) Read(p []byte) (int, error) {
	if w.pendingBuffer != nil && w.pendingBuffer.Len() > 0 {
		pending := w.pendingBuffer.Bytes()
		copied := copy(p, pending)
		w.pendingBuffer.Next(copied)
		return copied, nil
	}
	n, err := w.Conn.Read(p)
	if n <= 0 || err != nil {
		return n, err
	}

	w.reqBuf.Write(p[:n])

	if w.reqBuf.Len() >= 24 {
		preface := w.reqBuf.Bytes()[:24]
		if string(preface) == http2.ClientPreface || w.prefetched {
			w.prefetched = true
			w.reqBuf.Next(24)
			newBuf, err := w.parseHttp2Req()
			if err != nil {
				logger.Error(fmt.Sprintf("Error parsing HTTP/2 request: %v", err))
				return n, err
			}
			w.reqBuf.Write(newBuf)

			// 安全地复制数据，避免越界
			copied := copy(p, newBuf)
			if copied < len(newBuf) {
				w.pendingBuffer = bytes.NewBuffer(newBuf[copied:])
			}
			return copied, err
		} else {
			// 明确不是 http2 才判定为 http1
			w.isHttp1 = true
			newReq, err := w.parseHttp1Req()
			if err != nil {
				return n, err
			}

			// 安全地复制数据，避免越界
			copied := copy(p, newReq)
			if copied < len(newReq) {
				w.pendingBuffer = bytes.NewBuffer(newReq[copied:])
			}
			return copied, err
		}
	} else {
		// 数据不够，等下一次
		return n, err
	}
}

func (w *WatcherWrapConn) parseHttp1Req() ([]byte, error) {
	logger.Debug("📦 HTTP/1 connection preface detected")
	return w.processH1ReqHeaders()
}

func (w *WatcherWrapConn) parseHttp2Req() ([]byte, error) {
	logger.Debug("📦 HTTP/2 connection preface detected")
	preBuff := bytes.NewBuffer([]byte{})
	for {
		// 避免粘包、半包的情况
		frame, ok := w.tryParseFrameFromBuf(&w.reqBuf, true)
		if !ok {
			break
		}
		newFrame, isEnd, err := w.processHttp2RequestFrame(frame)
		preBuff.Write(newFrame)
		if err != nil {
			logger.Error(fmt.Sprintf("Error processing HTTP/2 request frame: %v", err))
			// 修复：[]byte 不能直接用 + 连接，应该用 append
			result := append(preBuff.Bytes(), w.reqBuf.Bytes()...)
			return result, nil
		}
		if isEnd {
			break
		}
	}
	// 修复：[]byte 不能直接用 + 连接，应该用 append
	result := append(preBuff.Bytes(), w.reqBuf.Bytes()...)
	return result, nil
}

func (w *WatcherWrapConn) Write(p []byte) (n int, err error) {
	// 先调用原本的Write逻辑
	n, err = w.Conn.Write(p)
	if err != nil {
		return n, err
	}

	// 假设是写入 HTTP/2 的 DATA 帧
	if w.isHttp1 {
		w.http1RespContent = string(p)
	} else {
		w.respBuf.Write(p[:n])
		for {
			frame, ok := w.tryParseFrameFromBuf(&w.respBuf, true)
			if ok {
				w.processHttp2ResponseFrame(frame)
			} else {
				break
			}
		}
	}

	return n, err
}

func (w *WatcherWrapConn) tryParseFrameFromBuf(buf *bytes.Buffer, shouldMove bool) ([]byte, bool) {
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
	if shouldMove {
		buf.Next(totalLen)
	}
	return frame, true
}
