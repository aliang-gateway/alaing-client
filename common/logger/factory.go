package logger

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// mainLoggerInstance is the global main logger instance
var (
	mainLoggerInstance Logger
	mainLoggerOnce     sync.Once
)

// GetMainLogger returns the global main logger instance
func GetMainLogger() Logger {
	mainLoggerOnce.Do(func() {
		mainLoggerInstance = NewMainLogger(GetLogConfig())
	})
	return mainLoggerInstance
}

// NewMainLogger creates a new main logger with the given configuration
func NewMainLogger(config *LogConfig) Logger {
	if config == nil {
		config = DefaultLogConfig()
	}

	// Create the log writer
	writers := []interface{}{os.Stdout}

	// Add file writer if path is specified
	if config.FileLogPath != "" {
		// Ensure directory exists
		logDir := filepath.Dir(config.FileLogPath)
		if err := os.MkdirAll(logDir, 0777); err == nil {
			// Set directory permissions
			os.Chmod(logDir, 0777)
			writers = append(writers, config.FileLogPath)
		}
	}

	return &mainLogger{
		config:  config,
		writers: writers,
		mu:      &sync.RWMutex{},
	}
}

// httpLoggerInstance is the global HTTP logger instance
var (
	httpLoggerInstance Logger
	httpLoggerOnce     sync.Once
)

// GetHTTPLogger returns the global HTTP logger instance
func GetHTTPLogger() Logger {
	httpLoggerOnce.Do(func() {
		httpLoggerInstance = NewHTTPLogger(HTTPLogConfig())
	})
	return httpLoggerInstance
}

// NewHTTPLogger creates a new HTTP logger with the given configuration
func NewHTTPLogger(config *LogConfig) Logger {
	if config == nil {
		config = HTTPLogConfig()
	}

	writers := []interface{}{}

	// Add file writer if path is specified
	if config.FileLogPath != "" {
		// Ensure directory exists
		logDir := filepath.Dir(config.FileLogPath)
		if err := os.MkdirAll(logDir, 0777); err == nil {
			// Set directory permissions
			os.Chmod(logDir, 0777)
			writers = append(writers, config.FileLogPath)
		}
	}

	return &httpLogger{
		config:  config,
		writers: writers,
		mu:      &sync.RWMutex{},
	}
}

// singBoxLoggerInstance is the global SingBox logger instance
var (
	singBoxLoggerInstance Logger
	singBoxLoggerOnce     sync.Once
)

// GetSingBoxLogger returns the global SingBox logger instance (thread-safe)
func GetSingBoxLogger() Logger {
	singBoxLoggerOnce.Do(func() {
		singBoxLoggerInstance = NewSingBoxLoggerAdapter(GetMainLogger())
	})
	return singBoxLoggerInstance
}

// NewSingBoxLoggerAdapter creates a new SingBox adapter that delegates to main logger
func NewSingBoxLoggerAdapter(baseLogger Logger) Logger {
	if baseLogger == nil {
		baseLogger = GetMainLogger()
	}

	return &singBoxAdapter{
		baseLogger: baseLogger,
		prefix:     "[VLESS]",
	}
}

// TracedLogger wraps a logger with connection tracing
type TracedLogger struct {
	baseLogger Logger
	traceID    string
}

// NewTracedLogger creates a new logger with connection tracing
func NewTracedLogger(baseLogger Logger, traceID string) Logger {
	if baseLogger == nil {
		baseLogger = GetMainLogger()
	}
	return &TracedLogger{
		baseLogger: baseLogger,
		traceID:    traceID,
	}
}

// Debug logs a debug message with trace ID
func (tl *TracedLogger) Debug(v ...interface{}) {
	tl.baseLogger.Debug(append([]interface{}{"[" + tl.traceID + "]"}, v...)...)
}

// Info logs an info message with trace ID
func (tl *TracedLogger) Info(v ...interface{}) {
	tl.baseLogger.Info(append([]interface{}{"[" + tl.traceID + "]"}, v...)...)
}

// Warn logs a warn message with trace ID
func (tl *TracedLogger) Warn(v ...interface{}) {
	tl.baseLogger.Warn(append([]interface{}{"[" + tl.traceID + "]"}, v...)...)
}

// Error logs an error message with trace ID
func (tl *TracedLogger) Error(v ...interface{}) {
	tl.baseLogger.Error(append([]interface{}{"[" + tl.traceID + "]"}, v...)...)
}

// Trace logs a trace message with trace ID
func (tl *TracedLogger) Trace(v ...interface{}) {
	tl.baseLogger.Trace(append([]interface{}{"[" + tl.traceID + "]"}, v...)...)
}

// Fatal logs a fatal message with trace ID and exits
func (tl *TracedLogger) Fatal(v ...interface{}) {
	tl.baseLogger.Fatal(append([]interface{}{"[" + tl.traceID + "]"}, v...)...)
}

// Panic logs a panic message with trace ID and panics
func (tl *TracedLogger) Panic(v ...interface{}) {
	tl.baseLogger.Panic(append([]interface{}{"[" + tl.traceID + "]"}, v...)...)
}

// Context variant methods for TracedLogger
func (tl *TracedLogger) DebugContext(ctx context.Context, v ...interface{}) {
	tl.baseLogger.DebugContext(ctx, append([]interface{}{"[" + tl.traceID + "]"}, v...)...)
}

func (tl *TracedLogger) InfoContext(ctx context.Context, v ...interface{}) {
	tl.baseLogger.InfoContext(ctx, append([]interface{}{"[" + tl.traceID + "]"}, v...)...)
}

func (tl *TracedLogger) WarnContext(ctx context.Context, v ...interface{}) {
	tl.baseLogger.WarnContext(ctx, append([]interface{}{"[" + tl.traceID + "]"}, v...)...)
}

func (tl *TracedLogger) ErrorContext(ctx context.Context, v ...interface{}) {
	tl.baseLogger.ErrorContext(ctx, append([]interface{}{"[" + tl.traceID + "]"}, v...)...)
}

func (tl *TracedLogger) TraceContext(ctx context.Context, v ...interface{}) {
	tl.baseLogger.TraceContext(ctx, append([]interface{}{"[" + tl.traceID + "]"}, v...)...)
}

func (tl *TracedLogger) FatalContext(ctx context.Context, v ...interface{}) {
	tl.baseLogger.FatalContext(ctx, append([]interface{}{"[" + tl.traceID + "]"}, v...)...)
}

func (tl *TracedLogger) PanicContext(ctx context.Context, v ...interface{}) {
	tl.baseLogger.PanicContext(ctx, append([]interface{}{"[" + tl.traceID + "]"}, v...)...)
}

// Flush flushes the base logger
func (tl *TracedLogger) Flush() {
	tl.baseLogger.Flush()
}

// WithContext for TracedLogger (implementation for interface)
func (tl *TracedLogger) WithContext(ctx context.Context) Logger {
	return tl
}

// httpLogger implements the Logger interface for HTTP-specific logging
type httpLogger struct {
	config  *LogConfig
	writers []interface{}
	mu      *sync.RWMutex
	loggers []*log.Logger
}

// Initialize HTTP logger from config
func (hl *httpLogger) initLoggers() {
	hl.mu.Lock()
	defer hl.mu.Unlock()

	if len(hl.loggers) > 0 {
		return // Already initialized
	}

	var writers []io.Writer

	// Add file writer if path is specified
	if hl.config.FileLogPath != "" {
		file, err := os.OpenFile(hl.config.FileLogPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err == nil {
			writers = append(writers, file)
			// Ensure file has 0666 permissions for all users
			os.Chmod(hl.config.FileLogPath, 0666)
		}
	}

	// Create logger
	var multiWriter io.Writer
	if len(writers) == 0 {
		multiWriter = io.Discard
	} else if len(writers) == 1 {
		multiWriter = writers[0]
	} else {
		multiWriter = io.MultiWriter(writers...)
	}

	logger := log.New(multiWriter, "", log.LstdFlags|log.Lshortfile)
	hl.loggers = append(hl.loggers, logger)
}

func (hl *httpLogger) Debug(v ...interface{}) {
	if hl.config.Level > DEBUG {
		return
	}
	hl.logf(DEBUG, "DEBUG", v...)
}

func (hl *httpLogger) Info(v ...interface{}) {
	if hl.config.Level > INFO {
		return
	}
	hl.logf(INFO, "INFO", v...)
}

func (hl *httpLogger) Warn(v ...interface{}) {
	if hl.config.Level > WARN {
		return
	}
	hl.logf(WARN, "WARN", v...)
}

func (hl *httpLogger) Error(v ...interface{}) {
	if hl.config.Level > ERROR {
		return
	}
	hl.logf(ERROR, "ERROR", v...)
	AppendToBuffer(&LogEntry{
		Level:     ERROR,
		Timestamp: time.Now(),
		Message:   fmt.Sprint(v...),
		Source:    "http",
	})
}

func (hl *httpLogger) Trace(v ...interface{}) {
	if hl.config.Level > TRACE {
		return
	}
	hl.logf(TRACE, "TRACE", v...)
}

func (hl *httpLogger) Fatal(v ...interface{}) {
	hl.logf(ERROR, "FATAL", v...)
	os.Exit(1)
}

func (hl *httpLogger) Panic(v ...interface{}) {
	msg := fmt.Sprint(v...)
	hl.logf(ERROR, "PANIC", v...)
	panic(msg)
}

// Context variants
func (hl *httpLogger) DebugContext(ctx context.Context, v ...interface{}) {
	hl.Debug(v...)
}

func (hl *httpLogger) InfoContext(ctx context.Context, v ...interface{}) {
	hl.Info(v...)
}

func (hl *httpLogger) WarnContext(ctx context.Context, v ...interface{}) {
	hl.Warn(v...)
}

func (hl *httpLogger) ErrorContext(ctx context.Context, v ...interface{}) {
	hl.Error(v...)
}

func (hl *httpLogger) TraceContext(ctx context.Context, v ...interface{}) {
	hl.Trace(v...)
}

func (hl *httpLogger) FatalContext(ctx context.Context, v ...interface{}) {
	hl.Fatal(v...)
}

func (hl *httpLogger) PanicContext(ctx context.Context, v ...interface{}) {
	hl.Panic(v...)
}

func (hl *httpLogger) WithContext(ctx context.Context) Logger {
	return hl
}

func (hl *httpLogger) Flush() {
	// No-op
}

func (hl *httpLogger) logf(level LogLevelType, prefix string, v ...interface{}) {
	hl.initLoggers()
	hl.mu.RLock()
	defer hl.mu.RUnlock()

	for _, logger := range hl.loggers {
		logger.Output(3, fmt.Sprintf("[%s] %s\n", prefix, fmt.Sprint(v...)))
	}
}

// singBoxAdapter implements the Logger interface as an adapter
type singBoxAdapter struct {
	baseLogger Logger
	prefix     string
}

func (sa *singBoxAdapter) Debug(v ...interface{}) {
	sa.baseLogger.Debug(v...)
}

func (sa *singBoxAdapter) Info(v ...interface{}) {
	sa.baseLogger.Info(v...)
}

func (sa *singBoxAdapter) Warn(v ...interface{}) {
	sa.baseLogger.Warn(v...)
}

func (sa *singBoxAdapter) Error(v ...interface{}) {
	sa.baseLogger.Error(v...)
}

func (sa *singBoxAdapter) Trace(v ...interface{}) {
	sa.baseLogger.Trace(v...)
}

func (sa *singBoxAdapter) Fatal(v ...interface{}) {
	sa.baseLogger.Fatal(v...)
}

func (sa *singBoxAdapter) Panic(v ...interface{}) {
	sa.baseLogger.Panic(v...)
}

// Context variants
func (sa *singBoxAdapter) DebugContext(ctx context.Context, v ...interface{}) {
	sa.baseLogger.DebugContext(ctx, v...)
}

func (sa *singBoxAdapter) InfoContext(ctx context.Context, v ...interface{}) {
	sa.baseLogger.InfoContext(ctx, v...)
}

func (sa *singBoxAdapter) WarnContext(ctx context.Context, v ...interface{}) {
	sa.baseLogger.WarnContext(ctx, v...)
}

func (sa *singBoxAdapter) ErrorContext(ctx context.Context, v ...interface{}) {
	sa.baseLogger.ErrorContext(ctx, v...)
}

func (sa *singBoxAdapter) TraceContext(ctx context.Context, v ...interface{}) {
	sa.baseLogger.TraceContext(ctx, v...)
}

func (sa *singBoxAdapter) FatalContext(ctx context.Context, v ...interface{}) {
	sa.baseLogger.FatalContext(ctx, v...)
}

func (sa *singBoxAdapter) PanicContext(ctx context.Context, v ...interface{}) {
	sa.baseLogger.PanicContext(ctx, v...)
}

func (sa *singBoxAdapter) WithContext(ctx context.Context) Logger {
	return sa
}

func (sa *singBoxAdapter) Flush() {
	sa.baseLogger.Flush()
}
