package logger

import (
	"errors"
	"time"
)

// ExampleUsage 展示如何使用改进后的日志记录器
func ExampleUsage() {
	// 1. 初始化日志系统
	err := Init()
	if err != nil {
		panic(err)
	}
	defer Shutdown()

	// 2. 可选：自定义错误去重配置
	config := &ErrorDedupConfig{
		ErrorWindow:     10 * time.Minute, // 10分钟窗口
		MaxErrorCount:   5,                // 最多发送5次
		CleanupInterval: 2 * time.Minute,  // 每2分钟清理一次
	}
	SetErrorDedupConfig(config)

	// 3. 使用日志记录器
	Info("应用启动成功")

	// 模拟重复错误
	testError := errors.New("网络连接失败")
	for i := 0; i < 20; i++ {
		Error("连接错误:", testError)
		// 前5次会发送到Sentry，后面的会被去重
	}

	Warn("这是一个警告信息")
}
