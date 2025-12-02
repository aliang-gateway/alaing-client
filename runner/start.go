package runner

import (
	"fmt"
	"os/signal"
	"syscall"
	"time"

	"go.uber.org/automaxprocs/maxprocs"
	"nursor.org/nursorgate/common/logger"
	"nursor.org/nursorgate/common/model"
	"nursor.org/nursorgate/inbound/tun/engine"
)

func Start() {
	defer func() {
		if r := recover(); r != nil {
			logger.Error(fmt.Sprintf("Recovered from panic in Start: %v", r))
			RunStatusChan <- map[string]string{"status": "failed", "message": fmt.Sprintf("Recovered from panic in Start: %v", r)}
		}
	}()

	domains := model.NewAllowProxyDomain()
	logger.Info(fmt.Sprintf("domain is: %v", domains))

	maxprocs.Set(maxprocs.Logger(func(string, ...any) {}))

	defaultConfig = GetDefaultDeviceConfiguration()

	// 添加设备状态监控
	go monitorTunDevice(defaultConfig.Device)

	engine.Insert(&defaultConfig)
	if err := engine.Start(); err != nil {
		logger.Error("engine 启动失败")
		RunStatusChan <- map[string]string{"status": "failed", "message": err.Error()}
		return
	}

	defer engine.Stop()
	_dfgw, err := GetDefaultGateway()
	if err != nil {
		logger.Error("获取默认网关失败: ", err)
		RunStatusChan <- map[string]string{"status": "failed", "message": err.Error()}
		return
	}
	defaultGateway = _dfgw

	if err := ConfigureTunInterface(defaultConfig.Device); err != nil {
		logger.Error(fmt.Sprintf("配置 TUN 接口失败: %v", err))
		RunStatusChan <- map[string]string{"status": "failed", "message": err.Error()}
		return
	}

	// 等待设备就绪，最多等待10秒
	if err := waitForTunDeviceReady(defaultConfig.Device, 10*time.Second); err != nil {
		logger.Error(fmt.Sprintf("等待 TUN 设备就绪失败: %v", err))
		RunStatusChan <- map[string]string{"status": "failed", "message": err.Error()}
		return
	}

	if err := ConfigureTunRoute(); err != nil {
		logger.Error(fmt.Sprintf("配置 TUN 路由失败: %v", err))
		RunStatusChan <- map[string]string{"status": "failed", "message": err.Error()}
		return
	}

	logger.Info("TUN 服务启动成功，设备名称: ", defaultConfig.Interface)
	RunStatusChan <- map[string]string{"status": "success", "message": "TUN service started successfully"}

	signal.Notify(TunSignal, syscall.SIGINT, syscall.SIGTERM)
	<-TunSignal

	// 收到信号后调用 Stop
	stopTun()
}

func Stop() {
	TunSignal <- syscall.SIGTERM // 或其他自定义信号
}
