package logger

import (
	"crypto/md5"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	"github.com/getsentry/sentry-go"
)

type LogLevel int

const (
	DEBUG LogLevel = iota
	INFO
	WARN
	ERROR
)

var (
	currentLevel = INFO

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

// 设置日志等级
func SetLogLevel(level LogLevel) {
	currentLevel = level
}

func init() {
	Init()
	InitHttp()
}

// 初始化日志系统
func Init() error {
	var home string
	var err error

	if runtime.GOOS == "darwin" {
		home = "/Library/Logs/Nursor"
	} else {
		home, err = os.UserHomeDir()
		if err != nil {
			return err
		}
	}

	logDir := filepath.Join(home, ".nursor")
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return err
	}

	logFilePath = filepath.Join(logDir, "nursor_core.log")
	logFile, err = os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return err
	}

	if LogSilent == "true" {
		logger = log.New(io.Discard, "", log.LstdFlags|log.Lshortfile)
	} else {
		logger = log.New(logFile, "", log.LstdFlags|log.Lshortfile)
	}

	startCleanupRoutine()

	return nil
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
		if now.Sub(v.LastSeen) > errorWindow {
			delete(errorCache, k)
		}
	}
}

func logf(level LogLevel, prefix string, v ...interface{}) {
	if level < currentLevel {
		return
	}
	logger.Output(3, fmt.Sprintf("[%s] %s\n", prefix, fmt.Sprint(v...)))
}

func Debug(v ...interface{}) { logf(DEBUG, "DEBUG", v...) }
func Info(v ...interface{})  { logf(INFO, "INFO", v...) }
func Warn(v ...interface{})  { logf(WARN, "WARN", v...) }

func Error(v ...interface{}) {
	logf(ERROR, "ERROR", v...)
	errHash := generateErrorHash(v...)
	if shouldSendError(errHash) {
		sentry.CaptureMessage(fmt.Sprint(v...))
		go sentry.Flush(2 * time.Second)
	}
}

func generateErrorHash(v ...interface{}) string {
	h := md5.New()
	fmt.Fprint(h, v...)
	return fmt.Sprintf("%x", h.Sum(nil))
}

func shouldSendError(hash string) bool {
	errorCacheMu.Lock()
	defer errorCacheMu.Unlock()

	now := time.Now()
	if info, exists := errorCache[hash]; exists {
		if now.Sub(info.FirstSeen) <= errorWindow && info.Count < maxErrorCnt {
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

func SetUserInfo(userID string) {
	sentry.ConfigureScope(func(scope *sentry.Scope) {
		scope.SetTag("user_id", userID)
	})
}
