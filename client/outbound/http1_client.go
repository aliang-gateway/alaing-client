package outbound

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
	"sync"

	"golang.org/x/net/http2"
	"nursor.org/nursorgate/client/outbound/cert"
	"nursor.org/nursorgate/common/logger"
)

var token string

func SetOutboundToken(t string) {
	token = t
}

func GetOutboundToken() string {
	return token
}

type OutboundClient struct {
	conn *tls.Conn
	Tr   *http2.Transport
}

func (c *OutboundClient) Write(b []byte) (int, error) {
	return c.conn.Write(b)
}

func (c *OutboundClient) Read(b []byte) (int, error) {
	return c.conn.Read(b)
}

func (c *OutboundClient) Close() error {
	return c.conn.Close()
}

func (c *OutboundClient) LocalAddr() net.Addr {
	return c.conn.LocalAddr()
}

func (c *OutboundClient) PreHttp2AuthCheck() error {
	// 构造身份验证数据
	tokenData := map[string]string{"token": token}
	jsonData, err := json.Marshal(tokenData)
	if err != nil {
		return err
	}
	magic := []byte("MAGIC:custom")
	length := uint32(len(jsonData))
	lengthBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(lengthBytes, length)
	payload := append(magic, lengthBytes...)
	payload = append(payload, jsonData...)

	// 构造 HTTP 请求
	authRequest := []byte("POST /auth HTTP/1.1\r\n" +
		"Host: ai-gateway.nursor.org\r\n" +
		"Content-Length: " + strconv.Itoa(len(payload)) + "\r\n" +
		"\r\n")
	authRequest = append(authRequest, payload...)

	// 通过现有 Conn 发送身份验证请求
	_, err = c.conn.Write(authRequest)
	if err != nil {
		return err
	}

	// 读取响应
	var responseBytes []byte
	buf := make([]byte, 1024)
	for {
		n, err := c.conn.Read(buf)
		if err != nil && err != io.EOF {
			return err
		}
		responseBytes = append(responseBytes, buf[:n]...)

		// 检查是否读取到完整响应
		if bytes.Contains(responseBytes, []byte("\r\n\r\n")) {
			// 找到头部结束位置
			headerEnd := bytes.Index(responseBytes, []byte("\r\n\r\n")) + 4
			if headerEnd < 4 {
				continue // 未找到完整头部，继续读取
			}

			// 解析头部，获取 Content-Length
			resp, err := http.ReadResponse(bufio.NewReader(bytes.NewReader(responseBytes[:headerEnd])), nil)
			if err != nil {
				return err
			}
			contentLength := resp.ContentLength
			if contentLength == -1 {
				// 如果没有 Content-Length，可能需要处理 chunked encoding（此处假设固定响应）
				break
			}

			// 检查是否已读取完整 body
			totalLength := int64(headerEnd) + contentLength
			if int64(len(responseBytes)) >= totalLength {
				break
			}
		}
		if errors.Is(err, io.EOF) && err != nil {
			return nil
		}
	}

	// 简单解析 HTTP 响应（假设响应是标准 HTTP 格式）
	respReader := bytes.NewReader(responseBytes)
	resp, err := http.ReadResponse(bufio.NewReader(respReader), nil)
	if err != nil {
		return err
	}
	// defer resp.Body.Close()

	authBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if string(authBody) != "success" {
		return errors.New(string(authBody))
	}

	// 返回经过认证的 Conn
	return nil
}

func NewOutboundClient(url string, SNIName string) (*OutboundClient, error) {
	conn, err := net.Dial("tcp", url)
	if err != nil {
		return nil, err
	}
	myCert := cert.GetOutboundCert(false, SNIName)
	tlsConfig := myCert.GetTLSConfig()
	tlsConn := tls.Client(conn, tlsConfig)
	err = tlsConn.Handshake()
	if err != nil {
		return nil, err
	}
	c := &OutboundClient{conn: tlsConn}
	//err = c.PreHttp2AuthCheck()
	if err != nil {
		return nil, err
	}

	// c.SetToken(myCert.GetToken())
	return c, nil
}

func (c *OutboundClient) Forward(localConn *tls.Conn) error {
	var wg sync.WaitGroup
	wg.Add(2)
	wrapConn := &WrappedTLSConn{
		Conn: localConn,
		Buf:  []byte(http2.ClientPreface),
	}
	go func() {
		n, err := io.Copy(c.conn, wrapConn)
		if err != nil {
			logger.Error(err.Error())
		}
		logger.Debug(fmt.Sprintf("forwarded send %d bytes", n))
		wg.Done()
	}()
	go func() {
		n, err := io.Copy(localConn, c.conn)
		if err != nil {
			logger.Error(err.Error())
		}
		logger.Debug(fmt.Sprintf("forwarded return %d bytes", n))
		wg.Done()
	}()
	wg.Wait()

	return nil
	// 配置 HTTP/2 服务器端
	// srv := &http2.Server{}
	// // 配置 HTTP/2 客户端端
	// tr := &http2.Transport{}
	// wrappedConn := &WrappedTLSConn{
	// 	Conn: c.Conn,
	// 	Buf:  []byte(http2.ClientPreface),
	// }
	// // 将客户端连接交给 http2.Server 处理
	// go func() {
	// 	srv.ServeConn(wrappedConn, &http2.ServeConnOpts{
	// 		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	// 			// 使用 http2.Transport 向目标服务器转发请求
	// 			resp, err := tr.RoundTrip(r)
	// 			if err != nil {
	// 				logger.Error(fmt.Sprintf("Failed to forward request: %v", err))
	// 				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	// 				return
	// 			}
	// 			defer resp.Body.Close()

	// 			// 复制响应头
	// 			for k, vv := range resp.Header {
	// 				for _, v := range vv {
	// 					w.Header().Add(k, v)
	// 				}
	// 			}
	// 			w.WriteHeader(resp.StatusCode)
	// 			io.Copy(w, resp.Body)
	// 		}),
	// 	})
	// }()
	// return nil
}
