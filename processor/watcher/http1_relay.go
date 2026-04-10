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

type httpRequestResult struct {
	req *http.Request
	err error
}

type http1RemoteStream struct {
	reader *io.PipeReader
	writer *io.PipeWriter
	events chan error
}

func startHTTP1RemoteStream(
	remoteConn net.Conn,
	capture *httpRelayCaptureBuffer,
	onFirstData func(),
	written *int64,
) *http1RemoteStream {
	pipeReader, pipeWriter := io.Pipe()
	stream := &http1RemoteStream{
		reader: pipeReader,
		writer: pipeWriter,
		events: make(chan error, 1),
	}

	go func() {
		defer close(stream.events)

		buf := make([]byte, 32*1024)
		localOnFirstData := onFirstData
		for {
			n, err := remoteConn.Read(buf)
			if n > 0 {
				if localOnFirstData != nil {
					localOnFirstData()
					localOnFirstData = nil
				}
				if capture != nil {
					capture.Write(buf[:n])
				}
				if written != nil {
					atomic.AddInt64(written, int64(n))
				}
				if _, writeErr := pipeWriter.Write(buf[:n]); writeErr != nil {
					stream.events <- writeErr
					_ = pipeWriter.CloseWithError(writeErr)
					return
				}
			}
			if err != nil {
				stream.events <- err
				_ = pipeWriter.CloseWithError(err)
				return
			}
		}
	}()

	return stream
}

func (s *http1RemoteStream) Reader() io.Reader {
	if s == nil {
		return nil
	}
	return s.reader
}

func (s *http1RemoteStream) Events() <-chan error {
	if s == nil {
		return nil
	}
	return s.events
}

func (s *http1RemoteStream) Close() {
	if s == nil {
		return
	}
	_ = s.reader.Close()
	_ = s.writer.Close()
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
	requestWriter := &httpRelayCountingWriter{
		writer:  remoteConn,
		capture: requestCapture,
		written: &stats.ClientToServerByte,
	}
	responseWriter := &httpRelayCountingWriter{
		writer: clientConn,
	}

	remoteStream := startHTTP1RemoteStream(remoteConn, responseCapture, markFirstResponse, &stats.ServerToClientByte)
	defer remoteStream.Close()
	remoteReader := bufio.NewReader(remoteStream.Reader())

	var relayErr error
	for {
		reqCh := make(chan httpRequestResult, 1)
		go func() {
			req, err := http.ReadRequest(clientReader)
			reqCh <- httpRequestResult{req: req, err: err}
		}()

		select {
		case <-ctx.Done():
			relayErr = ctx.Err()
			goto done
		case remoteErr, ok := <-remoteStream.Events():
			if ok {
				relayErr = normalizeIdleRemoteClose(remoteErr)
			}
			goto done
		case reqRes := <-reqCh:
			if reqRes.err != nil {
				if errors.Is(reqRes.err, io.EOF) {
					goto done
				}
				relayErr = reqRes.err
				goto done
			}

			injectHTTP1AuthorizationHeader(reqRes.req)

			if err := reqRes.req.Write(requestWriter); err != nil {
				relayErr = err
				goto done
			}

			resp, err := http.ReadResponse(remoteReader, reqRes.req)
			if err != nil {
				if isNetTimeout(err) {
					if writeErr := writeHTTP1GatewayTimeout(responseWriter, reqRes.req); writeErr != nil {
						relayErr = fmt.Errorf("http1 relay timeout while writing local gateway timeout: %w", writeErr)
					} else {
						relayErr = err
					}
				} else {
					relayErr = err
				}
				goto done
			}

			if err := resp.Write(responseWriter); err != nil {
				relayErr = err
				goto done
			}

			if shouldCloseHTTP1Relay(reqRes.req, resp) {
				goto done
			}
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

	return stats, normalizeIdleRemoteClose(relayErr)
}

func normalizeIdleRemoteClose(err error) error {
	if err == nil || errors.Is(err, io.EOF) || errors.Is(err, net.ErrClosed) {
		return nil
	}
	return err
}

func injectHTTP1AuthorizationHeader(req *http.Request) {
	if req == nil {
		return
	}

	rewriteAliangHTTPRequestHost(req)

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
		logger.Debug(fmt.Sprintf(
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
