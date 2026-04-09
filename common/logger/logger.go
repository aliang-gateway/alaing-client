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
	"gopkg.in/natefinch/lumberjack.v2"
)

var (
	errorCache   = make(map[string]*errorInfo)
	errorCacheMu sync.RWMutex
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
	writers []io.Writer
	mu      *sync.RWMutex
	loggers []*log.Logger
}

// initLoggers initializes the loggers with rotation support
func (ml *mainLogger) initLoggers() {
	ml.mu.Lock()
	defer ml.mu.Unlock()

	if len(ml.loggers) > 0 {
		return // Already initialized
	}

	var writers []io.Writer

	// Always add stdout
	writers = append(writers, os.Stdout)

	// Add file writer with rotation if path is specified
	if ml.config.FileLogPath != "" {
		if ml.config.EnableFileRotation {
			// Use lumberjack for rotation
			rotateLogger := &lumberjack.Logger{
				Filename:   ml.config.FileLogPath,
				MaxSize:    int(ml.config.MaxLogSize / 1024 / 1024), // lumberjack uses MB
				MaxBackups: ml.config.MaxLogBackups,
				Compress:   true, // compress rotated files
			}
			writers = append(writers, rotateLogger)
		} else {
			// Simple append mode
			file, err := os.OpenFile(ml.config.FileLogPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
			if err == nil {
				writers = append(writers, file)
				os.Chmod(ml.config.FileLogPath, 0666)
			}
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
		msg := fmt.Sprint(v...)
		sentry.WithScope(func(scope *sentry.Scope) {
			scope.SetTag("source", "mainLogger")
			scope.SetExtra("raw_args", fmt.Sprintf("%v", v))
			sentry.CaptureMessage(msg)
		})
		go sentry.Flush(2 * time.Second)
	}
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

// Context variants — currently context is not utilized, kept for interface compatibility
func (ml *mainLogger) DebugContext(ctx context.Context, v ...interface{}) {
	ml.Debug(v...)
}

// Context variants — currently context is not utilized, kept for interface compatibility
func (ml *mainLogger) InfoContext(ctx context.Context, v ...interface{}) {
	ml.Info(v...)
}

// Context variants — currently context is not utilized, kept for interface compatibility
func (ml *mainLogger) WarnContext(ctx context.Context, v ...interface{}) {
	ml.Warn(v...)
}

// Context variants — currently context is not utilized, kept for interface compatibility
func (ml *mainLogger) ErrorContext(ctx context.Context, v ...interface{}) {
	ml.Error(v...)
}

// Context variants — currently context is not utilized, kept for interface compatibility
func (ml *mainLogger) TraceContext(ctx context.Context, v ...interface{}) {
	ml.Trace(v...)
}

// Context variants — currently context is not utilized, kept for interface compatibility
func (ml *mainLogger) FatalContext(ctx context.Context, v ...interface{}) {
	ml.Fatal(v...)
}

// Context variants — currently context is not utilized, kept for interface compatibility
func (ml *mainLogger) PanicContext(ctx context.Context, v ...interface{}) {
	ml.Panic(v...)
}

// WithContext returns a logger with context.
// NOTE: context is currently not utilized; this is kept for interface compatibility.
func (ml *mainLogger) WithContext(ctx context.Context) Logger {
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

	message := fmt.Sprint(v...)
	for _, logger := range ml.loggers {
		logger.Output(3, fmt.Sprintf("[%s] %s\n", prefix, message))
	}

	AppendToBuffer(&LogEntry{
		Level:     level,
		Timestamp: time.Now(),
		Message:   message,
		Source:    "main",
	})
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

func init() {
	startCleanupRoutineOnce()
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

// Backward compatible global logging functions
func Debug(v ...interface{}) {
	GetMainLogger().Debug(v...)
}

func Info(v ...interface{}) {
	GetMainLogger().Info(v...)
}

func Warn(v ...interface{}) {
	GetMainLogger().Warn(v...)
}

func Error(v ...interface{}) {
	GetMainLogger().Error(v...)
}

func SetUserInfo(userIdentity string) {
	sentry.ConfigureScope(func(scope *sentry.Scope) {
		scope.SetTag("session_user", userIdentity)
	})
}
