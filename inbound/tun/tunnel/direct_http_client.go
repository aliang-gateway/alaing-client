package tunnel

import (
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"sync"
	"time"

	"golang.org/x/net/context"
	"golang.org/x/net/http2"
	"nursor.org/nursorgate/common/logger"
	"nursor.org/nursorgate/inbound/tun/dialer"
	"nursor.org/nursorgate/inbound/tun/runner/utils"
	cert_server "nursor.org/nursorgate/processor/cert/server"
	watcher "nursor.org/nursorgate/processor/watcher"
)

type OutboundClient2 struct {
	conn     *tls.Conn
	Tr       *http2.Transport
	streamID uint32
}

func (c *OutboundClient2) Forward(localConn *tls.Conn, req *http.Request) error {
	var wg sync.WaitGroup
	wg.Add(2)
	wrapConn := watcher.NewWatcherWrapConn(localConn)
	// wrapConn := localConn

	go func() {
		// 奇怪得是本地得转发到server得竟然有timeout得情况，不理解

		n, err := io.Copy(c.conn, wrapConn)
		if err != nil {
			if ne, ok := err.(net.Error); !ok || !ne.Timeout() {
				// 忽略 timeout 错误
				logger.Warn("--->remote", err.Error(), req)
			}
		}
		logger.Debug(fmt.Sprintf("forwarded send %d bytes for host: %s", n, req.Host))
		c.conn.CloseWrite()

		wg.Done()
	}()
	go func() {
		n, err := io.Copy(wrapConn, c.conn)
		if err != nil {
			if errors.Is(err, io.EOF) {
				logger.Warn("local<---", err.Error(), req)
			}
		}
		logger.Debug(fmt.Sprintf("forwarded return %d bytes from host: %s", n, req.Host))
		err = localConn.CloseWrite()
		if err != nil {
			logger.Warn("local<---", err, req)
		}
		wg.Done()
	}()
	wg.Wait()
	return nil
}

func handleTlsConnect(conn *tls.Conn, req *http.Request) {
	outboundClient, err := NewHttp2ProxyClient(utils.GetServerHost(), req.Host)
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
	//err = c.PreHttp2AuthCheck()
	// if err != nil {
	// 	return nil, err
	// }
	return c, nil
}
