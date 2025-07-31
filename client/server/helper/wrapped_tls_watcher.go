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

	streams   map[uint32]*http2Stream // 存储活跃的 HTTP/2 流
	streamsMu sync.Mutex              // 保护 streams map 的并发访问
}

func NewWatcherWrapConn(conn1 *tls.Conn) *WatcherWrapConn {
	return &WatcherWrapConn{Conn: conn1, streams: map[uint32]*http2Stream{}, hpackDecoderReq: hpack.NewDecoder(4096, nil), hpackDecoderResp: hpack.NewDecoder(4096, nil)}
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
	n, err := w.Conn.Read(p)
	if n <= 0 {
		return n, err
	}
	if IsWatcherAllowed {
		w.reqBuf.Write(p[:n])
		if !w.prefetched {
			if w.reqBuf.Len() >= 24 {
				preface := w.reqBuf.Bytes()[:24]
				if string(preface) == http2.ClientPreface {
					w.prefetched = true
					w.reqBuf.Next(24)
					w.parseHttp2Req()
					return n, err
				} else {
					// 明确不是 http2 才判定为 http1
					w.isHttp1 = true
					w.parseHttp1Req()
				}
			} else {
				// 数据不够，等下一次
				return n, err
			}
		}
	}
	return n, err
}

func (w *WatcherWrapConn) parseHttp1Req() {
	logger.Debug("📦 HTTP/1 connection preface detected")
	w.processH1ReqHeaders()
}

func (w *WatcherWrapConn) parseHttp2Req() {
	logger.Debug("📦 HTTP/2 connection preface detected")
	for {
		// 避免粘包、半包的情况
		frame, ok := w.tryParseFrameFromBuf(&w.reqBuf, true)
		if !ok {
			break
		}
		w.processHttp2RequestFrame(frame)
	}
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
		if IsWatcherAllowed {
			logger.HttpInfo(fmt.Sprintf("-----------starth1----------------\n%s--->-h1-<---\n%s-----------------endh1------------------\n\n", w.http1ReqContent, w.http1RespContent))
		}
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
