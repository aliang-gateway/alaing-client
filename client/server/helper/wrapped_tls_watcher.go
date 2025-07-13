package helper

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"golang.org/x/net/http2/hpack"
	"net"
)

type WatcherWrapConn struct {
	net.Conn
	buf      bytes.Buffer
	captured bool
}

func (w *WatcherWrapConn) Read(p []byte) (int, error) {
	n, err := w.Conn.Read(p)
	if n > 0 && !w.captured {
		w.buf.Write(p[:n])
		if tryExtractAuthorization(w.buf.Bytes()) {
			w.captured = true
		}
	}
	return n, err
}

// ========== HTTP/2 解析逻辑 =========

type FrameHeader struct {
	Length   uint32
	Type     uint8
	Flags    uint8
	StreamID uint32
}

func parseFrameHeader(data []byte) (*FrameHeader, error) {
	if len(data) < 9 {
		return nil, fmt.Errorf("too short")
	}
	length := binary.BigEndian.Uint32(append([]byte{0}, data[0:3]...))
	return &FrameHeader{
		Length:   length,
		Type:     data[3],
		Flags:    data[4],
		StreamID: binary.BigEndian.Uint32(data[5:9]) & 0x7FFFFFFF,
	}, nil
}

func decodeHeadersFragment(fragment []byte) (map[string]string, error) {
	headers := make(map[string]string)
	decoder := hpack.NewDecoder(4096, func(f hpack.HeaderField) {
		headers[f.Name] = f.Value
	})
	_, err := decoder.Write(fragment)
	return headers, err
}

func tryExtractAuthorization(data []byte) bool {
	for len(data) >= 9 {
		fh, err := parseFrameHeader(data)
		if err != nil || uint32(len(data)) < 9+fh.Length {
			break
		}
		if fh.Type == 0x1 { // HEADERS
			payload := data[9 : 9+fh.Length]
			headers, err := decodeHeadersFragment(payload)
			if err == nil {
				for k, v := range headers {
					if k == "authorization" {
						fmt.Println("🚨 Authorization:", v)
						return true
					}
				}
			}
		}
		data = data[9+fh.Length:]
	}
	return false
}
