package http

import (
	"fmt"
	"io"
	"io/fs"
	"log"
	"net"
	"net/http"
	"path/filepath"
	"strings"

	"nursor.org/nursorgate/app"
	"nursor.org/nursorgate/app/http/middleware"
	"nursor.org/nursorgate/app/http/routes"
	"nursor.org/nursorgate/common/logger"
)

var (
	// mux is the custom request multiplexer for applying middleware
	mux *http.ServeMux
)

// StartHttpServer 启动HTTP服务器，注册所有路由
func StartHttpServer() {
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
				logger.Info(fmt.Sprintf("HTTP server listening on alternative port: %s", actualAddr.String()))
			} else {
				log.Fatalf("HTTP server failed: %v", err)
			}
		}

		err = http.Serve(listener, wrappedMux)
		if err != nil {
			log.Fatalf("HTTP server failed: %v", err)
		}
	}()

	// 保持主线程运行
	select {}
}

// registerAllRoutes 注册所有HTTP路由
func registerAllRoutes() {
	// Create all handlers with dependency injection
	handlers := routes.NewHandlers()

	// Register all feature-grouped routes (using custom mux)
	// registerRoutesWithMux(handlers)
	routes.RegisterRoutes(handlers, mux)

	// Register static file server for web dashboard
	registerStaticFiles()
}

// registerStaticFiles 注册静态文件服务（使用 embed 嵌入的文件）
func registerStaticFiles() {
	// 从嵌入的文件系统中获取 website 子目录
	// WebsiteFS 已经包含了 website 目录的内容
	websiteRoot, err := fs.Sub(app.WebsiteFS, "website")
	if err != nil {
		logger.Warn(fmt.Sprintf("Failed to access embedded website files: %v", err))
		return
	}

	// 注册根路径处理器，支持 SPA 路由
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// 清理路径
		path := r.URL.Path
		if path == "/" {
			path = "/index.html"
		}

		// 移除前导斜杠以匹配 embed 文件系统
		filePath := strings.TrimPrefix(path, "/")

		// 尝试打开文件
		file, err := websiteRoot.Open(filePath)
		if err != nil {
			// 如果文件不存在，尝试返回 index.html（SPA 路由支持）
			if filePath != "index.html" {
				indexFile, err := websiteRoot.Open("index.html")
				if err == nil {
					defer indexFile.Close()
					w.Header().Set("Content-Type", "text/html; charset=utf-8")

					// 读取并写入文件内容
					io.Copy(w, indexFile)
					return
				}
			}
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
		setContentType(w, path)

		// 如果文件实现了 io.ReadSeeker，使用 ServeContent（支持 Range 请求）
		if rs, ok := file.(io.ReadSeeker); ok {
			http.ServeContent(w, r, filepath.Base(path), info.ModTime(), rs)
		} else {
			// 否则直接复制内容
			w.Header().Set("Content-Length", fmt.Sprintf("%d", info.Size()))
			io.Copy(w, file)
		}
	})

	logger.Info("Static file server registered using embedded website files")
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
		w.Header().Set("Content-Type", "font/woff2")
	case ".ttf":
		w.Header().Set("Content-Type", "font/ttf")
	case ".eot":
		w.Header().Set("Content-Type", "application/vnd.ms-fontobject")
	}
}
