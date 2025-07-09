package listener

import (
	"crypto/tls"
	"errors"
	"fmt"
	"io"

	"golang.org/x/net/http2"
	"nursor.org/nursorgate/client/outbound"
	"nursor.org/nursorgate/common/logger"
	"nursor.org/nursorgate/server/handler"
)

func HandleTLSConnection(tlsConn *tls.Conn) {
	host := tlsConn.ConnectionState().ServerName
	if host == "" {
		host = tlsConn.ConnectionState().PeerCertificates[0].Subject.CommonName
	}
	logger.Info("Detected HTTP/1 connection")
	if tlsConn.ConnectionState().NegotiatedProtocol == "h2" {
		logger.Info("Detected HTTP/2 connection")
		buf := make([]byte, len(http2.ClientPreface))
		n, err := io.ReadFull(tlsConn, buf)
		if err != nil && err != io.EOF && !errors.Is(err, io.ErrUnexpectedEOF) {
			logger.Error(fmt.Printf("Failed to read connection preface: %v", err))
			tlsConn.Close()
			return
		}

		// 将读取的数据放回连接
		wrapedTlsConn := &outbound.WrappedTLSConn{
			Conn: tlsConn,
			Buf:  buf[:n],
		}
		//检查是否是 HTTP/2
		if string(buf[:n]) == http2.ClientPreface {
			// 处理 HTTP/2 连接
			logger.Info("Detected HTTP/2 connection")

			if err != nil {
				err = tlsConn.Close()
				if err != nil {
					logger.Error(err)
				}
			}
			return
		} else {
			// http1.1
			handler.ForwardHttpDirect(wrapedTlsConn, host)
		}
	} else {

		handler.ForwardHttpDirect(tlsConn, host)
	}

}
