package logger

import (
	"context"
	"crypto/md5"
	"fmt"
	"io"
	"log"
	"os"
	"sync"
	"time"

	"github.com/getsentry/sentry-go"
)

// LogLevel is kept for backward compatibility
type LogLevel int

const (
	DEBUG_COMPAT LogLevel = iota
	INFO_COMPAT
	WARN_COMPAT
	ERROR_COMPAT
)

var (
	// For backward compatibility with global functions
	currentLevel = WARN_COMPAT

	logger       *log.Logger
	logFile      *os.File
	logFilePath  string
	errorCache   = make(map[string]*errorInfo)
	errorCacheMu sync.RWMutex
	errorWindow  = 5 * time.Minute
	maxErrorCnt  = 10
	cleanupTick  *time.Ticker
	cleanupDone  chan bool
)

type errorInfo struct {
	Count     int
	FirstSeen time.Time
	LastSeen  time.Time
}

// mainLogger implements the Logger interface
type mainLogger struct {
	config  *LogConfig
	writers []interface{}
	mu      *sync.RWMutex
	loggers []*log.Logger
}

// Initialize main logger implementation from config
func (ml *mainLogger) initLoggers() {
	ml.mu.Lock()
	defer ml.mu.Unlock()

	if len(ml.loggers) > 0 {
		return // Already initialized
	}

	var writers []io.Writer

	// Always add stdout
	writers = append(writers, os.Stdout)

	// Add file writer if path is specified
	if ml.config.FileLogPath != "" {
		file, err := os.OpenFile(ml.config.FileLogPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err == nil {
			writers = append(writers, file)
		}
	}

	// Create multi-writer
	var multiWriter io.Writer
	if len(writers) == 1 {
		multiWriter = writers[0]
	} else {
		multiWriter = io.MultiWriter(writers...)
	}

	// Create logger with multi-writer
	logger := log.New(multiWriter, "", log.LstdFlags|log.Lshortfile)
	ml.loggers = append(ml.loggers, logger)
}

func (ml *mainLogger) Debug(v ...interface{}) {
	if ml.config.Level > DEBUG {
		return
	}
	ml.logf(DEBUG, "DEBUG", v...)
}

func (ml *mainLogger) Info(v ...interface{}) {
	if ml.config.Level > INFO {
		return
	}
	ml.logf(INFO, "INFO", v...)
}

func (ml *mainLogger) Warn(v ...interface{}) {
	if ml.config.Level > WARN {
		return
	}
	ml.logf(WARN, "WARN", v...)
}

func (ml *mainLogger) Error(v ...interface{}) {
	if ml.config.Level > ERROR {
		return
	}
	ml.logf(ERROR, "ERROR", v...)

	// Error deduplication and Sentry
	errHash := ml.generateErrorHash(v...)
	if ml.shouldSendError(errHash) && ml.config.EnableSentry {
		sentry.CaptureMessage(fmt.Sprint(v...))
		go sentry.Flush(2 * time.Second)
	}

	// Add to buffer
	AppendToBuffer(&LogEntry{
		Level:     ERROR,
		Timestamp: time.Now(),
		Message:   fmt.Sprint(v...),
		Source:    "main",
	})
}

func (ml *mainLogger) Trace(v ...interface{}) {
	if ml.config.Level > TRACE {
		return
	}
	ml.logf(TRACE, "TRACE", v...)
}

func (ml *mainLogger) Fatal(v ...interface{}) {
	ml.logf(ERROR, "FATAL", v...)
	os.Exit(1)
}

func (ml *mainLogger) Panic(v ...interface{}) {
	msg := fmt.Sprint(v...)
	ml.logf(ERROR, "PANIC", v...)
	panic(msg)
}

// Context variants
func (ml *mainLogger) DebugContext(ctx context.Context, v ...interface{}) {
	ml.Debug(v...)
}

func (ml *mainLogger) InfoContext(ctx context.Context, v ...interface{}) {
	ml.Info(v...)
}

func (ml *mainLogger) WarnContext(ctx context.Context, v ...interface{}) {
	ml.Warn(v...)
}

func (ml *mainLogger) ErrorContext(ctx context.Context, v ...interface{}) {
	ml.Error(v...)
}

func (ml *mainLogger) TraceContext(ctx context.Context, v ...interface{}) {
	ml.Trace(v...)
}

func (ml *mainLogger) FatalContext(ctx context.Context, v ...interface{}) {
	ml.Fatal(v...)
}

func (ml *mainLogger) PanicContext(ctx context.Context, v ...interface{}) {
	ml.Panic(v...)
}

// WithContext returns a logger with context (for tracing)
func (ml *mainLogger) WithContext(ctx context.Context) Logger {
	// In a full implementation, extract trace ID from context
	// For now, just return self
	return ml
}

// Flush flushes all writers
func (ml *mainLogger) Flush() {
	// No-op for stdout/file writers
}

// logf formats and logs a message
func (ml *mainLogger) logf(level LogLevelType, prefix string, v ...interface{}) {
	ml.initLoggers()
	ml.mu.RLock()
	defer ml.mu.RUnlock()

	for _, logger := range ml.loggers {
		logger.Output(3, fmt.Sprintf("[%s] %s\n", prefix, fmt.Sprint(v...)))
	}
}

func (ml *mainLogger) generateErrorHash(v ...interface{}) string {
	h := md5.New()
	fmt.Fprint(h, v...)
	return fmt.Sprintf("%x", h.Sum(nil))
}

func (ml *mainLogger) shouldSendError(hash string) bool {
	errorCacheMu.Lock()
	defer errorCacheMu.Unlock()

	now := time.Now()
	if info, exists := errorCache[hash]; exists {
		if now.Sub(info.FirstSeen) <= ml.config.ErrorWindow && info.Count < ml.config.MaxErrorCount {
			info.Count++
			info.LastSeen = now
			return true
		}
		info.LastSeen = now
		return false
	}
	errorCache[hash] = &errorInfo{Count: 1, FirstSeen: now, LastSeen: now}
	return true
}

// Backward compatibility functions
// 设置日志等级
func SetLogLevel(level LogLevel) {
	currentLevel = level
}

func init() {
	err := Init()
	if err != nil {
		fmt.Println("init failure")
	}
	err2 := InitHttp()
	if err2 != nil {
		fmt.Println("init http failure")
	}
}

// 初始化日志系统
func Init() error {
	if LogSilent == "true" {
		logger = log.New(io.Discard, "", log.LstdFlags|log.Lshortfile)
	} else {
		logger = log.New(os.Stdout, "", log.LstdFlags|log.Lshortfile)
	}

	startCleanupRoutineOnce()

	return nil
}

var cleanupOnce sync.Once

func startCleanupRoutineOnce() {
	cleanupOnce.Do(func() {
		startCleanupRoutine()
	})
}

func Shutdown() {
	if cleanupTick != nil {
		cleanupTick.Stop()
		close(cleanupDone)
	}
	if logFile != nil {
		logFile.Close()
	}
}

func startCleanupRoutine() {
	cleanupTick = time.NewTicker(1 * time.Minute)
	cleanupDone = make(chan bool)

	go func() {
		for {
			select {
			case <-cleanupTick.C:
				cleanupExpiredErrors()
			case <-cleanupDone:
				return
			}
		}
	}()
}

func cleanupExpiredErrors() {
	errorCacheMu.Lock()
	defer errorCacheMu.Unlock()

	now := time.Now()
	for k, v := range errorCache {
		if now.Sub(v.LastSeen) > GetLogConfig().ErrorWindow {
			delete(errorCache, k)
		}
	}
}

func logf(level LogLevel, prefix string, v ...interface{}) {
	if level < currentLevel {
		return
	}
	err := logger.Output(3, fmt.Sprintf("[%s] %s\n", prefix, fmt.Sprint(v...)))
	if err != nil {
		return
	}
}

// Backward compatible global logging functions
func Debug(v ...interface{}) {
	logf(DEBUG_COMPAT, "DEBUG", v...)
	GetMainLogger().Debug(v...)
}

func Info(v ...interface{})  {
	logf(INFO_COMPAT, "INFO", v...)
	GetMainLogger().Info(v...)
}

func Warn(v ...interface{})  {
	logf(WARN_COMPAT, "WARN", v...)
	GetMainLogger().Warn(v...)
}

func Error(v ...interface{}) {
	logf(ERROR_COMPAT, "ERROR", v...)
	GetMainLogger().Error(v...)
}

func SetUserInfo(innerToken string) {
	sentry.ConfigureScope(func(scope *sentry.Scope) {
		scope.SetTag("inner_token", innerToken)
	})
}
