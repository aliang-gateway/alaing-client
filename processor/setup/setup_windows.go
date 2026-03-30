package setup

import (
	"fmt"

	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/mgr"
	"nursor.org/nursorgate/common/logger"
)

// WindowsServiceManager Windows 服务管理器
type WindowsServiceManager struct {
	name string
}

// NewServiceManager 创建当前平台的服务管理器
func NewServiceManager(opts InstallOptions) ServiceManager {
	return &WindowsServiceManager{
		name: opts.Name,
	}
}

// Install 安装 Windows 服务
func (w *WindowsServiceManager) Install(options InstallOptions) error {
	logger.Info("Installing Windows service...", "name", options.Name)

	// 检查权限
	if !IsRoot() {
		return ErrNotRoot
	}

	// 检查服务是否已存在
	if w.IsInstalled() {
		return ErrServiceExists
	}

	// 获取可执行文件路径
	execPath := options.ExecutablePath
	if execPath == "" {
		var err error
		execPath, err = GetCurrentExecutable()
		if err != nil {
			return fmt.Errorf("failed to get executable path: %w", err)
		}
	}

	// 打开服务管理器
	m, err := mgr.Connect()
	if err != nil {
		return fmt.Errorf("failed to connect to service manager: %w", err)
	}
	defer m.Disconnect()

	// 准备启动参数
	args := []string{}
	if options.ConfigPath != "" {
		args = append(args, "--config", options.ConfigPath)
	}
	if len(options.Args) > 0 {
		args = append(args, options.Args...)
	}

	// 确定启动类型
	var startType uint32 = mgr.StartManual
	switch options.StartType {
	case StartAutomatic:
		startType = mgr.StartAutomatic
	case StartManual:
		startType = mgr.StartManual
	case StartDisabled:
		startType = mgr.StartDisabled
	}

	// 创建服务
	s, err := m.CreateService(
		options.Name,
		execPath,
		mgr.Config{
			DisplayName: options.DisplayName,
			Description: options.Description,
			StartType:   startType,
		},
		args...,
	)
	if err != nil {
		return fmt.Errorf("failed to create service: %w", err)
	}
	defer s.Close()

	logger.Info("Service installed successfully")
	return nil
}

// Uninstall 卸载 Windows 服务
func (w *WindowsServiceManager) Uninstall() error {
	logger.Info("Uninstalling Windows service...", "name", w.name)

	if !w.IsInstalled() {
		return ErrServiceNotInstalled
	}

	// 打开服务管理器
	m, err := mgr.Connect()
	if err != nil {
		return fmt.Errorf("failed to connect to service manager: %w", err)
	}
	defer m.Disconnect()

	// 打开服务
	s, err := m.OpenService(w.name)
	if err != nil {
		return fmt.Errorf("failed to open service: %w", err)
	}
	defer s.Close()

	// 停止服务
	status, err := s.Query()
	if err == nil && status.State != svc.Stopped {
		if _, err := s.Control(svc.Stop); err != nil {
			return fmt.Errorf("failed to stop service before uninstall: %w", err)
		}
	}

	// 删除服务
	if err := s.Delete(); err != nil {
		return fmt.Errorf("failed to delete service: %w", err)
	}

	logger.Info("Service uninstalled successfully")
	return nil
}

// Start 启动 Windows 服务
func (w *WindowsServiceManager) Start() error {
	logger.Info("Starting Windows service...", "name", w.name)

	if !w.IsInstalled() {
		return ErrServiceNotInstalled
	}

	// 打开服务管理器
	m, err := mgr.Connect()
	if err != nil {
		return fmt.Errorf("failed to connect to service manager: %w", err)
	}
	defer m.Disconnect()

	// 打开服务
	s, err := m.OpenService(w.name)
	if err != nil {
		return fmt.Errorf("failed to open service: %w", err)
	}
	defer s.Close()

	// 启动服务
	if err := s.Start(); err != nil {
		return fmt.Errorf("failed to start service: %w", err)
	}

	logger.Info("Service started successfully")
	return nil
}

// Stop 停止 Windows 服务
func (w *WindowsServiceManager) Stop() error {
	logger.Info("Stopping Windows service...", "name", w.name)

	if !w.IsInstalled() {
		return ErrServiceNotInstalled
	}

	// 打开服务管理器
	m, err := mgr.Connect()
	if err != nil {
		return fmt.Errorf("failed to connect to service manager: %w", err)
	}
	defer m.Disconnect()

	// 打开服务
	s, err := m.OpenService(w.name)
	if err != nil {
		return fmt.Errorf("failed to open service: %w", err)
	}
	defer s.Close()

	// 停止服务
	_, err = s.Control(svc.Stop)
	if err != nil {
		return fmt.Errorf("failed to stop service: %w", err)
	}

	logger.Info("Service stopped successfully")
	return nil
}

// Restart 重启 Windows 服务
func (w *WindowsServiceManager) Restart() error {
	if err := w.Stop(); err != nil {
		return err
	}
	return w.Start()
}

// Status 获取 Windows 服务状态
func (w *WindowsServiceManager) Status() (*ServiceStatus, error) {
	status := &ServiceStatus{
		IsInstalled: w.IsInstalled(),
		IsRunning:   false,
		PID:         0,
		Status:      "unknown",
	}

	if !status.IsInstalled {
		status.Status = "not_installed"
		return status, nil
	}

	// 打开服务管理器
	m, err := mgr.Connect()
	if err != nil {
		return status, fmt.Errorf("failed to connect to service manager: %w", err)
	}
	defer m.Disconnect()

	// 打开服务
	s, err := m.OpenService(w.name)
	if err != nil {
		return status, nil
	}
	defer s.Close()

	// 查询服务状态
	svcStatus, err := s.Query()
	if err != nil {
		return status, nil
	}

	// 解析状态
	status.IsRunning = svcStatus.State == svc.Running
	status.PID = int(svcStatus.ProcessId)

	switch svcStatus.State {
	case svc.Stopped:
		status.Status = "stopped"
	case svc.StartPending:
		status.Status = "start_pending"
	case svc.StopPending:
		status.Status = "stop_pending"
	case svc.Running:
		status.Status = "running"
	case svc.ContinuePending:
		status.Status = "continue_pending"
	case svc.PausePending:
		status.Status = "pause_pending"
	case svc.Paused:
		status.Status = "paused"
	default:
		status.Status = "unknown"
	}

	return status, nil
}

// IsInstalled 检查服务是否已安装
func (w *WindowsServiceManager) IsInstalled() bool {
	m, err := mgr.Connect()
	if err != nil {
		return false
	}
	defer m.Disconnect()

	s, err := m.OpenService(w.name)
	if err != nil {
		return false
	}
	s.Close()

	return true
}

// GetName 获取服务名称
func (w *WindowsServiceManager) GetName() string {
	return w.name
}
