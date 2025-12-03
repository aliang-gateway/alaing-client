package cmd

import (
	"fmt"

	"nursor.org/nursorgate/common/logger"
)

// ExampleUsage 展示如何使用配置加载功能
func ExampleUsage() {
	// 加载配置文件
	configPath := "config.json"
	if err := LoadAndApplyConfig(configPath); err != nil {
		logger.Error(fmt.Sprintf("Failed to load config: %v", err))
		return
	}

	// 或者分步加载和应用
	config, err := LoadConfig(configPath)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to load config: %v", err))
		return
	}

	if err := ApplyConfig(config); err != nil {
		logger.Error(fmt.Sprintf("Failed to apply config: %v", err))
		return
	}
}
