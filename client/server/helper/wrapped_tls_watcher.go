package helper

import (
	"bytes"
	"encoding/binary"
	"net"

	"golang.org/x/net/http2"
	"golang.org/x/net/http2/hpack"
	"nursor.org/nursorgate/common/logger"
)

const (
	// Length of the HTTP/2 frame header
	frameHeaderLen = 9

	// HTTP/2 frame types
	frameTypeData    = 0x0
	frameTypeHeaders = 0x1
	frameTypeCont    = 0x9

	// HTTP/2 flags
	flagEndStream  = 0x1
	flagEndHeaders = 0x4
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
			if w.reqBuf.Len() < 24 || string(preface) != http2.ClientPreface {
				// logger.Debug("❌ Not a valid HTTP/2 connection preface")
				w.isHttp1 = true // 标记为已完成预取（虽然这不是 HTTP/2 请求）
				// 不是http2,就是http1
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
		frame, ok := w.tryParseFrameFromBuf(&w.respBuf)
		if ok {
			w.processHttp2ResponseFrame(frame)
		}
	} else {

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
