package inbound

import (
	"crypto/tls"
	"io"
	"net"
	"net/http"
	"sync"
	"time"

	"nursor.org/nursorgate/client/outbound"
	"nursor.org/nursorgate/client/server/tun/buffer"
	"nursor.org/nursorgate/client/utils"
	"nursor.org/nursorgate/common/logger"
)

const tcpWaitTimeout = time.Second * 60 * 10

func HandleTLSConnectionSimple(tlsConn *tls.Conn, req *http.Request) {
	logger.Info("parse ", req.Host)
	var isHttp2 = true
	alpnVersion := tlsConn.ConnectionState().NegotiatedProtocol
	if alpnVersion != "h2" {
		isHttp2 = false
	}
	outBoundClient, err := outbound.NewHttp2ProxyClient(utils.GetServerHost(), req.Host, isHttp2)
	if err != nil {
		logger.Error(err.Error(), req.Host)
		return
	}

	// err = outBoundClient.ForwardSimple(tlsConn, req.Host)
	// if err != nil {
	// 	logger.Error(err)
	// }
	pipe(tlsConn, outBoundClient.Conn)
}

// pipe copies data to & from provided net.Conn(s) bidirectionally.
func pipe(origin, remote net.Conn) {
	wg := sync.WaitGroup{}
	wg.Add(2)

	go unidirectionalStream(remote, origin, "origin->remote", &wg)
	go unidirectionalStream(origin, remote, "remote->origin", &wg)

	wg.Wait()
}

func unidirectionalStream(dst, src net.Conn, dir string, wg *sync.WaitGroup) {
	defer wg.Done()
	buf := buffer.Get(buffer.RelayBufferSize)
	if _, err := io.CopyBuffer(dst, src, buf); err != nil {
		//log.Debugf("[TCP] copy data for %s: %v", dir, err)
	}
	buffer.Put(buf)
	// Do the upload/download side TCP half-close.
	if cr, ok := src.(interface{ CloseRead() error }); ok {
		cr.CloseRead()
	}
	if cw, ok := dst.(interface{ CloseWrite() error }); ok {
		cw.CloseWrite()
	}
	// Set TCP half-close timeout.
	dst.SetReadDeadline(time.Now().Add(tcpWaitTimeout))
}
