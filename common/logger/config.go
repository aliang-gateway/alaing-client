package logger

import "time"

// ErrorDedupConfig 错误去重配置
type ErrorDedupConfig struct {
	// 错误去重时间窗口，默认5分钟
	ErrorWindow time.Duration
	// 同一错误在时间窗口内的最大发送次数，默认10次
	MaxErrorCount int
	// 清理间隔，默认1分钟
	CleanupInterval time.Duration
}

// DefaultErrorDedupConfig 返回默认的错误去重配置
func DefaultErrorDedupConfig() *ErrorDedupConfig {
	return &ErrorDedupConfig{
		ErrorWindow:     1 * time.Hour,
		MaxErrorCount:   4,
		CleanupInterval: 2 * time.Hour,
	}
}

// 全局配置变量
var errorDedupConfig = DefaultErrorDedupConfig()

// SetErrorDedupConfig 设置错误去重配置
func SetErrorDedupConfig(config *ErrorDedupConfig) {
	if config != nil {
		errorDedupConfig = config
		// 更新全局变量
		errorWindow = config.ErrorWindow
		// maxErrorCount = config.MaxErrorCount
	}
}
