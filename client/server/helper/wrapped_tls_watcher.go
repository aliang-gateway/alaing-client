package helper

import (
	"bytes"
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

	streams   map[uint32]*http2Stream // 存储活跃的 HTTP/2 流
	streamsMu sync.Mutex              // 保护 streams map 的并发访问

}

func (w *WatcherWrapConn) Read(p []byte) (int, error) {
	n, err := w.Conn.Read(p)
	if n > 0 && !w.isTokenFound {
		w.reqBuf.Write(p[:n])

		if len(p) < 24 || string(p[:24]) != http2.ClientPreface {
			// http1
			logger.Info("📦 HTTP/1 connection preface detected")
			w.isHttp1 = true
			if !w.isTokenFound && w.isHttp1 {
				if err := w.processH1ReqHeaders(); err != nil {
					return n, err
				}
			}
		} else {
			w.prefetched = true
			logger.Debug("✅ HTTP/2 connection preface detected")
			if w.reqBuf.Len() == 0 {
				return n, err // preface done, no other data
			}
			for {
				frame, ok := w.tryParseFrameFromBuf(&w.reqBuf)
				if !ok {
					break
				}
				w.processHttp2Frame(frame)
			}
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
	if w.isHttp1 {
		w.http1RespContent = string(p)
		logger.HttpInfo(fmt.Sprintf("-----------starth1----------------\n%s--->--<---\n%s-----------------endh1------------------\n\n", w.http1ReqContent, w.http1RespContent))

	} else {
		w.respBuf.Write(p[:n])
		for {
			frame, ok := w.tryParseFrameFromBuf(&w.respBuf)
			if ok {
				w.processHttp2ResponseFrame(frame)
			} else {
				break
			}
		}

		for streaId, payload := range w.streams {
			if payload.RespEndStream {
				logger.HttpInfo(fmt.Sprintf("-----------starth2----------------\n%v-------------------------endh2------------------\n\n", w.streams[streaId]))
				delete(w.streams, streaId)
			}
		}

	}

	return n, err
}

func (w *WatcherWrapConn) tryParseFrameFromBuf(buf *bytes.Buffer) ([]byte, bool) {
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

func decodeHeaderBlock(block []byte) (map[string]string, error) {
	headers := make(map[string]string)
	decoder := hpack.NewDecoder(4096, func(hf hpack.HeaderField) {
		headers[hf.Name] = hf.Value
	})
	_, err := decoder.Write(block)
	return headers, err
}
