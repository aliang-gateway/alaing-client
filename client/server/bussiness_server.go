package server

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"sync"

	"github.com/getsentry/sentry-go"
	"github.com/gorilla/websocket"
	"nursor.org/nursorgate/client/install"
	"nursor.org/nursorgate/client/outbound"
	"nursor.org/nursorgate/client/server/tun"
	"nursor.org/nursorgate/client/user"
	"nursor.org/nursorgate/client/utils"
	"nursor.org/nursorgate/common/logger"
)

type LoginRequest struct {
	RefreshToken string `json:"refreshToken"`
	AccessToken  string `json:"accessToken"`
	SubId        string `json:"subId"`
}

var cacheQueue = make(chan LoginRequest, 100)

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

	// Create a done channel to signal when the connection should be closed
	done := make(chan struct{})

	// Start a goroutine to read from cacheQueue and send to client
	go func() {
		for {
			select {
			case loginReq := <-cacheQueue:
				err := conn.WriteJSON(loginReq)
				if err != nil {
					log.Printf("Failed to send message to client: %v", err)
					close(done)
					return
				}
			case <-done:
				return
			}
		}
	}()

	// Handle incoming messages and connection closure
	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			close(done)
			break
		}
	}
}

func StartHttpServer() {
	// 定义 HTTP 服务端口
	port := "127.0.0.1:56431"
	// 注册路由
	http.HandleFunc("/data/set", handleDataSet)
	http.HandleFunc("/data/get", handleDataGet)
	http.HandleFunc("/core/path", handleCoreExtensionPath)
	http.HandleFunc("/db/path", handleDbPath)
	http.HandleFunc("/data/delete", handleDataDelete)
	http.HandleFunc("/token/set", handleTokenSet)
	http.HandleFunc("/isJsModified", handleIsJsModified)
	http.HandleFunc("/token/get", handleTokenGet)
	http.HandleFunc("/status", handleStatus)
	http.HandleFunc("/data/login", handleShouldNewLogin)
	// run/stop
	http.HandleFunc("/run/start", handleRun)
	http.HandleFunc("/run/stop", handleStop)
	http.HandleFunc("/run/userInfo", handleRunUserInfo)

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
		UserToken string `json:"user_token"`
	}
	if err := decodeRequest(r, &req); err != nil {
		sendError(w, "Invalid request body", http.StatusBadRequest, nil)
		return
	}
	user.SetUserToken(req.UserToken)
	go tun.Start()
	res := <-tun.RunStatusChan
	sendResponse(w, res)
}

func handleStop(w http.ResponseWriter, r *http.Request) {
	tun.Stop()
	sendResponse(w, map[string]string{"status": "success"})
}

func handleRunUserInfo(w http.ResponseWriter, r *http.Request) {
	var req struct {
		UserToken string `json:"user_token"`
		UserId    string `json:"user_id"`
		Username  string `json:"username"`
		Password  string `json:"password"`
	}
	if err := decodeRequest(r, &req); err != nil {
		sendError(w, "Invalid request body", http.StatusBadRequest, nil)
		return
	}
	sentry.ConfigureScope(func(scope *sentry.Scope) {
		scope.SetTag("user_id", req.UserId)
	})
	user.SetUsername(req.Username)
	user.SetPassword(req.Password)
	print("set user info tag")
	sendResponse(w, map[string]string{
		"status":  "success",
		"user_id": fmt.Sprintf("%d", user.GetUserId()),
	})
}

// 处理 /data/delete
func handleDataDelete(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Key string `json:"key"`
	}
	if err := decodeRequest(r, &req); err != nil {
		sendError(w, "Invalid request body", http.StatusBadRequest, nil)
		return
	}
	utils.NewKVStore().Delete(req.Key)
	sendResponse(w, map[string]string{"key": req.Key})
}

// 处理 /data/set
func handleDataSet(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Key   string `json:"key"`
		Value any    `json:"value"`
	}
	if err := decodeRequest(r, &req); err != nil {
		sendError(w, "Invalid request body", http.StatusBadRequest, nil)
		return
	}
	utils.NewKVStore().Set(req.Key, req.Value)
	sendResponse(w, map[string]string{"key": req.Key})
}

// 处理 /data/get
func handleDataGet(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Key string `json:"key"`
	}
	if err := decodeRequest(r, &req); err != nil {
		sendError(w, "Invalid request body", http.StatusBadRequest, nil)
		return
	}
	value, err := utils.NewKVStore().Read(req.Key)

	if err != nil {
		// jsonNull, _ := json
		sendResponse(w, nil)
		return
	}
	sendResponse(w, value)
}

// 处理 /core/path
func handleCoreExtensionPath(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Path string `json:"path"`
	}
	if err := decodeRequest(r, &req); err != nil {
		sendError(w, "Invalid request body", http.StatusBadRequest, nil)
		return
	}
	extensionPath := req.Path
	if _, err := os.Stat(extensionPath); os.IsNotExist(err) {
		logger.Error("Extension path does not exist")
		sendError(w, "Extension path does not exist", http.StatusBadRequest, nil)
		return
	}
	// 设置扩展路径
	// 这里可以根据实际需要进行设置
	install.SetExtensionPath(extensionPath)
	// 备份原始的 core.js 文件
	var homeDir string
	if runtime.GOOS == "windows" {
		homeDir = os.Getenv("USERPROFILE")
	} else {
		homeDir = os.Getenv("HOME")
	}
	if err := install.BackCoreJSFile(filepath.Join(homeDir, ".nursor"), extensionPath); err != nil {
		logger.Error("Failed to install core.js file:", err)
		sendError(w, "Failed to install core.js file", http.StatusInternalServerError, nil)
		return
	}
	// 修改 core.js 文件
	if err := install.ModifyJSFile(filepath.Join(homeDir, ".nursor"), extensionPath); err != nil {
		logger.Error("Failed to install core.js file:", err)
		sendError(w, "Failed to install core.js file", http.StatusInternalServerError, nil)
		return
	}
	// 避免sentry
	workbenchPath := filepath.Join(filepath.Dir(extensionPath), "out", "vs", "workbench", "workbench.desktop.main.js")
	install.ReplaceSentryJs(workbenchPath)
	// 修改 login.js 文件
	if err := install.ReplaceLoginAncher(workbenchPath); err != nil {
		logger.Error("Failed to install login.js file:", err)
		sendError(w, "Failed to install login.js file", http.StatusInternalServerError, nil)
		return
	}
	sendResponse(w, map[string]string{"status": "success"})
}

func handleIsJsModified(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Path string `json:"path"`
	}
	if err := decodeRequest(r, &req); err != nil {
		sendError(w, "Invalid request body", http.StatusBadRequest, nil)
		return
	}
	sendResponse(w, map[string]string{"status": "success", "isModified": fmt.Sprintf("%t", install.IsJsModified(req.Path))})
}

// 处理 /db/path
func handleDbPath(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Path string `json:"path"`
	}
	if err := decodeRequest(r, &req); err != nil {
		sendError(w, "Invalid request body", http.StatusBadRequest, nil)
		return
	}
	utils.NewKVStore().SetDBPath(req.Path)
	sendResponse(w, map[string]string{"path": req.Path})
}

func handleLoginJsFetch(w http.ResponseWriter, r *http.Request) {
	sendResponse(w, map[string]string{"status": "success"})
}

func handleShouldNewLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		var req LoginRequest
		if err := decodeRequest(r, &req); err != nil {
			sendError(w, "Invalid request body", http.StatusBadRequest, nil)
			return
		}
		cacheQueue <- req
		sendResponse(w, map[string]string{"status": "success"})
	} else {
		select {
		case tokenInfo := <-cacheQueue:
			sendResponse(w, tokenInfo)
		default:
			// channel 为空时立即返回
			sendError(w, "No login request in cache", http.StatusNotFound, nil)
			return
		}
	}
}

// 处理 /status
func handleStatus(w http.ResponseWriter, r *http.Request) {
	sendResponse(w, map[string]string{"server_port": "56431", "proxy_port": "56432"})
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
