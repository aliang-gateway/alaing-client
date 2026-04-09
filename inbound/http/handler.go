package http

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/netip"
	"strconv"
	"strings"
	"time"

	"aliang.one/nursorgate/common/logger"
	M "aliang.one/nursorgate/inbound/tun/metadata"
	"aliang.one/nursorgate/processor/tcp"
)

func HandleHttpConnection(conn net.Conn, reader *bufio.Reader, req *http.Request) {
	defer conn.Close()

	log.Printf("Received non-CONNECT request: %s %s", req.Method, req.URL.String())

	// Extract metadata from HTTP request for transparent proxy handling
	metadata, err := extractMetadataFromHTTP(req, conn)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to extract metadata from HTTP request: %v", err))
		respWriter := NewCustomResponseWriter(conn)
		respWriter.WriteHeader(http.StatusBadGateway)
		respWriter.Write([]byte(fmt.Sprintf("Failed to process request: %v", err)))
		respWriter.Flush()
		return
	}

	logger.Debug(fmt.Sprintf("HTTP transparent proxy: hostname=%s, port=%d, srcIP=%s, dstIP=%s",
		metadata.HostName, metadata.DstPort, metadata.SrcIP, metadata.DstIP))

	// Import tcp handler for unified TCP processing
	tcpHandler := tcp.GetHandler()

	replayConn, err := newHTTPRequestReplayConn(conn, reader, req)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to rebuild HTTP request stream: %v", err))
		respWriter := NewCustomResponseWriter(conn)
		respWriter.WriteHeader(http.StatusBadGateway)
		respWriter.Write([]byte(fmt.Sprintf("Failed to rebuild request stream: %v", err)))
		respWriter.Flush()
		return
	}

	// Create context for the handler
	ctx := context.Background()

	// Delegate to processor/tcp for routing and relay
	// The handler will:
	// 1. Detect protocol (TLS on 443, HTTP on 80, direct for others)
	// 2. Route based on domain rules (cursor proxy, door proxy, or direct)
	// 3. Handle bidirectional relay with statistics
	logger.Debug(fmt.Sprintf("Routing HTTP request through TCP handler for %s:%d", metadata.HostName, metadata.DstPort))
	if err := tcpHandler.Handle(ctx, replayConn, metadata); err != nil {
		logger.Error(fmt.Sprintf("TCP handler failed for %s: %v", metadata.HostName, err))
		return
	}

	logger.Debug(fmt.Sprintf("HTTP connection closed: %s:%d", metadata.HostName, metadata.DstPort))
}

// extractMetadataFromHTTP extracts connection metadata from HTTP request
func extractMetadataFromHTTP(req *http.Request, conn net.Conn) (*M.Metadata, error) {
	metadata := &M.Metadata{
		Network: M.TCP,
	}

	// Extract host and port from HTTP request
	host := req.Header.Get("Host")
	if host == "" {
		host = req.URL.Host
	}

	if host == "" {
		return nil, fmt.Errorf("cannot determine host from HTTP request")
	}

	// Parse host:port
	var hostOnly string
	var port uint16 = 80 // Default HTTP port

	if strings.Contains(host, ":") {
		parts := strings.Split(host, ":")
		hostOnly = parts[0]
		portNum, err := strconv.ParseUint(parts[1], 10, 16)
		if err == nil {
			port = uint16(portNum)
		}
	} else {
		hostOnly = host
	}

	// Set hostname with HTTP binding source
	if hostOnly != "" {
		metadata.SetHostName(hostOnly, M.BindingSourceHTTP, 10*time.Minute)
	}
	metadata.DstPort = port

	// Try to parse host as IP
	if ip := net.ParseIP(hostOnly); ip != nil {
		if addr, err := convertNetIPToNetipAddr(ip); err == nil {
			metadata.DstIP = addr
		}
	} else {
		// Try to resolve hostname
		ips, err := net.LookupIP(hostOnly)
		if err != nil {
			logger.Debug(fmt.Sprintf("DNS resolution failed for %s: %v (will retry at tunnel time)", hostOnly, err))
		} else if len(ips) > 0 {
			if addr, err := convertNetIPToNetipAddr(ips[0]); err == nil {
				metadata.DstIP = addr
				logger.Debug(fmt.Sprintf("Resolved %s to %s", hostOnly, metadata.DstIP.String()))
			}
		}
	}

	// Extract source
	if remoteAddr, ok := conn.RemoteAddr().(*net.TCPAddr); ok {
		if addr, err := convertNetIPToNetipAddr(remoteAddr.IP); err == nil {
			metadata.SrcIP = addr
		}
		metadata.SrcPort = uint16(remoteAddr.Port)
	}

	// Extract local address
	if localAddr, ok := conn.LocalAddr().(*net.TCPAddr); ok {
		if addr, err := convertNetIPToNetipAddr(localAddr.IP); err == nil {
			metadata.MidIP = addr
		}
		metadata.MidPort = uint16(localAddr.Port)
	}

	return metadata, nil
}

// convertNetIPToNetipAddr converts net.IP to netip.Addr
func convertNetIPToNetipAddr(ip net.IP) (netip.Addr, error) {
	if ip == nil {
		return netip.Addr{}, fmt.Errorf("nil IP")
	}
	return netip.ParseAddr(ip.String())
}

type replayConn struct {
	net.Conn
	reader io.Reader
}

func (c *replayConn) Read(p []byte) (int, error) {
	return c.reader.Read(p)
}

func (c *replayConn) CloseRead() error {
	if closer, ok := c.Conn.(interface{ CloseRead() error }); ok {
		return closer.CloseRead()
	}
	return nil
}

func (c *replayConn) CloseWrite() error {
	if closer, ok := c.Conn.(interface{ CloseWrite() error }); ok {
		return closer.CloseWrite()
	}
	return nil
}

func newHTTPRequestReplayConn(conn net.Conn, reader *bufio.Reader, req *http.Request) (net.Conn, error) {
	if conn == nil {
		return nil, errors.New("conn is nil")
	}
	if reader == nil {
		return nil, errors.New("reader is nil")
	}
	head, err := serializeHTTPRequestHead(req)
	if err != nil {
		return nil, err
	}

	return &replayConn{
		Conn:   conn,
		reader: io.MultiReader(bytes.NewReader(head), reader),
	}, nil
}

func serializeHTTPRequestHead(req *http.Request) ([]byte, error) {
	if req == nil {
		return nil, errors.New("request is nil")
	}

	target := strings.TrimSpace(req.RequestURI)
	if target == "" && req.URL != nil {
		target = req.URL.String()
	}
	if target == "" {
		return nil, errors.New("request target is empty")
	}

	proto := strings.TrimSpace(req.Proto)
	if proto == "" {
		proto = "HTTP/1.1"
	}

	var buf bytes.Buffer
	if _, err := fmt.Fprintf(&buf, "%s %s %s\r\n", req.Method, target, proto); err != nil {
		return nil, err
	}

	hostWritten := false
	if req.Host != "" {
		if _, err := fmt.Fprintf(&buf, "Host: %s\r\n", req.Host); err != nil {
			return nil, err
		}
		hostWritten = true
	}

	contentLengthWritten := false
	transferEncodingWritten := false
	for key, values := range req.Header {
		if strings.EqualFold(key, "Host") {
			if !hostWritten && len(values) > 0 {
				if _, err := fmt.Fprintf(&buf, "Host: %s\r\n", values[0]); err != nil {
					return nil, err
				}
				hostWritten = true
			}
			continue
		}
		if strings.EqualFold(key, "Content-Length") {
			contentLengthWritten = true
		}
		if strings.EqualFold(key, "Transfer-Encoding") {
			transferEncodingWritten = true
		}
		for _, value := range values {
			if _, err := fmt.Fprintf(&buf, "%s: %s\r\n", key, value); err != nil {
				return nil, err
			}
		}
	}

	if !contentLengthWritten && req.ContentLength > 0 {
		if _, err := fmt.Fprintf(&buf, "Content-Length: %d\r\n", req.ContentLength); err != nil {
			return nil, err
		}
	}
	if !transferEncodingWritten && len(req.TransferEncoding) > 0 {
		if _, err := fmt.Fprintf(&buf, "Transfer-Encoding: %s\r\n", strings.Join(req.TransferEncoding, ", ")); err != nil {
			return nil, err
		}
	}

	if _, err := buf.WriteString("\r\n"); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}
