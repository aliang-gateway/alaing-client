package server

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync"

	"github.com/gorilla/websocket"
	"nursor.org/nursorgate/common/logger"
	tun "nursor.org/nursorgate/inbound/tun/engine"
	"nursor.org/nursorgate/outbound"
	user "nursor.org/nursorgate/processor/auth"
	proxyConfig "nursor.org/nursorgate/processor/config"
	proxyRegistry "nursor.org/nursorgate/processor/proxy"
	"nursor.org/nursorgate/runner"
)

type LoginRequest struct {
	RefreshToken string `json:"refreshToken"`
	AccessToken  string `json:"accessToken"`
	SubId        string `json:"subId"`
}

var (
	wsUpgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			return true // Allow all origins for local development
		},
	}
	wsClients    = make(map[*websocket.Conn]bool)
	wsClientsMux sync.Mutex
)

func StartWebSocketServer() {
	wsPort := "127.0.0.1:56433"
	http.HandleFunc("/ws", handleWebSocket)

	go func() {
		logger.Debug(fmt.Sprintf("Starting WebSocket server on %s...\n", wsPort))
		err := http.ListenAndServe(wsPort, nil)
		if err != nil {
			log.Fatalf("WebSocket server failed: %v", err)
		}
	}()
}

func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := wsUpgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Failed to upgrade connection to WebSocket: %v", err)
		return
	}
	defer conn.Close()

	// Register new client
	wsClientsMux.Lock()
	wsClients[conn] = true
	wsClientsMux.Unlock()

	// Remove client when function returns
	defer func() {
		wsClientsMux.Lock()
		delete(wsClients, conn)
		wsClientsMux.Unlock()
	}()

	// Handle incoming messages and connection closure
	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}
	}
}

func StartHttpServer() {
	// 初始化代理注册中心（从配置加载）
	if err := proxyRegistry.GetRegistry().InitializeFromConfig(); err != nil {
		logger.Warn(fmt.Sprintf("Failed to initialize proxy registry: %v", err))
	} else {
		logger.Info("Proxy registry initialized successfully")
	}

	// 定义 HTTP 服务端口
	port := "127.0.0.1:56431"
	// 注册路由
	http.HandleFunc("/token/set", handleTokenSet)
	http.HandleFunc("/token/get", handleTokenGet)
	// run/stop
	http.HandleFunc("/run/start", handleRun)
	http.HandleFunc("/run/stop", handleStop)
	http.HandleFunc("/run/userInfo", handleRunUserInfo)
	// proxy config
	http.HandleFunc("/proxy/set", handleProxySet)
	http.HandleFunc("/proxy/get", handleProxyGet)
	// proxy registry
	http.HandleFunc("/proxy/registry/list", handleProxyRegistryList)
	http.HandleFunc("/proxy/registry/get", handleProxyRegistryGet)
	http.HandleFunc("/proxy/registry/register", handleProxyRegistryRegister)
	http.HandleFunc("/proxy/registry/unregister", handleProxyRegistryUnregister)
	http.HandleFunc("/proxy/registry/set-default", handleProxyRegistrySetDefault)
	http.HandleFunc("/proxy/registry/set-door", handleProxyRegistrySetDoor)
	http.HandleFunc("/proxy/registry/switch", handleProxyRegistrySwitch)
	// proxy registry
	http.HandleFunc("/proxy/registry/list", handleProxyRegistryList)
	http.HandleFunc("/proxy/registry/get", handleProxyRegistryGet)
	http.HandleFunc("/proxy/registry/register", handleProxyRegistryRegister)
	http.HandleFunc("/proxy/registry/unregister", handleProxyRegistryUnregister)
	http.HandleFunc("/proxy/registry/set-default", handleProxyRegistrySetDefault)
	http.HandleFunc("/proxy/registry/set-door", handleProxyRegistrySetDoor)
	http.HandleFunc("/proxy/registry/switch", handleProxyRegistrySwitch)

	// Start WebSocket server
	StartWebSocketServer()

	// 启动 HTTP 服务（非阻塞）
	go func() {
		logger.Info(fmt.Sprintf("Starting HTTP server on %s...\n", port))
		err := http.ListenAndServe(port, nil)
		if err != nil {
			log.Fatalf("HTTP server failed: %v", err)
		}
	}()

	// 写入端口到 ~/.cursor/nursor
	// err := writePortToFile(port)
	// if err != nil {
	// 	log.Printf("Failed to write port to file: %v", err)
	// }

	// 保持主线程运行
	select {}
}

// 通用的响应结构体
type Response struct {
	Code int         `json:"code"`
	Msg  string      `json:"msg"`
	Data interface{} `json:"data"`
}

// 写入端口到文件
func writePortToFile(port string) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	filePath := filepath.Join(homeDir, ".cursor", "nursor")
	err = os.MkdirAll(filepath.Dir(filePath), 0755)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(filePath, []byte(port[1:]), 0644) // 去掉冒号，只写 56431
}

func handleRun(w http.ResponseWriter, r *http.Request) {
	var req struct {
		// UserToken  string `json:"user_token"`
		InnerToken string `json:"inner_token"`
	}
	if err := decodeRequest(r, &req); err != nil {
		sendError(w, "Invalid request body", http.StatusBadRequest, nil)
		return
	}
	// user.SetUserToken(req.UserToken)
	user.SetInnerToken(req.InnerToken)
	go runner.Start()
	res := <-runner.RunStatusChan
	sendResponse(w, res)
}

func handleStop(w http.ResponseWriter, r *http.Request) {
	tun.Stop()
	sendResponse(w, map[string]string{"status": "success"})
}

func handleRunUserInfo(w http.ResponseWriter, r *http.Request) {
	var req struct {
		UserUUID   string `json:"user_uuid"`
		InnerToken string `json:"inner_token"`
		Username   string `json:"username"`
		Password   string `json:"password"`
	}
	if err := decodeRequest(r, &req); err != nil {
		sendError(w, "Invalid request body", http.StatusBadRequest, nil)
		return
	}
	logger.SetUserInfo(req.InnerToken)
	user.SetUsername(req.Username)
	user.SetPassword(req.Password)
	user.SetUserUUID(req.UserUUID)
	logger.Info("set user info tag")
	sendResponse(w, map[string]string{
		"status":  "success",
		"user_id": fmt.Sprintf("%d", user.GetUserId()),
	})
}

// 处理 /token/set
func handleTokenSet(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Token string `json:"token"`
	}
	if err := decodeRequest(r, &req); err != nil {
		sendError(w, "Invalid request body", http.StatusBadRequest, nil)
		return
	}
	outbound.SetOutboundToken(req.Token)
	sendResponse(w, map[string]string{"token": req.Token})
}

func handleTokenGet(w http.ResponseWriter, r *http.Request) {
	sendResponse(w, map[string]string{"token": outbound.GetOutboundToken()})
}

// handleProxySet 设置代理配置
func handleProxySet(w http.ResponseWriter, r *http.Request) {
	var cfg proxyConfig.ProxyConfig
	if err := decodeRequest(r, &cfg); err != nil {
		sendError(w, "Invalid request body", http.StatusBadRequest, nil)
		return
	}

	if err := proxyConfig.SetProxyConfig(&cfg); err != nil {
		sendError(w, err.Error(), http.StatusBadRequest, nil)
		return
	}

	sendResponse(w, map[string]string{"status": "success"})
}

// handleProxyGet 获取代理配置
func handleProxyGet(w http.ResponseWriter, r *http.Request) {
	vlessCfg := proxyConfig.GetVLESSConfig()
	ssCfg := proxyConfig.GetShadowsocksConfig()

	data := map[string]interface{}{
		"vless":       vlessCfg,
		"shadowsocks": ssCfg,
	}

	sendResponse(w, data)
}

// handleProxyRegistryList 列出所有已注册的代理
func handleProxyRegistryList(w http.ResponseWriter, r *http.Request) {
	info := proxyRegistry.GetRegistry().ListWithInfo()
	sendResponse(w, map[string]interface{}{
		"proxies": info,
		"count":   len(info),
	})
}

// handleProxyRegistryGet 获取指定代理
func handleProxyRegistryGet(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("name")
	if name == "" {
		sendError(w, "name parameter is required", http.StatusBadRequest, nil)
		return
	}

	info := proxyRegistry.GetRegistry().ListWithInfo()
	proxyInfo, exists := info[name]
	if !exists {
		sendError(w, "proxy info not found", http.StatusNotFound, nil)
		return
	}

	sendResponse(w, proxyInfo)
}

// handleProxyRegistryRegister 注册新代理
func handleProxyRegistryRegister(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name   string                   `json:"name"`
		Config *proxyConfig.ProxyConfig `json:"config"`
	}

	if err := decodeRequest(r, &req); err != nil {
		sendError(w, "Invalid request body", http.StatusBadRequest, nil)
		return
	}

	if req.Name == "" {
		sendError(w, "name is required", http.StatusBadRequest, nil)
		return
	}

	if req.Config == nil {
		sendError(w, "config is required", http.StatusBadRequest, nil)
		return
	}

	if err := proxyRegistry.GetRegistry().RegisterFromConfig(req.Name, req.Config); err != nil {
		sendError(w, err.Error(), http.StatusBadRequest, nil)
		return
	}

	sendResponse(w, map[string]string{"status": "success", "name": req.Name})
}

// handleProxyRegistryUnregister 注销代理
func handleProxyRegistryUnregister(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name string `json:"name"`
	}

	if err := decodeRequest(r, &req); err != nil {
		sendError(w, "Invalid request body", http.StatusBadRequest, nil)
		return
	}

	if err := proxyRegistry.GetRegistry().Unregister(req.Name); err != nil {
		sendError(w, err.Error(), http.StatusBadRequest, nil)
		return
	}

	sendResponse(w, map[string]string{"status": "success"})
}

// handleProxyRegistrySetDefault 设置默认代理
func handleProxyRegistrySetDefault(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name string `json:"name"`
	}

	if err := decodeRequest(r, &req); err != nil {
		sendError(w, "Invalid request body", http.StatusBadRequest, nil)
		return
	}

	if err := proxyRegistry.GetRegistry().SetDefault(req.Name); err != nil {
		sendError(w, err.Error(), http.StatusBadRequest, nil)
		return
	}

	sendResponse(w, map[string]string{"status": "success", "default": req.Name})
}

// handleProxyRegistrySetDoor 设置门代理
func handleProxyRegistrySetDoor(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name string `json:"name"`
	}

	if err := decodeRequest(r, &req); err != nil {
		sendError(w, "Invalid request body", http.StatusBadRequest, nil)
		return
	}

	if err := proxyRegistry.GetRegistry().SetDoor(req.Name); err != nil {
		sendError(w, err.Error(), http.StatusBadRequest, nil)
		return
	}

	sendResponse(w, map[string]string{"status": "success", "door": req.Name})
}

// handleProxyRegistrySwitch 切换代理（设置默认代理并更新 tunnel）
func handleProxyRegistrySwitch(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name string `json:"name"`
	}

	if err := decodeRequest(r, &req); err != nil {
		sendError(w, "Invalid request body", http.StatusBadRequest, nil)
		return
	}

	// 设置默认代理
	if err := proxyRegistry.GetRegistry().SetDefault(req.Name); err != nil {
		sendError(w, err.Error(), http.StatusBadRequest, nil)
		return
	}

	// 获取代理实例并更新 tunnel
	p, err := proxyRegistry.GetRegistry().Get(req.Name)
	if err != nil {
		sendError(w, err.Error(), http.StatusBadRequest, nil)
		return
	}

	// 更新 tunnel 的默认代理
	// 注意：这里需要导入 tunnel 包
	// tunnel.SetDefaultProxy(p)

	sendResponse(w, map[string]string{
		"status": "success",
		"name":   req.Name,
		"addr":   p.Addr(),
		"type":   p.Proto().String(),
	})
}

// 解析请求体
func decodeRequest(r *http.Request, v interface{}) error {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return err
	}
	return json.Unmarshal(body, v)
}

// 发送成功响应
func sendResponse(w http.ResponseWriter, data interface{}) {
	resp := Response{
		Code: 0,
		Msg:  "success",
		Data: data,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// 发送错误响应
func sendError(w http.ResponseWriter, msg string, statusCode int, data interface{}) {
	resp := Response{
		Code: statusCode,
		Msg:  msg,
		Data: data,
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(resp)
}
