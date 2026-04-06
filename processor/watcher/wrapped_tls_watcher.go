package tls

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"net"
	"strings"
	"sync"

	"aliang.one/nursorgate/common/logger"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/hpack"
)

const maxProtocolSniffBytes = 8192

var http1Methods = []string{
	"GET",
	"POST",
	"PUT",
	"HEAD",
	"PATCH",
	"DELETE",
	"OPTIONS",
	"CONNECT",
	"TRACE",
}

type WatcherWrapConn struct {
	net.Conn

	reqBuf           bytes.Buffer
	respBuf          bytes.Buffer
	prefetched       bool
	http2PrefaceSent bool
	isTokenFound     bool
	isHttp1          bool
	http1HeaderDone  bool
	passthrough      bool
	http1ReqContent  string
	http1RespContent string
	hpackDecoderReq  *hpack.Decoder
	hpackDecoderResp *hpack.Decoder

	toServerBuffer       *bytes.Buffer
	hpackEncoderToServer *hpack.Encoder

	streams   map[uint32]*http2Stream
	streamsMu sync.Mutex

	http2Settings map[uint16]uint32
	settingsMu    sync.Mutex

	pendingBuffer *bytes.Buffer

	mu                 sync.Mutex
	encoderToserverMap sync.Map
}

func NewWatcherWrapConn(conn net.Conn) *WatcherWrapConn {
	newBuffer := bytes.NewBuffer([]byte{})
	encoder := hpack.NewEncoder(newBuffer)

	return &WatcherWrapConn{
		Conn:                 conn,
		streams:              map[uint32]*http2Stream{},
		http2Settings:        map[uint16]uint32{},
		hpackDecoderReq:      hpack.NewDecoder(65536, nil),
		hpackDecoderResp:     hpack.NewDecoder(65536, nil),
		toServerBuffer:       newBuffer,
		hpackEncoderToServer: encoder,
	}
}

func (m *WatcherWrapConn) GetDecoder(streamID uint32, emitFunc func(hpack.HeaderField)) *hpack.Decoder {
	if d, ok := m.encoderToserverMap.Load(streamID); ok {
		decoder := d.(*hpack.Decoder)
		decoder.SetEmitFunc(emitFunc)
		return decoder
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if d, ok := m.encoderToserverMap.Load(streamID); ok {
		decoder := d.(*hpack.Decoder)
		decoder.SetEmitFunc(emitFunc)
		return decoder
	}

	decoder := hpack.NewDecoder(4096, emitFunc)
	m.encoderToserverMap.Store(streamID, decoder)
	return decoder
}

func (m *WatcherWrapConn) SetDecoder(streamID uint32, decoder hpack.Decoder) {
	m.encoderToserverMap.Store(streamID, decoder)
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
	for {
		if w.pendingBuffer != nil && w.pendingBuffer.Len() > 0 {
			return w.readFromPending(p)
		}

		if w.passthrough || (w.isHttp1 && w.http1HeaderDone) {
			if w.reqBuf.Len() > 0 {
				buffered := append([]byte(nil), w.reqBuf.Bytes()...)
				w.reqBuf.Reset()
				w.pendingBuffer = bytes.NewBuffer(buffered)
				continue
			}
			return w.Conn.Read(p)
		}

		out, progressed, err := w.prepareBufferedOutput()
		if err != nil {
			return 0, err
		}
		if len(out) > 0 {
			w.pendingBuffer = bytes.NewBuffer(out)
			continue
		}
		if progressed {
			continue
		}

		readSize := len(p)
		if readSize < 1 {
			readSize = 1
		}
		tmp := make([]byte, readSize)
		n, err := w.Conn.Read(tmp)
		if n > 0 {
			w.reqBuf.Write(tmp[:n])
		}
		if err != nil {
			if n > 0 {
				continue
			}
			return 0, err
		}
	}
}

func (w *WatcherWrapConn) readFromPending(p []byte) (int, error) {
	copied := copy(p, w.pendingBuffer.Bytes())
	w.pendingBuffer.Next(copied)
	if w.pendingBuffer.Len() == 0 {
		w.pendingBuffer = nil
	}
	return copied, nil
}

func (w *WatcherWrapConn) prepareBufferedOutput() ([]byte, bool, error) {
	if w.prefetched {
		out, err := w.parseHttp2Req()
		if err != nil {
			logger.Error(fmt.Sprintf("Error parsing HTTP/2 request: %v", err))
			return nil, false, err
		}
		if !w.http2PrefaceSent {
			w.http2PrefaceSent = true
			return append([]byte(http2.ClientPreface), out...), true, nil
		}
		if len(out) > 0 {
			return out, true, nil
		}
		return nil, false, nil
	}

	if w.isHttp1 {
		out, ready, err := w.parseHttp1Req()
		if err != nil {
			return nil, false, err
		}
		if ready {
			return out, true, nil
		}
		return nil, false, nil
	}

	decision, decided := classifyBufferedProtocol(w.reqBuf.Bytes())
	if !decided {
		if w.reqBuf.Len() >= maxProtocolSniffBytes {
			w.passthrough = true
			return nil, true, nil
		}
		return nil, false, nil
	}

	switch decision {
	case "http2":
		w.prefetched = true
		w.reqBuf.Next(len(http2.ClientPreface))
		logger.Debug("HTTP/2 connection preface detected")
		return nil, true, nil
	case "http1":
		w.isHttp1 = true
		logger.Debug("HTTP/1 connection preface detected")
		return nil, true, nil
	default:
		w.passthrough = true
		return nil, true, nil
	}
}

func classifyBufferedProtocol(buf []byte) (string, bool) {
	if len(buf) == 0 {
		return "", false
	}

	preface := []byte(http2.ClientPreface)
	if len(buf) >= len(preface) && bytes.Equal(preface, buf[:len(preface)]) {
		return "http2", true
	}
	if len(buf) < len(preface) && bytes.Equal(preface[:len(buf)], buf) {
		return "", false
	}

	upperPrefix := strings.ToUpper(string(buf))
	for _, method := range http1Methods {
		switch {
		case strings.HasPrefix(upperPrefix, method+" "):
			return "http1", true
		case strings.HasPrefix(method, upperPrefix):
			return "", false
		}
	}

	if len(buf) < len(preface) {
		return "", false
	}
	return "passthrough", true
}

func (w *WatcherWrapConn) parseHttp1Req() ([]byte, bool, error) {
	return w.processH1ReqHeaders()
}

func (w *WatcherWrapConn) parseHttp2Req() ([]byte, error) {
	preBuff := bytes.NewBuffer(nil)
	if err := w.processHttp2RequestFrame(preBuff); err != nil {
		logger.Error(fmt.Sprintf("Error processing HTTP/2 request frame: %v", err))
		return nil, err
	}
	return preBuff.Bytes(), nil
}

func (w *WatcherWrapConn) Write(p []byte) (n int, err error) {
	n, err = w.Conn.Write(p)
	if err != nil {
		return n, err
	}
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
