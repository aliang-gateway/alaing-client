package logger

import (
	"crypto/md5"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/getsentry/sentry-go"
)

var (
	// 全局日志记录器
	logger *log.Logger
	// 日志文件
	logFile     *os.File
	logFilePath string

	// 错误去重相关变量
	errorCache    = make(map[string]*errorInfo)
	errorCacheMux sync.RWMutex
	errorWindow   = 5 * time.Minute // 错误去重时间窗口
	maxErrorCount = 10              // 同一错误在时间窗口内的最大发送次数
	cleanupTicker *time.Ticker
	cleanupDone   chan bool
)

// errorInfo 存储错误信息
type errorInfo struct {
	Count     int
	FirstSeen time.Time
	LastSeen  time.Time
}

func init() {
	logger = log.New(os.Stdout, "", log.LstdFlags)
	logger.SetFlags(log.LstdFlags | log.Lshortfile)

	// 启动清理协程
	home, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}
	logDir := filepath.Join(home, ".nursor")
	os.MkdirAll(logDir, 0755)
	logFilePath = filepath.Join(logDir, "running.log")

	// 打开文件（追加模式）
	f, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		panic(err)
	}
	logger = log.New(f, "", log.LstdFlags)
	startCleanupRoutine()

}

// startCleanupRoutine 启动定期清理过期错误记录的协程
func startCleanupRoutine() {
	cleanupTicker = time.NewTicker(errorDedupConfig.CleanupInterval)
	cleanupDone = make(chan bool)

	go func() {
		for {
			select {
			case <-cleanupTicker.C:
				cleanupExpiredErrors()
			case <-cleanupDone:
				return
			}
		}
	}()
}

// cleanupExpiredErrors 清理过期的错误记录
func cleanupExpiredErrors() {
	errorCacheMux.Lock()
	defer errorCacheMux.Unlock()

	now := time.Now()
	for key, info := range errorCache {
		if now.Sub(info.LastSeen) > errorWindow {
			delete(errorCache, key)
		}
	}
}

// generateErrorHash 生成错误的哈希值用于去重
func generateErrorHash(v ...interface{}) string {
	h := md5.New()
	fmt.Fprint(h, v...)
	return fmt.Sprintf("%x", h.Sum(nil))
}

// shouldSendError 判断是否应该发送错误到Sentry
func shouldSendError(errorHash string) bool {
	errorCacheMux.Lock()
	defer errorCacheMux.Unlock()

	now := time.Now()

	// 检查是否已存在该错误
	if info, exists := errorCache[errorHash]; exists {
		// 如果错误在时间窗口内且未超过最大次数，则发送
		if now.Sub(info.FirstSeen) <= errorWindow && info.Count < maxErrorCount {
			info.Count++
			info.LastSeen = now
			return true
		}
		// 如果超过最大次数，更新最后出现时间但不发送
		info.LastSeen = now
		return false
	}

	// 新错误，添加到缓存并发送
	errorCache[errorHash] = &errorInfo{
		Count:     1,
		FirstSeen: now,
		LastSeen:  now,
	}
	return true
}

func Info(v ...interface{}) {
	log.Println(v...)
}

func Error(v ...interface{}) {
	log.Println(v...)
	fmt.Println(v...)

	// 生成错误哈希
	errorHash := generateErrorHash(v...)

	// 检查是否应该发送到Sentry
	if shouldSendError(errorHash) {
		sentry.CaptureMessage(fmt.Sprintf("%v", v...))
		go func() {
			sentry.Flush(2 * time.Second)
		}()
	}
}

func Warn(v ...interface{}) {
	log.Println(v...)
}

// 初始化日志记录器
func Init() error {
	// 获取用户主目录
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("获取用户主目录失败: %v", err)
	}

	// 创建日志目录
	logDir := filepath.Join(homeDir, ".nursor")
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return fmt.Errorf("创建日志目录失败: %v", err)
	}

	// 打开日志文件
	logPath := filepath.Join(logDir, "nursor_core.log")
	logFile, err = os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("打开日志文件失败: %v", err)
	}

	// 创建日志记录器
	logger = log.New(logFile, "", log.LstdFlags)
	logger.Printf("日志系统初始化成功，日志文件: %s", logPath)

	return nil
}

func GetCustomLogger() *log.Logger {
	return logger
}

// Shutdown 关闭日志系统，清理资源
func Shutdown() {
	if cleanupTicker != nil {
		cleanupTicker.Stop()
		close(cleanupDone)
	}

	if logFile != nil {
		logFile.Close()
	}
}
