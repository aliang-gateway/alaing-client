package tun

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"

	"nursor.org/nursorgate/common/model"

	"runtime"
	"time"

	"go.uber.org/automaxprocs/maxprocs"
	"nursor.org/nursorgate/client/server/tun/engine"
	"nursor.org/nursorgate/client/utils"
	"nursor.org/nursorgate/common/logger"
)

var TunSignal = make(chan os.Signal, 1)
var RunStatusChan = make(chan map[string]string, 1)
var defaultKey engine.Key
var defaultGateway = "192.168.1.1"

func Start() {
	defer func() {
		if r := recover(); r != nil {
			logger.Error(fmt.Sprintf("Recovered from panic in Start: %v", r))
		}
	}()

	domains := model.NewAllowProxyDomain()
	logger.Info(fmt.Sprintf("domain is: %v", domains))

	maxprocs.Set(maxprocs.Logger(func(string, ...any) {}))

	defaultKey = InitArgs()

	// 添加设备状态监控
	go monitorTunDevice(defaultKey.Device)

	engine.Insert(&defaultKey)
	engine.Start()
	defer engine.Stop()
	defaultGateway2, err := getDefaultGateway()
	if err != nil {
		logger.Error("获取默认网关失败: ", err)
		RunStatusChan <- map[string]string{"status": "failed", "message": err.Error()}
		return
	}
	defaultGateway = defaultGateway2

	if err := ConfigureTunInterface(defaultKey.Device); err != nil {
		logger.Error(fmt.Sprintf("配置 TUN 接口失败: %v", err))
		RunStatusChan <- map[string]string{"status": "failed", "message": err.Error()}
		return
	}

	// 等待设备就绪，最多等待10秒
	if err := waitForTunDeviceReady(defaultKey.Device, 10*time.Second); err != nil {
		logger.Error(fmt.Sprintf("等待 TUN 设备就绪失败: %v", err))
		RunStatusChan <- map[string]string{"status": "failed", "message": err.Error()}
		return
	}

	if err := ConfigureTunRoute(); err != nil {
		logger.Error(fmt.Sprintf("配置 TUN 路由失败: %v", err))
		RunStatusChan <- map[string]string{"status": "failed", "message": err.Error()}
		return
	}

	logger.Info("TUN 服务启动成功，设备名称: ", defaultKey.Interface)
	RunStatusChan <- map[string]string{"status": "success", "message": "TUN service started successfully"}

	signal.Notify(TunSignal, syscall.SIGINT, syscall.SIGTERM)
	<-TunSignal

	// 收到信号后调用 Stop
	stopTun()
}

func Stop() {
	TunSignal <- syscall.SIGTERM // 或其他自定义信号
}

func stopTun() {
	logger.Info("Stopping TUN service...")

	// 1. 停止 engine
	engine.Stop()

	// 2. 清理 TUN 路由
	if err := CleanupTunRoute(); err != nil {
		logger.Error("Failed to cleanup TUN route:", err)
	}

	// 3. 关闭 TUN 接口
	if err := CleanupTunInterface(defaultKey.Device); err != nil {
		logger.Error("Failed to cleanup TUN interface:", err)
	}
	// 4. 恢复默认网关
	if err := SetDefaultGateway(defaultGateway); err != nil {
		logger.Error("Failed to set default gateway:", err)
	}

	logger.Info("TUN service stopped successfully")
}

// CleanupTunRoute 清理 TUN 路由配置
func CleanupTunRoute() error {
	// 根据操作系统选择路由清理命令
	var routes [][]string
	switch runtime.GOOS {
	case "windows":
		routes = [][]string{
			{"route", "DELETE", "0.0.0.0", "MASK", "128.0.0.0", "10.0.0.1"},
			{"route", "DELETE", "128.0.0.0", "MASK", "128.0.0.0", "10.0.0.1"},
		}
	case "linux":
		routes = [][]string{
			{"ip", "route", "delete", "0.0.0.0/1", "via", "10.0.0.1"},
			{"ip", "route", "delete", "128.0.0.0/1", "via", "10.0.0.1"},
		}
	case "darwin": // macOS
		routes = [][]string{
			{"route", "-n", "delete", "-net", "1.0.0.0/8", "10.0.0.1"},
			{"route", "-n", "delete", "-net", "2.0.0.0/7", "10.0.0.1"},
			{"route", "-n", "delete", "-net", "4.0.0.0/6", "10.0.0.1"},
			{"route", "-n", "delete", "-net", "8.0.0.0/5", "10.0.0.1"},
			{"route", "-n", "delete", "-net", "32.0.0.0/3", "10.0.0.1"},
			{"route", "-n", "delete", "-net", "64.0.0.0/2", "10.0.0.1"},
			{"route", "-n", "delete", "-net", "128.0.0.0/1", "10.0.0.1"},
			{"route", "-n", "delete", "-net", "198.18.0.0/15", "10.0.0.1"},
		}
	default:
		return fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
	}

	// 清理路由
	for _, r := range routes {
		cmd := utils.GetRunCommand(r[0], r[1:]...)
		if err := cmd.Run(); err != nil {
			// 忽略路由不存在的错误
			logger.Info("Route deletion warning:", err)
			continue
		}
	}

	// 清理防火墙规则
	switch runtime.GOOS {
	case "darwin": // macOS
		// 清理 pf 规则
		cmd := utils.GetRunCommand("pfctl", "-f", "/dev/null")
		if err := cmd.Run(); err != nil {
			logger.Info("Failed to reset pf rules:", err)
			// 继续尝试禁用 pf
		}

		// 禁用 pf
		cmd = utils.GetRunCommand("pfctl", "-d")
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to disable pf: %w", err)
		}

	case "linux":
		// 清理 iptables 规则（假设使用 iptables）
		cmd := utils.GetRunCommand("iptables", "-F")
		if err := cmd.Run(); err != nil {
			logger.Info("Failed to flush iptables rules:", err)
		}
		// 可选：清理 nftables（如果使用 nftables）
		cmd = utils.GetRunCommand("nft", "flush", "ruleset")
		if err := cmd.Run(); err != nil {
			logger.Info("Failed to flush nftables rules:", err)
		}

	case "windows":
		// 清理 Windows 防火墙规则（假设规则名为 "TUNRule"）
		cmd := utils.GetRunCommand("powershell", "-Command", `Get-NetFirewallRule -DisplayName "TUNRule" | Remove-NetFirewallRule`)
		if err := cmd.Run(); err != nil {
			logger.Info("Failed to delete firewall rule:", err)
		}
	}

	return nil
}

// CleanupTunInterface 清理 TUN 接口，适配 Windows、Linux 和 macOS
func CleanupTunInterface(ifName string) error {
	switch runtime.GOOS {
	case "linux":
		// 关闭 TUN 接口
		cmd := utils.GetRunCommand("ip", "link", "set", ifName, "down")
		if err := cmd.Run(); err != nil {
			logger.Info("Failed to bring down interface:", err)
			// 继续尝试删除接口
		}

		// 删除 TUN 接口
		cmd = utils.GetRunCommand("ip", "link", "delete", ifName)
		if err := cmd.Run(); err != nil {
			logger.Info("Failed to delete interface:", err)
			return fmt.Errorf("failed to delete TUN interface %s: %w", ifName, err)
		}

	case "darwin": // macOS
		// 关闭 TUN 接口
		cmd := utils.GetRunCommand("ifconfig", ifName, "down")
		if err := cmd.Run(); err != nil {
			logger.Error(fmt.Sprintf("Failed to bring down interface: %v", err))
			// macOS 不支持直接删除 TUN 接口，继续执行
		}

		// macOS 的 TUN 接口通常由用户态程序管理，无法通过 ifconfig destroy 删除
		// 可选：尝试通过 route 命令清理关联路由
		cmd = utils.GetRunCommand("route", "-n", "flush")
		if err := cmd.Run(); err != nil {
			logger.Error(fmt.Sprintf("Failed to flush routes for interface: %v", err))
		}

	case "windows":
		// 禁用 TUN 接口
		cmd := utils.GetRunCommand("powershell", "-Command", `Disable-NetAdapter -Name "`+ifName+`" -Confirm:$false`)
		if err := cmd.Run(); err != nil {
			logger.Error(fmt.Sprintf("Failed to disable interface: %v", err))
			// 继续尝试删除接口
		}

		// Windows 的 TUN 接口（如 Wintun 或 TAP-Windows）通常由驱动管理
		// 删除接口需要通过设备管理或驱动工具，netsh 不直接支持
		// 这里仅记录警告，实际删除可能依赖外部工具（如 tapinstall）
		logger.Info("Windows TUN interface deletion requires driver-specific tools (e.g., tapinstall remove)")

	default:
		return fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
	}

	return nil
}

// GetDefaultGateway tries multiple methods to extract default gateway IP on macOS.
func GetDefaultGateway() (string, error) {
	methods := []func() (string, error){
		func() (string, error) {
			out, err := exec.Command("sh", "-c", "netstat -rn | grep '^default' | awk '{print $2}'").Output()
			if err != nil {
				return "", err
			}
			return strings.TrimSpace(string(out)), nil
		},
		func() (string, error) {
			return runAndTrim("ipconfig", "getoption", "en0", "router")
		},
		func() (string, error) {
			return runAndTrim("ipconfig", "getoption", "en1", "router")
		},
	}

	for _, m := range methods {
		if gw, err := m(); err == nil && gw != "" {
			return gw, nil
		}
	}

	return "", fmt.Errorf("default gateway not found by any method")
}

func SetDefaultGateway(gateway string) error {
	if runtime.GOOS == "darwin" {
		cmd := utils.GetRunCommand("route", "add", "default", gateway)
		return cmd.Run()
	}
	// if runtime.GOOS == "linux" {
	// 	cmd := utils.GetRunCommand("ip", "route", "change", "default", "via", gateway)
	// 	return cmd.Run()
	// }

	return nil
}

// runAndTrim runs a command and trims the output.
func runAndTrim(name string, args ...string) (string, error) {
	var out bytes.Buffer
	cmd := utils.GetRunCommand(name, args...)
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out.String()), nil
}

// monitorTunDevice 监控 TUN 设备状态
func monitorTunDevice(ifname string) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	var memStats runtime.MemStats
	for range ticker.C {
		// 检查设备状态
		if err := checkTunDeviceStatus(ifname); err != nil {
			logger.Warn(fmt.Sprintf("TUN 设备状态异常: %v", err))
		}

		// 获取内存统计
		runtime.ReadMemStats(&memStats)
		// 打印系统信息
		logger.Info(fmt.Sprintf("系统信息 - Goroutines: %d, 内存使用: %v MB",
			runtime.NumGoroutine(),
			memStats.Alloc/1024/1024))
	}
}

// checkTunDeviceStatus 检查 TUN 设备状态
func checkTunDeviceStatus(ifname string) error {
	switch runtime.GOOS {
	case "windows":
		return checkWindowsTunStatus(ifname)
	case "darwin":
		return checkDarwinTunStatus(ifname)
	case "linux":
		return checkLinuxTunStatus(ifname)
	default:
		return fmt.Errorf("不支持的操作系统: %s", runtime.GOOS)
	}
}

// checkWindowsTunStatus 检查 Windows TUN 设备状态
func checkWindowsTunStatus(ifname string) error {
	// 使用 netsh 检查接口状态
	cmd := utils.GetRunCommand("powershell", "-Command", `Get-NetIPInterface | Format-Table -AutoSize`)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("获取接口状态失败: %w", err)
	}
	logger.Info(fmt.Sprintf("checkWindowsTunStatus output: %s", string(output)))

	// 转换输出编码
	outputStr, err := convertGBKToUTF8(string(output))
	if err != nil {
		return fmt.Errorf("转换编码失败: %w", err)
	}

	// 查找 TUN 设备状态
	lines := strings.Split(outputStr, "\n")
	for _, line := range lines {
		if strings.Contains(line, ifname) {
			logger.Info(fmt.Sprintf("TUN 设备状态: %s", line))
			// 检查接口是否启用
			if strings.Contains(line, "已禁用") || strings.Contains(line, "disabled") {
				return fmt.Errorf("TUN 设备已禁用")
			}
			return nil
		}
	}

	return fmt.Errorf("未找到 TUN 设备: %s", ifname)
}

// checkDarwinTunStatus 检查 macOS TUN 设备状态
func checkDarwinTunStatus(ifname string) error {
	cmd := utils.GetRunCommand("ifconfig", ifname)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("获取接口状态失败: %w", err)
	}

	logger.Info(fmt.Sprintf("TUN 设备状态: %s", string(output)))
	return nil
}

// checkLinuxTunStatus 检查 Linux TUN 设备状态
func checkLinuxTunStatus(ifname string) error {
	cmd := utils.GetRunCommand("ip", "link", "show", ifname)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("获取接口状态失败: %w", err)
	}

	logger.Info("TUN 设备状态: %s", string(output))
	return nil
}

// waitForTunDeviceReady 等待TUN设备就绪
func waitForTunDeviceReady(deviceName string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		var cmd *exec.Cmd
		var checkString string

		// 根据操作系统选择命令
		switch runtime.GOOS {
		case "windows":
			// Windows 使用 netsh 检查接口状态
			cmd = utils.GetRunCommand("powershell", "-Command", "@(Get-NetAdapter | Select-Object Name, Status, InterfaceDescription, ifIndex) | ConvertTo-Json")
			checkString = "connected"
		case "linux": // darwin 是 macOS
			// Linux/macOS 使用 ip link show 检查接口状态
			cmd = utils.GetRunCommand("ip", "link", "show", deviceName)
			checkString = "UP"
		case "darwin": // macOS
			// macOS uses ifconfig to check interface status
			cmd = utils.GetRunCommand("ifconfig", deviceName)
			output, err := cmd.Output()
			if err != nil {
				// If the interface doesn't exist yet, ifconfig will return an error
				if strings.Contains(err.Error(), "can't find interface") {
					fmt.Printf("Interface %s not found yet (macOS), retrying...\n", deviceName)
					time.Sleep(1 * time.Second) // Wait and retry
					continue
				}
				return fmt.Errorf("failed to execute ifconfig command on macOS: %w", err)
			}

			outputStr := string(output)
			// Check for both UP and RUNNING flags in the output
			if strings.Contains(outputStr, "UP") && strings.Contains(outputStr, "RUNNING") {
				fmt.Printf("Interface %s status is good (macOS) - UP, RUNNING\n", deviceName)
				return nil
			} else {
				fmt.Printf("Interface %s not yet UP or RUNNING (macOS): %s\n", deviceName, outputStr)
			}

		default:
			return fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
		}

		// 执行命令获取接口状态
		output, err := cmd.CombinedOutput()
		if err != nil {
			logger.Error(fmt.Sprintf("检查接口状态失败: %v", err))
			time.Sleep(500 * time.Millisecond)
			continue
		}

		// 检查设备是否存在且状态为预期值
		outputStr := string(output)
		if runtime.GOOS == "windows" {
			// Windows 需要处理编码（假设 utils.AutoConvertEncoding 处理 GBK 到 UTF-8）
			outputStr, err = utils.AutoConvertEncoding(output)
			if err != nil {
				logger.Error(fmt.Sprintf("转换编码失败: %v", err))
				time.Sleep(2000 * time.Millisecond)
				continue
			}
		}

		// 检查输出是否包含设备名称和状态
		if strings.Contains(outputStr, deviceName) && strings.Contains(outputStr, checkString) {
			// 进一步验证网卡可用性，通过 ping 测试
			pingCmd := utils.GetRunCommand("ping", "-n", "1", "10.0.0.1")

			if err := pingCmd.Run(); err == nil {
				logger.Info("TUN 设备已就绪 OS: ", runtime.GOOS)
				time.Sleep(500 * time.Millisecond)
				return nil
			}
			logger.Info("Ping 测试失败，设备可能尚未完全就绪")
		}

		time.Sleep(500 * time.Millisecond)
	}

	return fmt.Errorf("等待 TUN 设备就绪超时")
}
