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

	toServerBuffer       *bytes.Buffer
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

		toServerBuffer:       newBuffer,
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
	// 1. 优先从 pendingBuffer 中读取
	if w.pendingBuffer != nil && w.pendingBuffer.Len() > 0 {
		copied := copy(p, w.pendingBuffer.Bytes())
		w.pendingBuffer.Next(copied)
		if w.pendingBuffer.Len() == 0 {
			w.pendingBuffer = nil
		}
		return copied, nil
	}

	// 2. 从底层连接读取数据
	n, err := w.Conn.Read(p)
	if n <= 0 || err != nil {
		// 代表这次请求真的结束了
		return n, err
	}

	// 3. 判断是否是 HTTP/2 preface
	if n >= 24 || w.prefetched {
		w.reqBuf.Write(p[:n])
		isHttp2 := w.prefetched
		isPrefaceDrop := false
		// 避免碎片化的数据导致的越界
		if !isHttp2 {
			preface := w.reqBuf.Bytes()[:24]
			isHttp2 = string(preface) == http2.ClientPreface
			if isHttp2 {
				w.prefetched = true
				w.reqBuf.Next(24) // 移除preface
				isPrefaceDrop = true
			}
		}

		if isHttp2 {
			originH2Content := string(p[:n])
			newBuf, pErr := w.parseHttp2Req()
			if pErr != nil {
				logger.Error(fmt.Sprintf("Error parsing HTTP/2 request: %v", pErr))
				// 相当于吹失败就直接放弃这次请求
				return n, pErr
			}
			// newBuf := w.reqBuf.Bytes()
			var finalBuf []byte
			if isPrefaceDrop {
				finalBuf = append([]byte(http2.ClientPreface), newBuf...)
			} else {
				finalBuf = newBuf
			}

			copied := copy(p, finalBuf)
			if copied < len(finalBuf) {
				w.pendingBuffer = bytes.NewBuffer(finalBuf[copied:])
			}
			endH2Content := string(finalBuf)
			fmt.Println("is the same? ->", originH2Content == endH2Content)
			if originH2Content != endH2Content {

			}
			return copied, nil
			// return n, nil
		} else {
			// HTTP/1.x 请求
			w.isHttp1 = true
			newReq, pErr := w.parseHttp1Req()
			if pErr != nil {
				// 反正http1全程就就一次，结束了亦可以将数据转发出去，所以0->n
				return n, pErr
			}

			copied := copy(p, newReq)
			if copied < len(newReq) {
				w.pendingBuffer = bytes.NewBuffer(newReq[copied:])
				return copied, nil
			}
			return copied, err
		}
	}

	// 数据还不够判断协议，等待下一轮
	return 0, nil
}

func (w *WatcherWrapConn) parseHttp1Req() ([]byte, error) {
	logger.Debug("📦 HTTP/1 connection preface detected")
	return w.processH1ReqHeaders()
}

func (w *WatcherWrapConn) parseHttp2Req() ([]byte, error) {
	logger.Debug("📦 HTTP/2 connection preface detected")
	preBuff := bytes.NewBuffer([]byte{})
	err := w.processHttp2RequestFrame(preBuff)
	if err != nil {
		logger.Error(fmt.Sprintf("Error processing HTTP/2 request frame: %v", err))
		// 直接丢弃
		result := append(preBuff.Bytes(), w.reqBuf.Bytes()...)
		return result, err
	}

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
	// if w.isHttp1 {
	// 	w.http1RespContent = string(p)
	// } else {
	// 	w.respBuf.Write(p[:n])
	// 	for {
	// 		frame, ok := w.tryExtractFrameFromBuf(&w.respBuf, true)
	// 		if ok {
	// 			w.processHttp2ResponseFrame(frame)
	// 		} else {
	// 			break
	// 		}
	// 	}
	// }

	return n, err
}

func (w *WatcherWrapConn) tryExtractFrameFromBuf(buf *bytes.Buffer, shouldMove bool) ([]byte, bool) {
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
