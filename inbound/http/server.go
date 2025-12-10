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
	"sync/atomic"
	"time"

	"nursor.org/nursorgate/common/logger"
)

// HTTP server state management
var (
	httpListener  net.Listener
	httpCtx       context.Context
	httpCancel    context.CancelFunc
	httpMutex     sync.Mutex
	isHttpRunning bool
	connIDCounter int64
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
		// Check if we should stop before accepting new connections
		select {
		case <-httpCtx.Done():
			logger.Info("HTTP proxy server shutting down")
			httpMutex.Lock()
			isHttpRunning = false
			httpMutex.Unlock()
			// Close listener to ensure cleanup
			if httpListener != nil {
				httpListener.Close()
			}
			return
		default:
		}

		// Set a read deadline to allow periodic checks for shutdown
		listener.(*net.TCPListener).SetDeadline(getDeadlineTime())

		conn, err := listener.Accept()
		if err != nil {
			// Check if context was cancelled during Accept
			select {
			case <-httpCtx.Done():
				logger.Info("HTTP proxy server shutting down (during accept)")
				httpMutex.Lock()
				isHttpRunning = false
				httpMutex.Unlock()
				return
			default:
			}

			// Check for timeout error (expected, continue loop)
			if opErr, ok := err.(*net.OpError); ok && opErr.Timeout() {
				// Timeout is expected, just continue to check for shutdown signal
				continue
			}

			// Check if listener was closed (shutdown signal)
			if opErr, ok := err.(*net.OpError); ok {
				if opErr.Err != nil {
					errStr := opErr.Err.Error()
					if errStr == "use of closed network connection" ||
						strings.Contains(errStr, "closed network connection") {
						logger.Info("HTTP proxy server listener closed")
						httpMutex.Lock()
						isHttpRunning = false
						httpMutex.Unlock()
						return
					}
				}
			}

			// For other errors, log and continue
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

	if !isHttpRunning {
		httpMutex.Unlock()
		logger.Warn("HTTP proxy is not running")
		return
	}

	logger.Info("Stopping HTTP proxy server...")

	// 先取消 context，通知 goroutine 停止
	if httpCancel != nil {
		httpCancel()
	}

	// 然后关闭 listener，这会中断 Accept() 调用
	listener := httpListener
	httpMutex.Unlock()

	// 在锁外关闭 listener，避免死锁
	if listener != nil {
		if err := listener.Close(); err != nil {
			logger.Debug(fmt.Sprintf("Error closing listener: %v", err))
		}
	}

	// 等待一小段时间，确保 goroutine 退出
	time.Sleep(100 * time.Millisecond)

	httpMutex.Lock()
	isHttpRunning = false
	httpMutex.Unlock()

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

	// 为每个连接分配唯一ID
	connID := atomic.AddInt64(&connIDCounter, 1)
	logger.Info(fmt.Sprintf("[CONN#%d] 新连接建立 - %s → 127.0.0.1:56432", connID, conn.RemoteAddr()))

	// 读取客户端初始数据，检查是否为 CONNECT 请求
	reader := bufio.NewReader(conn)
	req, err := http.ReadRequest(reader)
	if err != nil {
		logger.Error(fmt.Sprintf("[CONN#%d] 读取请求失败: %v", connID, err))
		return
	}

	if req.Method == "CONNECT" {
		logger.Info(fmt.Sprintf("[CONN#%d] CONNECT请求: %s", connID, req.Host))

		// Extract metadata from CONNECT request
		metadata, err := ExtractMetadataFromCONNECT(req, conn)
		if err != nil {
			logger.Error(fmt.Sprintf("[CONN#%d] 提取元数据失败: %v", connID, err))
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

		logger.Debug(fmt.Sprintf("[CONN#%d] 目标信息: %s:%d (IP:%s)", connID, metadata.HostName, metadata.DstPort, metadata.DstIP))

		// Send 200 Connection Established
		// IMPORTANT: Use raw write instead of http.Response for proper formatting
		response200 := "HTTP/1.1 200 Connection Established\r\n\r\n"
		if _, err := conn.Write([]byte(response200)); err != nil {
			logger.Error(fmt.Sprintf("[CONN#%d] 发送200应答失败: %v", connID, err))
			return
		}
		logger.Info(fmt.Sprintf("[CONN#%d] ✓ 已发送200 Connection Established", connID))

		// Handle CONNECT tunnel - establishes direct tunnel between client and target
		logger.Debug(fmt.Sprintf("[CONN#%d] 开始建立隧道...", connID))
		if err := HandleRawConnect(conn, metadata); err != nil {
			logger.Error(fmt.Sprintf("[CONN#%d] ❌ 隧道建立失败: %v", connID, err))
		} else {
			logger.Info(fmt.Sprintf("[CONN#%d] ✓ 隧道建立成功并已关闭", connID))
		}
	} else {
		// 透明代理，基本不存在
		HandleHttpConnection(conn, req)
	}
}
