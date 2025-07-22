package logger

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime"
)

var (
	currentHttpLevel = INFO

	httpLogger      *log.Logger
	httpLogFile     *os.File
	httpLogFilePath string
)

var LogSilent = "false1"

// 设置日志等级
func SetHttpLogLevel(level LogLevel) {
	currentHttpLevel = level
}

// 初始化日志系统
func InitHttp() error {
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

	httpLogFilePath = filepath.Join(logDir, "nursor_http.log")
	httpLogFile, err = os.OpenFile(httpLogFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return err
	}
	if LogSilent == "true" {
		httpLogger = log.New(io.Discard, "", log.LstdFlags|log.Lshortfile)
	} else {
		httpLogger = log.New(httpLogFile, "", log.LstdFlags|log.Lshortfile)
	}

	startCleanupRoutine()

	return nil
}

func httpLogf(level LogLevel, prefix string, v ...interface{}) {
	if level < currentHttpLevel {
		return
	}
	httpLogger.Output(3, fmt.Sprintf("[%s] %s\n", prefix, fmt.Sprint(v...)))
}

func HttpDebug(v ...interface{}) { httpLogf(DEBUG, "DEBUG", v...) }
func HttpInfo(v ...interface{})  { httpLogf(INFO, "INFO", v...) }
func HttpWarn(v ...interface{})  { httpLogf(WARN, "WARN", v...) }
