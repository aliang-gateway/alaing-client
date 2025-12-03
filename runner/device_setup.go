package runner

import (
	"fmt"

	"nursor.org/nursorgate/common/logger"
	"nursor.org/nursorgate/processor/config"
	"nursor.org/nursorgate/runner/utils"
)

func GetDefaultDeviceConfiguration() config.EngineConf {
	// 获取默认网络接口
	defaultInterface, err := utils.GetDefaultInterface()
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to get default interface: %v", err))
		defaultInterface = "en0" // 设置一个默认值
	}

	defaultEngineConf := config.EngineConf{
		MTU:       0,
		Mark:      0,
		Device:    utils.GetDefaultTunName(),
		Interface: defaultInterface,
	}
	return defaultEngineConf
}
