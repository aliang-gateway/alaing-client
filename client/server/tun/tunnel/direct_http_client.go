package tunnel

import (
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net/http"
	"nursor.org/nursorgate/client/server/helper"
	"sync"
	"time"

	"golang.org/x/net/context"
	"golang.org/x/net/http2"
	"nursor.org/nursorgate/client/outbound/cert"
	"nursor.org/nursorgate/client/server/tun/dialer"
	"nursor.org/nursorgate/client/utils"
	"nursor.org/nursorgate/common/logger"
)

type OutboundClient2 struct {
	conn     *tls.Conn
	Tr       *http2.Transport
	streamID uint32
}

func (c *OutboundClient2) Forward(localConn *tls.Conn, req *http.Request) error {
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		wrapConn := &helper.WatcherWrapConn{Conn: localConn}
		n, err := io.Copy(c.conn, wrapConn)
		if err != nil {
			if !errors.Is(err, io.EOF) {
				logger.Error("--->remote", err.Error(), req)
			}
		}
		logger.Info(fmt.Sprintf("forwarded send %d bytes for host: %s", n, req.Host))
		err = c.conn.CloseWrite()
		//if err != nil {
		//	logger.Error("--->remote", err, req)
		//}
		wg.Done()
	}()
	go func() {

		n, err := io.Copy(localConn, c.conn)
		if err != nil {
			if errors.Is(err, io.EOF) {
				logger.Error("local<---", err.Error(), req)
			}
		}

		logger.Info(fmt.Sprintf("forwarded return %d bytes from host: %s", n, req.Host))
		err = localConn.CloseWrite()
		if err != nil {
			logger.Error("local<---", err, req)
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

	myCert := cert.GetOutboundCert(true, SNIName)
	tlsConfig := myCert.GetTLSConfig()
	tlsConfig.NextProtos = []string{"h2", "http/1.1"}
	tlsConn := tls.Client(conn, tlsConfig)
	err = tlsConn.Handshake()
	if err != nil {
		return nil, err
	}
	c := &OutboundClient2{conn: tlsConn}
	//err = c.PreHttp2AuthCheck()
	if err != nil {
		return nil, err
	}
	return c, nil
}
