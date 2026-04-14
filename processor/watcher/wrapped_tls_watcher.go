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
	passthrough      bool
	http1ReqContent  string
	http1RespContent string
	http1BodyTracker *http1BodyTracker

	// Connection-scoped HPACK lifecycle for request path:
	// client encoder -> proxy requestDecoderFromClient -> proxy requestEncoderToServer -> server decoder
	requestDecoderFromClient *hpack.Decoder
	requestEncoderBuffer     *bytes.Buffer
	requestEncoderToServer   *hpack.Encoder

	// Connection-scoped HPACK lifecycle for response path:
	// server encoder -> proxy responseDecoderFromServer -> proxy responseEncoderToClient -> client decoder
	//
	// responseEncoderToClient is prepared for response-side header rewriting and must
	// track the client-advertised SETTINGS even when current logic only observes
	// responses instead of re-encoding them.
	responseDecoderFromServer *hpack.Decoder
	responseEncoderBuffer     *bytes.Buffer
	responseEncoderToClient   *hpack.Encoder

	streams   map[uint32]*http2Stream
	streamsMu sync.Mutex

	clientHTTP2Settings map[uint16]uint32
	serverHTTP2Settings map[uint16]uint32
	settingsMu          sync.Mutex

	pendingBuffer *bytes.Buffer
}

func NewWatcherWrapConn(conn net.Conn) *WatcherWrapConn {
	requestBuffer := bytes.NewBuffer([]byte{})
	requestEncoder := hpack.NewEncoder(requestBuffer)
	responseBuffer := bytes.NewBuffer([]byte{})
	responseEncoder := hpack.NewEncoder(responseBuffer)

	return &WatcherWrapConn{
		Conn:                      conn,
		streams:                   map[uint32]*http2Stream{},
		clientHTTP2Settings:       map[uint16]uint32{},
		serverHTTP2Settings:       map[uint16]uint32{},
		requestDecoderFromClient:  hpack.NewDecoder(65536, nil),
		requestEncoderBuffer:      requestBuffer,
		requestEncoderToServer:    requestEncoder,
		responseDecoderFromServer: hpack.NewDecoder(65536, nil),
		responseEncoderBuffer:     responseBuffer,
		responseEncoderToClient:   responseEncoder,
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
	for {
		if w.pendingBuffer != nil && w.pendingBuffer.Len() > 0 {
			return w.readFromPending(p)
		}

		if w.passthrough {
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
		if w.http1BodyTracker != nil {
			out, progressed, err := w.consumeHTTP1Body()
			if err != nil {
				return nil, false, err
			}
			if len(out) > 0 {
				return out, true, nil
			}
			if progressed {
				return nil, true, nil
			}
			return nil, false, nil
		}

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
	if n > 0 && w.prefetched && !w.isHttp1 && !w.passthrough {
		w.respBuf.Write(p[:n])
		if parseErr := w.processBufferedHTTP2Responses(); parseErr != nil {
			logger.Warn(fmt.Sprintf("Error processing HTTP/2 response frame: %v", parseErr))
		}
	}
	return n, err
}

func (w *WatcherWrapConn) processBufferedHTTP2Responses() error {
	for {
		frame, ok := w.tryExtractFrameFromBuf(&w.respBuf, true)
		if !ok {
			return nil
		}
		if err := w.processHttp2ResponseFrame(frame); err != nil {
			return err
		}
	}
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
