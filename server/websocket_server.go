package server

import (
	"fmt"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
	"nursor.org/nursorgate/common/logger"
)

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

// StartWebSocketServer 启动WebSocket服务器
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

// handleWebSocket 处理WebSocket连接
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
