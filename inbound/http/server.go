package http

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"nursor.org/nursorgate/common/logger"
)

// HTTP server state management
var (
	httpListener net.Listener
	httpCtx      context.Context
	httpCancel   context.CancelFunc
	httpMutex    sync.Mutex
	isHttpRunning bool
)

// StartHttpProxy starts the HTTP CONNECT proxy server
// This is a blocking function that runs until stopped
func StartMitmHttp() {
	httpMutex.Lock()
	if isHttpRunning {
		httpMutex.Unlock()
		logger.Warn("HTTP proxy is already running")
		return
	}

	// Create a cancellable context for shutdown
	httpCtx, httpCancel = context.WithCancel(context.Background())
	isHttpRunning = true
	httpMutex.Unlock()

	// Create listener
	listener, err := net.Listen("tcp", "127.0.0.1:56432")
	if err != nil {
		httpMutex.Lock()
		isHttpRunning = false
		httpMutex.Unlock()
		log.Fatalf("Failed to listen on 127.0.0.1:56432: %v", err)
	}

	httpMutex.Lock()
	httpListener = listener
	httpMutex.Unlock()

	logger.Info("HTTP CONNECT proxy server starting on 127.0.0.1:56432")

	// Accept connections in a loop
	for {
		// Check if we should stop
		select {
		case <-httpCtx.Done():
			logger.Info("HTTP proxy server shutting down")
			httpMutex.Lock()
			isHttpRunning = false
			httpMutex.Unlock()
			return
		default:
		}

		// Set a read deadline to allow periodic checks for shutdown
		listener.(*net.TCPListener).SetDeadline(getDeadlineTime())

		conn, err := listener.Accept()
		if err != nil {
			if opErr, ok := err.(*net.OpError); ok && opErr.Timeout() {
				// Timeout is expected, just continue to check for shutdown signal
				continue
			}
			logger.Debug(fmt.Sprintf("Accept error: %v", err))
			continue
		}

		go handleRawConnection(conn)
	}
}

// getDeadlineTime returns a deadline time for socket operations
func getDeadlineTime() time.Time {
	return time.Now().Add(1 * time.Second)
}

// StopHttpProxy stops the HTTP proxy server gracefully
func StopHttpProxy() {
	httpMutex.Lock()
	defer httpMutex.Unlock()

	if !isHttpRunning {
		logger.Warn("HTTP proxy is not running")
		return
	}

	logger.Info("Stopping HTTP proxy server...")

	if httpCancel != nil {
		httpCancel()
	}

	if httpListener != nil {
		httpListener.Close()
	}

	isHttpRunning = false
	logger.Info("HTTP proxy server stopped")
}

// IsHttpRunning returns the current state of HTTP proxy
func IsHttpRunning() bool {
	httpMutex.Lock()
	defer httpMutex.Unlock()
	return isHttpRunning
}

// 处理客户端连接
func handleRawConnection(conn net.Conn) {
	defer conn.Close()

	// 读取客户端初始数据，检查是否为 CONNECT 请求
	reader := bufio.NewReader(conn)
	req, err := http.ReadRequest(reader)
	if err != nil {
		log.Printf("Failed to read initial request: %v", err)
		return
	}

	if req.Method == "CONNECT" {
		logger.Debug("Received CONNECT request for " + req.Host)

		// Extract metadata from CONNECT request
		metadata, err := ExtractMetadataFromCONNECT(req, conn)
		if err != nil {
			logger.Error("Failed to extract metadata from CONNECT: " + err.Error())
			resp := &http.Response{
				Status:        "400 Bad Request",
				StatusCode:    http.StatusBadRequest,
				Proto:         "HTTP/1.1",
				ProtoMajor:    1,
				ProtoMinor:    1,
				Body:          io.NopCloser(strings.NewReader("")),
				ContentLength: 0,
			}
			resp.Write(conn)
			return
		}

		logger.Debug(fmt.Sprintf("CONNECT metadata: host=%s, port=%d, srcIP=%s, dstIP=%s",
			metadata.HostName, metadata.DstPort, metadata.SrcIP, metadata.DstIP))

		// Send 200 Connection Established
		// IMPORTANT: Use raw write instead of http.Response for proper formatting
		response200 := "HTTP/1.1 200 Connection Established\r\n\r\n"
		if _, err := conn.Write([]byte(response200)); err != nil {
			logger.Error("Failed to send 200 OK: " + err.Error())
			return
		}
		logger.Debug("Sent 200 Connection Established")

		// Handle CONNECT tunnel - establishes direct tunnel between client and target
		logger.Debug(fmt.Sprintf("Starting CONNECT tunnel for %s:%d", metadata.HostName, metadata.DstPort))
		if err := HandleCONNECTTunnel(conn, metadata); err != nil {
			logger.Error(fmt.Sprintf("CONNECT tunnel error for %s: %v", metadata.HostName, err))
		}
		logger.Debug("CONNECT tunnel closed for " + req.Host)
	} else {
		// 透明代理，基本不存在
		HandleHttpConnection(conn, req)
	}
}
