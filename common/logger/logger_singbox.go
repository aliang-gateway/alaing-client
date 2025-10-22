package logger

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"
)

// SingBoxLogger 实现 sing-box 的 Logger 和 ContextLogger 接口
// 将所有日志输出到命令行（标准输出）
type SingBoxLogger struct {
	prefix string
	logger *log.Logger
}

// NewSingBoxLogger 创建一个新的 SingBox 兼容的 logger
func NewSingBoxLogger() *SingBoxLogger {
	return &SingBoxLogger{
		prefix: "[VLESS]",
		logger: log.New(os.Stdout, "", 0), // 不使用默认前缀，我们自己格式化
	}
}

// NewSingBoxLoggerWithPrefix 创建带自定义前缀的 logger
func NewSingBoxLoggerWithPrefix(prefix string) *SingBoxLogger {
	return &SingBoxLogger{
		prefix: prefix,
		logger: log.New(os.Stdout, "", 0),
	}
}

// formatMessage 格式化日志消息
func (l *SingBoxLogger) formatMessage(level string, args ...any) string {
	timestamp := time.Now().Format("2006-01-02 15:04:05.000")
	message := fmt.Sprint(args...)
	return fmt.Sprintf("%s [%s] %s %s", timestamp, level, l.prefix, message)
}

// Trace 输出 TRACE 级别日志
func (l *SingBoxLogger) Trace(args ...any) {
	l.logger.Println(l.formatMessage("TRACE", args...))
}

// Debug 输出 DEBUG 级别日志
func (l *SingBoxLogger) Debug(args ...any) {
	l.logger.Println(l.formatMessage("DEBUG", args...))
}

// Info 输出 INFO 级别日志
func (l *SingBoxLogger) Info(args ...any) {
	l.logger.Println(l.formatMessage("INFO", args...))
}

// Warn 输出 WARN 级别日志
func (l *SingBoxLogger) Warn(args ...any) {
	l.logger.Println(l.formatMessage("WARN", args...))
}

// Error 输出 ERROR 级别日志
func (l *SingBoxLogger) Error(args ...any) {
	l.logger.Println(l.formatMessage("ERROR", args...))
}

// Fatal 输出 FATAL 级别日志并退出程序
func (l *SingBoxLogger) Fatal(args ...any) {
	l.logger.Println(l.formatMessage("FATAL", args...))
	os.Exit(1)
}

// Panic 输出 PANIC 级别日志并触发 panic
func (l *SingBoxLogger) Panic(args ...any) {
	msg := l.formatMessage("PANIC", args...)
	l.logger.Println(msg)
	panic(msg)
}

// TraceContext 输出带上下文的 TRACE 级别日志
func (l *SingBoxLogger) TraceContext(ctx context.Context, args ...any) {
	l.logger.Println(l.formatMessage("TRACE", args...))
}

// DebugContext 输出带上下文的 DEBUG 级别日志
func (l *SingBoxLogger) DebugContext(ctx context.Context, args ...any) {
	l.logger.Println(l.formatMessage("DEBUG", args...))
}

// InfoContext 输出带上下文的 INFO 级别日志
func (l *SingBoxLogger) InfoContext(ctx context.Context, args ...any) {
	l.logger.Println(l.formatMessage("INFO", args...))
}

// WarnContext 输出带上下文的 WARN 级别日志
func (l *SingBoxLogger) WarnContext(ctx context.Context, args ...any) {
	l.logger.Println(l.formatMessage("WARN", args...))
}

// ErrorContext 输出带上下文的 ERROR 级别日志
func (l *SingBoxLogger) ErrorContext(ctx context.Context, args ...any) {
	l.logger.Println(l.formatMessage("ERROR", args...))
}

// FatalContext 输出带上下文的 FATAL 级别日志并退出程序
func (l *SingBoxLogger) FatalContext(ctx context.Context, args ...any) {
	l.logger.Println(l.formatMessage("FATAL", args...))
	os.Exit(1)
}

// PanicContext 输出带上下文的 PANIC 级别日志并触发 panic
func (l *SingBoxLogger) PanicContext(ctx context.Context, args ...any) {
	msg := l.formatMessage("PANIC", args...)
	l.logger.Println(msg)
	panic(msg)
}

// singBoxLoggerInstance 全局单例
var singBoxLoggerInstance *SingBoxLogger

// GetSingBoxLogger 获取全局 SingBox logger 实例
func GetSingBoxLogger() *SingBoxLogger {
	if singBoxLoggerInstance == nil {
		singBoxLoggerInstance = NewSingBoxLogger()
	}
	return singBoxLoggerInstance
}

// SilentLogger 静默 logger（不输出任何内容）
type SilentLogger struct{}

func (l *SilentLogger) Trace(args ...any)                             {}
func (l *SilentLogger) Debug(args ...any)                             {}
func (l *SilentLogger) Info(args ...any)                              {}
func (l *SilentLogger) Warn(args ...any)                              {}
func (l *SilentLogger) Error(args ...any)                             {}
func (l *SilentLogger) Fatal(args ...any)                             { os.Exit(1) }
func (l *SilentLogger) Panic(args ...any)                             { panic("silent panic") }
func (l *SilentLogger) TraceContext(ctx context.Context, args ...any) {}
func (l *SilentLogger) DebugContext(ctx context.Context, args ...any) {}
func (l *SilentLogger) InfoContext(ctx context.Context, args ...any)  {}
func (l *SilentLogger) WarnContext(ctx context.Context, args ...any)  {}
func (l *SilentLogger) ErrorContext(ctx context.Context, args ...any) {}
func (l *SilentLogger) FatalContext(ctx context.Context, args ...any) { os.Exit(1) }
func (l *SilentLogger) PanicContext(ctx context.Context, args ...any) { panic("silent panic") }

// NewSilentLogger 创建静默 logger
func NewSilentLogger() *SilentLogger {
	return &SilentLogger{}
}
