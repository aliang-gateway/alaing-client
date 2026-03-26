package middleware

import (
	"bufio"
	"fmt"
	"net"
	"net/http"
	"time"

	"nursor.org/nursorgate/common/logger"
)

// LoggingMiddleware logs incoming requests and outgoing responses
func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Record request start time
		startTime := time.Now()

		// Create a response writer wrapper to capture status code
		wrappedWriter := &responseWriterWrapper{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}

		// Log request
		logger.HttpTrace(fmt.Sprintf("[HTTP] %s %s from %s", r.Method, r.RequestURI, r.RemoteAddr))

		// Call next handler
		next.ServeHTTP(wrappedWriter, r)

		// Calculate elapsed time
		elapsed := time.Since(startTime)

		// Log response
		logger.HttpTrace(fmt.Sprintf("[HTTP] %s %s - Status: %d - Duration: %v",
			r.Method, r.RequestURI, wrappedWriter.statusCode, elapsed))
	})
}

// responseWriterWrapper wraps http.ResponseWriter to capture status code
type responseWriterWrapper struct {
	http.ResponseWriter
	statusCode int
}

// WriteHeader captures the status code
func (w *responseWriterWrapper) WriteHeader(statusCode int) {
	w.statusCode = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}

// Hijack implements http.Hijacker interface for WebSocket support
func (w *responseWriterWrapper) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if hijacker, ok := w.ResponseWriter.(http.Hijacker); ok {
		return hijacker.Hijack()
	}
	return nil, nil, fmt.Errorf("underlying ResponseWriter does not implement http.Hijacker")
}
