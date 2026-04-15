package tunnel

import (
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"aliang.one/nursorgate/common/logger"
	"aliang.one/nursorgate/inbound/tun/dialer"
	cert_server "aliang.one/nursorgate/processor/cert/server"
	"aliang.one/nursorgate/processor/config"
	watcher "aliang.one/nursorgate/processor/watcher"
	"golang.org/x/net/context"
	"golang.org/x/net/http2"
)

type OutboundClient2 struct {
	conn     *tls.Conn
	Tr       *http2.Transport
	streamID uint32
}

type http2RelayResult struct {
	direction string
	bytes     int64
	err       error
	duration  time.Duration
}

var http2ForwardCounter uint64

func (c *OutboundClient2) Forward(localConn *tls.Conn, req *http.Request) error {
	wrapConn := watcher.NewWatcherWrapConn(localConn)
	forwardID := atomic.AddUint64(&http2ForwardCounter, 1)
	startedAt := time.Now()

	logger.Debug(fmt.Sprintf(
		"[HTTP/2 FORWARD] conn=%d start method=%s host=%s path=%s local=%s remote=%s local_tls=%s remote_tls=%s",
		forwardID,
		safeRequestMethod(req),
		safeRequestHost(req),
		safeRequestPath(req),
		describeNetConn(localConn),
		describeNetConn(c.conn),
		describeTLSConn(localConn),
		describeTLSConn(c.conn),
	))

	results := make(chan http2RelayResult, 2)
	copyStream := func(direction string, dst io.Writer, src io.Reader) {
		streamStartedAt := time.Now()
		n, err := io.Copy(dst, src)
		results <- http2RelayResult{
			direction: direction,
			bytes:     n,
			err:       err,
			duration:  time.Since(streamStartedAt),
		}
	}

	go copyStream("client->gateway", c.conn, wrapConn)
	go copyStream("gateway->client", wrapConn, c.conn)

	var closeOnce sync.Once
	closeBoth := func(trigger string) {
		closeOnce.Do(func() {
			logger.Warn(fmt.Sprintf(
				"[HTTP/2 FORWARD] conn=%d teardown trigger=%s strategy=full-close local=%s remote=%s",
				forwardID,
				trigger,
				describeNetConn(localConn),
				describeNetConn(c.conn),
			))
			if err := c.conn.Close(); err != nil && !isIgnorableRelayErr(err) {
				logger.Warn(fmt.Sprintf("[HTTP/2 FORWARD] conn=%d remote close error: %v", forwardID, err))
			}
			if err := localConn.Close(); err != nil && !isIgnorableRelayErr(err) {
				logger.Warn(fmt.Sprintf("[HTTP/2 FORWARD] conn=%d local close error: %v", forwardID, err))
			}
		})
	}

	var firstSignificantErr error
	for i := 0; i < 2; i++ {
		result := <-results
		logHTTP2RelayResult(forwardID, result)

		if i == 0 {
			// HTTP/2 is multiplexed over a single TCP/TLS connection, so avoid TCP half-close.
			// Once one direction exits we tear down the full connection pair instead of sending
			// a partial FIN that could leave sibling streams in an ambiguous state.
			closeBoth("stream-exit:" + result.direction)
		}

		if firstSignificantErr == nil && !isIgnorableRelayErr(result.err) {
			firstSignificantErr = result.err
		}
	}

	logger.Debug(fmt.Sprintf(
		"[HTTP/2 FORWARD] conn=%d finished duration=%s err=%v",
		forwardID,
		time.Since(startedAt).Round(time.Millisecond),
		firstSignificantErr,
	))
	return firstSignificantErr
}

func handleTlsConnect(conn *tls.Conn, req *http.Request) {
	outboundClient, err := NewHttp2ProxyClient(config.GetCursorAiGatewayHost(), req.Host)
	if err != nil {
		logger.Error(err)
		return
	}
	err = outboundClient.Forward(conn, req)
	if err != nil {
		logger.Error(err)
	}
}

func NewHttp2ProxyClient(server string, SNIName string) (*OutboundClient2, error) {
	ctx := context.Background()
	timeoutCtx, _ := context.WithTimeout(ctx, time.Second)
	conn, err := dialer.DialContext(timeoutCtx, "tcp", server)
	if err != nil {
		return nil, err
	}

	myCert := cert_server.GetOutboundCert(true, SNIName)
	tlsConfig := myCert.GetTLSConfig()
	tlsConfig.NextProtos = []string{"h2", "http/1.1"}
	tlsConn := tls.Client(conn, tlsConfig)
	err = tlsConn.Handshake()
	if err != nil {
		return nil, err
	}
	c := &OutboundClient2{conn: tlsConn}
	return c, nil
}

func logHTTP2RelayResult(forwardID uint64, result http2RelayResult) {
	if result.err == nil {
		logger.Debug(fmt.Sprintf(
			"[HTTP/2 FORWARD] conn=%d stream_end dir=%s bytes=%d duration=%s err=nil",
			forwardID,
			result.direction,
			result.bytes,
			result.duration.Round(time.Millisecond),
		))
		return
	}

	errType := classifyRelayErr(result.err)
	message := fmt.Sprintf(
		"[HTTP/2 FORWARD] conn=%d stream_end dir=%s bytes=%d duration=%s err_type=%s err=%v",
		forwardID,
		result.direction,
		result.bytes,
		result.duration.Round(time.Millisecond),
		errType,
		result.err,
	)
	if errType == "timeout" || errType == "eof" || errType == "closed" {
		logger.Debug(message)
		return
	}
	logger.Warn(message)
}

func classifyRelayErr(err error) string {
	switch {
	case err == nil:
		return "none"
	case errors.Is(err, io.EOF):
		return "eof"
	case isNetTimeoutError(err):
		return "timeout"
	case errors.Is(err, net.ErrClosed):
		return "closed"
	default:
		return "other"
	}
}

func isIgnorableRelayErr(err error) bool {
	if err == nil {
		return true
	}
	return errors.Is(err, io.EOF) || errors.Is(err, net.ErrClosed) || isNetTimeoutError(err)
}

func isNetTimeoutError(err error) bool {
	var netErr net.Error
	return errors.As(err, &netErr) && netErr.Timeout()
}

func describeNetConn(conn net.Conn) string {
	if conn == nil {
		return "nil"
	}
	localAddr := "unknown"
	remoteAddr := "unknown"
	if addr := conn.LocalAddr(); addr != nil {
		localAddr = addr.String()
	}
	if addr := conn.RemoteAddr(); addr != nil {
		remoteAddr = addr.String()
	}
	return fmt.Sprintf("%s->%s", localAddr, remoteAddr)
}

func describeTLSConn(conn *tls.Conn) string {
	if conn == nil {
		return "nil"
	}
	state := conn.ConnectionState()
	serverName := state.ServerName
	if serverName == "" {
		serverName = "unknown"
	}
	proto := state.NegotiatedProtocol
	if proto == "" {
		proto = "unknown"
	}
	return fmt.Sprintf("sni=%s alpn=%s vers=0x%04x resumed=%t", serverName, proto, state.Version, state.DidResume)
}

func safeRequestMethod(req *http.Request) string {
	if req == nil || req.Method == "" {
		return "unknown"
	}
	return req.Method
}

func safeRequestHost(req *http.Request) string {
	if req == nil {
		return "unknown"
	}
	if req.Host != "" {
		return req.Host
	}
	if req.URL != nil && req.URL.Host != "" {
		return req.URL.Host
	}
	return "unknown"
}

func safeRequestPath(req *http.Request) string {
	if req == nil || req.URL == nil || req.URL.Path == "" {
		return "/"
	}
	return req.URL.Path
}
