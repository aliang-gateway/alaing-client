package out

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	"golang.org/x/net/http2"
	"nursor.org/nursorgate/common/logger"
	"nursor.org/nursorgate/processor/cert/server"
)

type OutboundClient2 struct {
	Conn     *tls.Conn
	Tr       *http2.Transport
	streamID uint32
}

func (c *OutboundClient2) Write(b []byte) (int, error) {
	return c.Conn.Write(b)
}

func (c *OutboundClient2) Read(b []byte) (int, error) {
	return c.Conn.Read(b)
}

func (c *OutboundClient2) Close() error {
	return c.Conn.Close()
}

func (c *OutboundClient2) LocalAddr() net.Addr {
	return c.Conn.LocalAddr()
}

func NewHttp2ProxyClient(proxyAddr string, reqTarget string, isHttp2 bool) (*OutboundClient2, error) {
	conn, err := net.Dial("tcp", proxyAddr)
	SetKeepAlive(conn)
	if err != nil {
		return nil, err
	}
	myCert := server.GetOutboundCert(isHttp2, reqTarget)
	tlsConfig := myCert.GetTLSConfig()
	if isHttp2 {
		tlsConfig.NextProtos = []string{"h2", "http/1.1"}
	} else {
		tlsConfig.NextProtos = []string{"http/1.1"}
	}

	tlsConn := tls.Client(conn, tlsConfig)
	err = tlsConn.Handshake()
	if err != nil {
		return nil, err
	}

	return &OutboundClient2{Conn: tlsConn}, nil
}

func (c *OutboundClient2) Forward(localConn *tls.Conn) error {
	var wg sync.WaitGroup
	wg.Add(2)
	_, err := c.Conn.Write([]byte(http2.ClientPreface))
	if err != nil {
		logger.Error(err)
		return err
	}

	go func() {
		n, err := io.Copy(c.Conn, localConn)
		if err != nil {
			logger.Error(err.Error())
		}
		logger.Debug(fmt.Sprintf("forwarded send %d bytes", n))
		err = c.Conn.CloseWrite()
		if err != nil {
			logger.Error(err)
		}
		wg.Done()
	}()
	go func() {
		n, err := io.Copy(localConn, c.Conn)
		if err != nil {
			logger.Error(err.Error())
		}
		logger.Debug(fmt.Sprintf("forwarded return %d bytes", n))
		err = localConn.CloseWrite()
		if err != nil {
			logger.Error(err)
		}
		wg.Done()
	}()
	wg.Wait()
	return nil
}

func (c *OutboundClient2) ForwardSimple(localConn net.Conn, reqHost string) error {
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		n, err := io.Copy(c.Conn, localConn)
		if err != nil {
			logger.Error(err.Error())
		}
		logger.Debug(fmt.Sprintf("forwarded send %d bytes for host: %s", n, reqHost))
		err = c.Conn.CloseWrite()
		if err != nil {
			logger.Error(err)
		}
		wg.Done()
	}()
	go func() {
		n, err := io.Copy(localConn, c.Conn)
		if err != nil {
			logger.Error(err.Error())
		}
		logger.Debug(fmt.Sprintf("forwarded return %d bytes from host: %s", n, reqHost))
		// 关闭写入方向
		localConnWriter, ok := localConn.(interface{ CloseWrite() error })
		if ok {
			err = localConnWriter.CloseWrite()
		} else {
			err = localConn.Close()
		}
		if err != nil {
			logger.Error(err)
		}
		// 关闭连接
		// err = localConn.CloseWrite()
		// if err != nil {
		// 	logger.Error(err)
		// }
		wg.Done()
	}()
	wg.Wait()
	return nil
}

func (c *OutboundClient2) ForwardHttp2(clientConn net.Conn) error {
	clientFramer := http2.NewFramer(c.Conn, clientConn)
	proxyFramer := http2.NewFramer(clientConn, c.Conn)

	// 简化起见：直接拷贝所有帧
	go func() {
		for {
			frame, err := clientFramer.ReadFrame()
			if err != nil {
				break
			}
			// 你可以在这里做修改，比如拦截 header 添加 token
			err = proxyFramer.WriteRawFrame(frame.Header().Type, frame.Header().Flags, frame.Header().StreamID, frame.(*http2.DataFrame).Data())
			if err != nil {
				break
			}
		}
	}()

	go func() {
		for {
			frame, err := proxyFramer.ReadFrame()
			if err != nil {
				break
			}
			// 你可以在这里做修改，比如拦截 header 添加 token
			err = clientFramer.WriteRawFrame(frame.Header().Type, frame.Header().Flags, frame.Header().StreamID, frame.(*http2.DataFrame).Data())
			if err != nil {
				break
			}
		}
	}()

	// 可选：从 mitmproxy 方向也读回来
	go func() {
		io.Copy(clientConn, c.Conn)
	}()

	return nil

}

// 创建直接连接到目标服务器的HTTP/2客户端
func NewDirectHttp2Client(targetHost string) (*OutboundClient2, error) {
	dialer := &net.Dialer{
		Timeout: 10 * time.Second,
		Resolver: &net.Resolver{
			PreferGo: true,
		},
	}
	host, _, err := net.SplitHostPort(targetHost)
	if err != nil {
		return nil, err
	}
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(server.CaCert)
	// 配置TLS客户端
	tlsConfig := &tls.Config{
		ServerName:         host,
		InsecureSkipVerify: true,                       // 生产环境中应设置为false并处理证书
		NextProtos:         []string{"h2", "http/1.1"}, // 指定HTTP/2协议
	}

	// 建立TCP连接
	tcpConn, err := dialer.Dial("tcp", targetHost)
	if err != nil {
		return nil, fmt.Errorf("failed to dial TCP: %v", err)
	}

	// 建立TLS连接
	tlsConn := tls.Client(tcpConn, tlsConfig)

	// 验证TLS握手
	if err := tlsConn.Handshake(); err != nil {
		tcpConn.Close()
		return nil, fmt.Errorf("OutboundClient2 TLS handshake failed: %v", err)
	}

	return &OutboundClient2{Conn: tlsConn}, nil
}

// 处理HTTP/2请求的转发（直接访问版本）
func (c *OutboundClient2) ForwardDirect(localConn *tls.Conn) error {
	var wg sync.WaitGroup
	wg.Add(2)

	_, err := c.Conn.Write([]byte(http2.ClientPreface))
	if err != nil {
		logger.Error(err)
		return err
	}

	// 从客户端读取数据并发送到目标服务器
	go func() {
		defer wg.Done()
		n, err := io.Copy(c.Conn, localConn)
		if err != nil {
			logger.Error(fmt.Sprintf("failed to forward to target: %v", err))
			return
		}
		logger.Info(fmt.Sprintf("forwarded %d bytes to target", n))

		// 关闭写入方向
		err = c.Conn.CloseWrite()
		if err != nil {
			logger.Error(err)
		}
	}()

	// 从目标服务器读取响应并返回给客户端
	go func() {
		defer wg.Done()
		n, err := io.Copy(localConn, c.Conn)
		if err != nil {
			logger.Error(fmt.Sprintf("failed to forward to client: %v", err))
			//errChan <- err
			return
		}
		logger.Info(fmt.Sprintf("forwarded %d bytes to client", n))

		// 关闭写入方向
		err = localConn.CloseWrite()
		if err != nil {
			logger.Error(err)
		}

	}()

	wg.Wait()
	_ = c.Conn.Close()
	_ = localConn.Close()
	return nil
}
