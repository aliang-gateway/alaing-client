package tls

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"sync/atomic"
	"time"

	"aliang.one/nursorgate/common/logger"
	"aliang.one/nursorgate/common/version"
	user "aliang.one/nursorgate/processor/auth"
)

const httpRelayCaptureLimit = 128 * 1024

type HTTP1RelayStats struct {
	StartedAt          time.Time
	FirstResponseAt    time.Time
	CompletedAt        time.Time
	ClientToServerByte int64
	ServerToClientByte int64
	RequestPayload     []byte
	ResponsePayload    []byte
}

type httpRelayCaptureBuffer struct {
	buf   bytes.Buffer
	limit int
}

func newHTTPRelayCaptureBuffer(limit int) *httpRelayCaptureBuffer {
	if limit <= 0 {
		limit = httpRelayCaptureLimit
	}
	return &httpRelayCaptureBuffer{limit: limit}
}

func (p *httpRelayCaptureBuffer) Write(data []byte) {
	if p == nil || len(data) == 0 {
		return
	}
	remaining := p.limit - p.buf.Len()
	if remaining <= 0 {
		return
	}
	if len(data) > remaining {
		data = data[:remaining]
	}
	_, _ = p.buf.Write(data)
}

func (p *httpRelayCaptureBuffer) Bytes() []byte {
	if p == nil {
		return nil
	}
	out := make([]byte, p.buf.Len())
	copy(out, p.buf.Bytes())
	return out
}

type httpRelayCountingWriter struct {
	writer      io.Writer
	capture     *httpRelayCaptureBuffer
	onFirstData func()
	written     *int64
}

func (w *httpRelayCountingWriter) Write(p []byte) (int, error) {
	if w.onFirstData != nil && len(p) > 0 {
		w.onFirstData()
		w.onFirstData = nil
	}
	if w.capture != nil && len(p) > 0 {
		w.capture.Write(p)
	}
	n, err := w.writer.Write(p)
	if n > 0 && w.written != nil {
		atomic.AddInt64(w.written, int64(n))
	}
	return n, err
}

func RelayHTTP1(ctx context.Context, clientConn, remoteConn net.Conn) (*HTTP1RelayStats, error) {
	stats := &HTTP1RelayStats{StartedAt: time.Now()}
	requestCapture := newHTTPRelayCaptureBuffer(httpRelayCaptureLimit)
	responseCapture := newHTTPRelayCaptureBuffer(httpRelayCaptureLimit)

	var firstResponseNano int64
	markFirstResponse := func() {
		if atomic.CompareAndSwapInt64(&firstResponseNano, 0, time.Now().UnixNano()) {
			stats.FirstResponseAt = time.Unix(0, atomic.LoadInt64(&firstResponseNano))
		}
	}

	clientReader := bufio.NewReader(clientConn)
	remoteReader := bufio.NewReader(remoteConn)
	requestWriter := &httpRelayCountingWriter{
		writer:  remoteConn,
		capture: requestCapture,
		written: &stats.ClientToServerByte,
	}
	responseWriter := &httpRelayCountingWriter{
		writer:      clientConn,
		capture:     responseCapture,
		onFirstData: markFirstResponse,
		written:     &stats.ServerToClientByte,
	}

	var relayErr error
	for {
		select {
		case <-ctx.Done():
			relayErr = ctx.Err()
			goto done
		default:
		}

		req, err := http.ReadRequest(clientReader)
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			relayErr = err
			break
		}

		injectHTTP1AuthorizationHeader(req)

		if err := req.Write(requestWriter); err != nil {
			relayErr = err
			break
		}

		resp, err := http.ReadResponse(remoteReader, req)
		if err != nil {
			if isNetTimeout(err) {
				if writeErr := writeHTTP1GatewayTimeout(responseWriter, req); writeErr != nil {
					relayErr = fmt.Errorf("http1 relay timeout while writing local gateway timeout: %w", writeErr)
				} else {
					relayErr = err
				}
			} else {
				relayErr = err
			}
			break
		}

		if err := resp.Write(responseWriter); err != nil {
			relayErr = err
			break
		}

		if shouldCloseHTTP1Relay(req, resp) {
			break
		}
	}

done:
	closeHTTPWrite(clientConn)
	closeHTTPRead(clientConn)
	closeHTTPWrite(remoteConn)
	closeHTTPRead(remoteConn)

	stats.CompletedAt = time.Now()
	stats.RequestPayload = requestCapture.Bytes()
	stats.ResponsePayload = responseCapture.Bytes()

	return stats, relayErr
}

func injectHTTP1AuthorizationHeader(req *http.Request) {
	if req == nil {
		return
	}

	if authHeader := strings.TrimSpace(user.GetCurrentAuthorizationHeader()); authHeader != "" {
		req.Header.Set("Authorization-Inner", authHeader)
	}

	requestLine := fmt.Sprintf("%s %s %s", req.Method, req.RequestURI, req.Proto)
	if req.Header.Get("Authorization-Inner") == "" {
		logger.Warn(fmt.Sprintf(
			"WatcherWrapConn: missing authorization-inner after HTTP/1 relay request=%q host=%q",
			requestLine,
			req.Host,
		))
	} else if !version.IsProdBuild() {
		logger.Info(fmt.Sprintf(
			"WatcherWrapConn: added authorization-inner for HTTP/1 relay request=%q host=%q",
			requestLine,
			req.Host,
		))
	}
}

func shouldCloseHTTP1Relay(req *http.Request, resp *http.Response) bool {
	if req == nil || resp == nil {
		return true
	}
	if req.Close || resp.Close {
		return true
	}
	if strings.EqualFold(req.Header.Get("Connection"), "close") {
		return true
	}
	if strings.EqualFold(resp.Header.Get("Connection"), "close") {
		return true
	}
	return false
}

func writeHTTP1GatewayTimeout(w io.Writer, req *http.Request) error {
	body := "gateway timeout"
	resp := &http.Response{
		Status:        "504 Gateway Timeout",
		StatusCode:    http.StatusGatewayTimeout,
		Proto:         "HTTP/1.1",
		ProtoMajor:    1,
		ProtoMinor:    1,
		Header:        make(http.Header),
		Body:          io.NopCloser(strings.NewReader(body)),
		ContentLength: int64(len(body)),
		Close:         true,
		Request:       req,
	}
	resp.Header.Set("Content-Type", "text/plain; charset=utf-8")
	resp.Header.Set("Connection", "close")
	return resp.Write(w)
}

func isNetTimeout(err error) bool {
	var netErr net.Error
	return errors.As(err, &netErr) && netErr.Timeout()
}

func closeHTTPWrite(conn net.Conn) {
	if conn == nil {
		return
	}
	if cw, ok := conn.(interface{ CloseWrite() error }); ok {
		_ = cw.CloseWrite()
	}
}

func closeHTTPRead(conn net.Conn) {
	if conn == nil {
		return
	}
	if cr, ok := conn.(interface{ CloseRead() error }); ok {
		_ = cr.CloseRead()
	}
}
