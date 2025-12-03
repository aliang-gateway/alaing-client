package http

import (
	"net/http"

	"nursor.org/nursorgate/app/http/handlers"
)

// RegisterLoggerRoutes registers all logger-related HTTP routes
func RegisterLoggerRoutes() {
	handler := handlers.NewLogHandler()
	streamHandler := handlers.NewLogStreamHandler()

	// Log retrieval and management
	http.HandleFunc("/api/logs", handler.HandleGetLogs)
	http.HandleFunc("/api/logs/clear", handler.HandleClearLogs)
	http.HandleFunc("/api/logs/level", handler.HandleSetLogLevel)
	http.HandleFunc("/api/logs/stream", streamHandler.HandleLogStream)

	// Logger configuration
	http.HandleFunc("/api/logs/config", handler.HandleLogConfig)
}
