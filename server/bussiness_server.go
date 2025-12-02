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
	// 定义 HTTP 服务端口
	port := "127.0.0.1:56431"
	// 注册路由
	http.HandleFunc("/token/set", handleTokenSet)
	http.HandleFunc("/token/get", handleTokenGet)
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
