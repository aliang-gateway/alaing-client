package http

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net"
	"net/http"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"nursor.org/nursorgate/app"
	"nursor.org/nursorgate/app/http/middleware"
	"nursor.org/nursorgate/app/http/routes"
	"nursor.org/nursorgate/common/logger"
)

var (
	// mux is the custom request multiplexer for applying middleware
	mux *http.ServeMux

	// server is the HTTP server instance
	server *http.Server

	// serverMutex protects server state
	serverMutex sync.Mutex

	// isRunning indicates if server is currently running
	isRunning bool

	// actualPort stores the actual port the server is listening on
	actualPort string
)

// StartHttpServer 启动HTTP服务器，注册所有路由
func StartHttpServer() {
	serverMutex.Lock()
	defer serverMutex.Unlock()

	if isRunning {
		logger.Info("HTTP server is already running")
		return
	}

	// 定义 HTTP 服务端口
	port := "127.0.0.1:56431"

	// Initialize custom mux
	mux = http.NewServeMux()

	// 注册所有路由
	registerAllRoutes()

	// 启动 HTTP 服务（非阻塞）
	go func() {
		logger.Info(fmt.Sprintf("Starting HTTP server on %s...\n", port))

		// Wrap mux with middleware stack
		middlewares := middleware.GetDefaultMiddleware()
		wrappedMux := middleware.Chain(mux, middlewares...)

		// 尝试监听端口，如果被占用则尝试其他端口
		listener, err := net.Listen("tcp", port)
		if err != nil {
			if strings.Contains(err.Error(), "address already in use") {
				// 尝试自动选择可用端口
				logger.Warn(fmt.Sprintf("Port %s is already in use, trying to find an available port...", port))
				listener, err = net.Listen("tcp", "127.0.0.1:0") // 0 means auto-select port
				if err != nil {
					log.Fatalf("HTTP server failed: unable to find available port: %v", err)
				}
				actualAddr := listener.Addr().(*net.TCPAddr)
				actualPort = fmt.Sprintf("%d", actualAddr.Port)
				logger.Info(fmt.Sprintf("HTTP server listening on alternative port: %s", actualAddr.String()))
			} else {
				log.Fatalf("HTTP server failed: %v", err)
			}
		} else {
			// Store the actual port from the default port
			_, portStr, _ := net.SplitHostPort(port)
			actualPort = portStr
		}

		// Create HTTP server
		server = &http.Server{
			Handler: wrappedMux,
		}

		isRunning = true

		err = server.Serve(listener)
		if err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP server failed: %v", err)
		}
	}()
}

// registerAllRoutes 注册所有HTTP路由
func registerAllRoutes() {
	// Create all handlers with dependency injection
	handlers := routes.NewHandlers()

	// Register all feature-grouped routes (using custom mux)
	// registerRoutesWithMux(handlers)
	routes.RegisterRoutes(handlers, mux)

	// NOTE: Rule engine initialization has been MOVED to cmd/start.go:InitializeGlobalRuleEngine()
	// This ensures the singleton rule engine is initialized only ONCE at startup
	// Previously this was duplicated in both HTTP mode and TUN mode
	logger.Info("HTTP: Rule engine has been initialized globally (see cmd/start.go)")

	// Initialize and start stats collector
	if handlers.TrafficStats != nil {
		logger.Info("Starting traffic stats collector...")
		// Note: statsCollector is created in routes.NewHandlers()
		// We need to access it through a package-level function
		routes.StartStatsCollector(handlers)
	}

	// Register static file server for web dashboard
	registerStaticFiles()
}

// initializeRuleEngine has been REMOVED - replaced by cmd/start.go:InitializeGlobalRuleEngine()
// This function was causing duplicate initialization of the singleton rule engine
// See: cmd/start.go for the new centralized initialization

// registerStaticFiles 注册静态文件服务（使用 embed 嵌入的文件）
func registerStaticFiles() {
	// 从嵌入的文件系统中获取 website 子目录
	// WebsiteFS 已经包含了 website 目录的内容
	websiteRoot, err := fs.Sub(app.WebsiteFS, "website")
	if err != nil {
		logger.Warn(fmt.Sprintf("Failed to access embedded website files: %v", err))
		return
	}

	// 注册 assets 路径处理器
	mux.HandleFunc("/assets/", func(w http.ResponseWriter, r *http.Request) {
		// 移除前导 /assets/
		filePath := strings.TrimPrefix(r.URL.Path, "/assets/")

		// 安全检查：确保没有目录遍历
		if strings.Contains(filePath, "..") {
			http.Error(w, "Invalid path", http.StatusBadRequest)
			return
		}

		// 尝试打开文件
		file, err := websiteRoot.Open("assets/" + filePath)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		defer file.Close()

		// 获取文件信息
		info, err := file.Stat()
		if err != nil {
			http.NotFound(w, r)
			return
		}

		// 设置 Content-Type
		setContentType(w, filePath)

		// 使用 ServeContent（支持 Range 请求和缓存）
		if rs, ok := file.(io.ReadSeeker); ok {
			http.ServeContent(w, r, filepath.Base(filePath), info.ModTime(), rs)
		} else {
			w.Header().Set("Content-Length", fmt.Sprintf("%d", info.Size()))
			io.Copy(w, file)
		}
	})

	// 注册根路径处理器，支持 SPA 路由
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// API路径返回404 JSON，防止返回HTML
		if strings.HasPrefix(r.URL.Path, "/api/") {
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(`{"code":404,"msg":"API endpoint not found","data":null}`))
			return
		}

		// 处理根路径
		if r.URL.Path == "/" {
			path := "/index.html"
			filePath := strings.TrimPrefix(path, "/")
			file, err := websiteRoot.Open(filePath)
			if err != nil {
				http.NotFound(w, r)
				return
			}
			defer file.Close()

			info, err := file.Stat()
			if err != nil {
				http.NotFound(w, r)
				return
			}

			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			if rs, ok := file.(io.ReadSeeker); ok {
				http.ServeContent(w, r, filepath.Base(path), info.ModTime(), rs)
			} else {
				w.Header().Set("Content-Length", fmt.Sprintf("%d", info.Size()))
				io.Copy(w, file)
			}
			return
		}

		// 对于其他路径，如果是非 assets 路径，返回 index.html（SPA 路由支持）
		indexFile, err := websiteRoot.Open("index.html")
		if err == nil {
			defer indexFile.Close()
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			io.Copy(w, indexFile)
		} else {
			http.NotFound(w, r)
		}
	})

	logger.Info("Static file server registered using embedded website files")
}

// setContentType 根据文件扩展名设置 Content-Type
// StopHttpServer gracefully stops the HTTP server
func StopHttpServer() error {
	serverMutex.Lock()
	defer serverMutex.Unlock()

	if !isRunning || server == nil {
		logger.Info("HTTP server is not running")
		return nil
	}

	logger.Info("Stopping HTTP server...")

	// Create shutdown context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Gracefully shutdown
	if err := server.Shutdown(ctx); err != nil {
		logger.Error(fmt.Sprintf("HTTP server shutdown error: %v", err))
		return err
	}

	isRunning = false
	server = nil
	logger.Info("HTTP server stopped successfully")
	return nil
}

// IsServerRunning returns whether the HTTP server is currently running
func IsServerRunning() bool {
	serverMutex.Lock()
	defer serverMutex.Unlock()
	return isRunning
}

// GetActualPort returns the actual port the HTTP server is listening on
func GetActualPort() string {
	serverMutex.Lock()
	defer serverMutex.Unlock()
	return actualPort
}

// setContentType 根据文件扩展名设置 Content-Type
func setContentType(w http.ResponseWriter, path string) {
	ext := filepath.Ext(path)
	switch ext {
	case ".html":
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
	case ".css":
		w.Header().Set("Content-Type", "text/css; charset=utf-8")
	case ".js":
		w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
	case ".json":
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
	case ".svg":
		w.Header().Set("Content-Type", "image/svg+xml")
	case ".png":
		w.Header().Set("Content-Type", "image/png")
	case ".jpg", ".jpeg":
		w.Header().Set("Content-Type", "image/jpeg")
	case ".ico":
		w.Header().Set("Content-Type", "image/x-icon")
	case ".woff", ".woff2":
		w.Header().Set("Content-Type", "fonts/woff2")
	case ".ttf":
		w.Header().Set("Content-Type", "fonts/ttf")
	case ".eot":
		w.Header().Set("Content-Type", "application/vnd.ms-fontobject")
	}
}
