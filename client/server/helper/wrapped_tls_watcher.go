package helper

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"log"
	"net"

	"golang.org/x/net/http2/hpack"
	"nursor.org/nursorgate/client/user"
)

const (
	frameHeaderLen   = 9
	frameTypeHeaders = 0x1
	frameTypeCont    = 0x9
	flagEndHeaders   = 0x4
)

type WatcherWrapConn struct {
	net.Conn
	buf          bytes.Buffer
	prefetched   bool
	isTokenFound bool
}

func (w *WatcherWrapConn) Read(p []byte) (int, error) {
	n, err := w.Conn.Read(p)
	if n > 0 && !w.isTokenFound {
		// step 1: prefetch the client preface (24 bytes)
		if !w.prefetched {
			w.buf.Write(p[:n])
			if w.buf.Len() < 24 {
				return n, err // wait for more
			}
			preface := w.buf.Next(24)
			if string(preface) != "PRI * HTTP/2.0\r\n\r\nSM\r\n\r\n" {
				log.Println("❌ Not a valid HTTP/2 connection preface")
				return n, fmt.Errorf("invalid preface")
			}
			w.prefetched = true
			log.Println("✅ HTTP/2 connection preface detected")
			if w.buf.Len() == 0 {
				return n, err // preface done, no other data
			}
		} else {
			w.buf.Write(p[:n])
		}

		for {
			frame, ok := w.tryParseFrame()
			if !ok {
				break
			}
			w.processFrame(frame)
		}
	}
	return n, err
}

func (w *WatcherWrapConn) tryParseFrame() ([]byte, bool) {
	data := w.buf.Bytes()
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
	w.buf.Next(totalLen)
	return frame, true
}

func (w *WatcherWrapConn) processFrame(frame []byte) {
	length := binary.BigEndian.Uint32(append([]byte{0}, frame[0:3]...))
	ftype := frame[3]
	flags := frame[4]
	streamID := binary.BigEndian.Uint32(frame[5:9]) & 0x7FFFFFFF
	log.Printf("Frame len=%d type=%d stream=%d", length, ftype, streamID)
	payload := frame[frameHeaderLen:]

	if ftype == frameTypeHeaders {
		headerBlock := append([]byte{}, payload...)
		if flags&flagEndHeaders == 0 {
			// Continue collecting CONTINUATION frames
			for {
				contFrame, ok := w.tryParseFrame()
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
				// fmt.Println("\u2705 Authorization:", val)
			}
		}
	}
}

func decodeHeaderBlock(block []byte) (map[string]string, error) {
	headers := make(map[string]string)
	decoder := hpack.NewDecoder(4096, func(hf hpack.HeaderField) {
		headers[hf.Name] = hf.Value
	})
	_, err := decoder.Write(block)
	return headers, err
}
