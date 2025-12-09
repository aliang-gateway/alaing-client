package http

import (
	"fmt"
	"log"
	"net/http"

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

		err := http.ListenAndServe(port, wrappedMux)
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
}
