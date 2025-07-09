package inbound

import (
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"strings"

	"golang.org/x/net/http2"
	"nursor.org/nursorgate/client/inbound/forward"
	"nursor.org/nursorgate/client/outbound"
	"nursor.org/nursorgate/common/logger"
)

func HandleTLSConnection(tlsConn *tls.Conn, req *http.Request) {
	buf := make([]byte, len(http2.ClientPreface))
	if strings.Contains(req.Host, "repo42.cursor.sh") {
		logger.Info("reading get an api42 info 0")
	}
	n, err := io.ReadFull(tlsConn, buf)
	if err != nil && err != io.EOF && !errors.Is(err, io.ErrUnexpectedEOF) {
		logger.Error(fmt.Printf("Failed to read connection preface: %v", err))
		tlsConn.Close()
		return
	}
	if strings.Contains(req.Host, "repo42.cursor.sh") {
		logger.Info("get an api42 info 1")
	}

	// 将读取的数据放回连接
	wrapedTlsConn := &outbound.WrappedTLSConn{
		Conn: tlsConn,
		Buf:  buf[:n],
	}

	//检查是否是 HTTP/2
	if string(buf[:n]) == http2.ClientPreface {
		logger.Info("Detected HTTP/2 connection")
		err = forward.HandleHttp2(tlsConn, req)
		if err != nil {
			err = tlsConn.Close()
			if err != nil {
				logger.Error(err)
			}
		}
		return
	}

	//if tlsConn.ConnectionState().NegotiatedProtocol == "h2" {
	//	logger.Info("Detected HTTP/2 connection")
	//	err := forward.HandleHttp2(tlsConn, req)
	//	if err != nil {
	//		logger.Error(err)
	//	}
	//	return
	//}

	// 开始http1的处理----------------
	clientReader := forward.NewRequestReader(false, wrapedTlsConn)
	for !clientReader.IsEOF() {
		newReq, err := clientReader.ReadRequest()
		if err != nil {
			logger.Error(err.Error())
			return
		}
		respWriter := NewCustomResponseWriter(tlsConn)
		shouldBreak := false
		if newReq == nil {
			logger.Error("newReq is nil")
			respWriter.WriteHeader(http.StatusBadRequest)
			respWriter.Write([]byte("Invalid TLS request"))
			respWriter.Flush()
			return
		}
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == "CONNECT" {
				logger.Error("newReq is nil")
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte("Invalid TLS request"))
				return
			}
			if r.Proto == "HTTP/2.0" {
				logger.Info("new http2 request coming, shouldn;t happend at here")
				shouldBreak = true
			} else {
				r.Host = req.Host
				r.URL.Host = req.Host
				r.URL.Scheme = "https"
				forward.HandleHttp1(respWriter, r, true)
			}
		})

		handler.ServeHTTP(respWriter, newReq)
		respWriter.Flush()

		// 检查是否需要结束循环（例如连接关闭）
		if newReq.Close || respWriter.status == http.StatusServiceUnavailable || shouldBreak {
			break
		}

	}
}

func HandleHttpConnection(conn net.Conn, req *http.Request) {
	log.Printf("Received non-CONNECT request: %s %s", req.Method, req.URL.String())
	clientReader := forward.NewRequestReader(false, conn)
	for !clientReader.IsEOF() {
		newReq, err := clientReader.ReadRequest()
		if err != nil {
			logger.Error(err.Error())
			return
		}
		respWriter := NewCustomResponseWriter(conn)
		shouldBreak := false
		if newReq == nil {
			logger.Error("newReq is nil")
			respWriter.WriteHeader(http.StatusBadRequest)
			respWriter.Flush()
			_ = conn.Close()
			return
		}
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == "CONNECT" {
				logger.Error("newReq is nil")
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			if r.Proto == "HTTP/2.0" {
				logger.Info("new http2 request coming, shouldn;t happend at here")
				shouldBreak = true
			} else {
				r.Host = req.Host
				r.URL.Host = req.Host
				r.URL.Scheme = "http"
				// 主要这里的clientTsl已经不能读取body了，所以需要使用一个reader来读取body
				forward.HandleHttp1(respWriter, r, false)
			}
		})

		handler.ServeHTTP(respWriter, newReq)
		respWriter.Flush()

		// 检查是否需要结束循环（例如连接关闭）
		if newReq.Close || respWriter.status == http.StatusServiceUnavailable || shouldBreak {
			break
		}

	}

}
