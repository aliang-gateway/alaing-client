package http

import (
	"fmt"
	"log"
	"net/http"

	"nursor.org/nursorgate/app/http/handlers"
	server "nursor.org/nursorgate/app/http/websocket"
	"nursor.org/nursorgate/common/logger"
)

// StartHttpServer 启动HTTP服务器，注册所有路由
func StartHttpServer() {
	// 定义 HTTP 服务端口
	port := "127.0.0.1:56431"

	// 注册所有路由
	registerAllRoutes()

	// 启动 HTTP 服务（非阻塞）
	go func() {
		logger.Info(fmt.Sprintf("Starting HTTP server on %s...\n", port))
		err := http.ListenAndServe(port, nil)
		if err != nil {
			log.Fatalf("HTTP server failed: %v", err)
		}
	}()

	// 启动WebSocket服务器
	server.StartWebSocketServer()

	// 保持主线程运行
	select {}
}

// registerAllRoutes 注册所有HTTP路由
func registerAllRoutes() {
	// Token相关路由
	handlers.RegisterTokenRoutes()

	// 运行控制相关路由
	handlers.RegisterRunRoutes()

	// 当前代理管理相关路由
	handlers.RegisterProxyRoutes()

	// 代理注册表相关路由
	handlers.RegisterProxyRegistryRoutes()

	// 配置相关路由
	handlers.RegisterConfigRoutes()
}
