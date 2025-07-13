package tun

import (
	"fmt"
	"regexp"
	"runtime"
	"strconv"
	"strings"

	"nursor.org/nursorgate/client/server/tun/engine"
	"nursor.org/nursorgate/client/server/tun/utils"
	utils2 "nursor.org/nursorgate/client/utils"
	"nursor.org/nursorgate/common/logger"
)

func InitArgs() engine.Key {
	// 获取默认网络接口
	defaultInterface, err := utils.GetDefaultInterface()
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to get default interface: %v", err))
		defaultInterface = "en0" // 设置一个默认值
	}

	defaultKey := engine.Key{
		//Proxy: "http://clash:asd123456@172.16.1.1:7890",
		Proxy:       "direct://",
		NursorProxy: "https://ai-gateway.nursor.org:8888",
		MTU:         0,
		Mark:        0,
		Device:      getDefaultTunName(),
		Interface:   defaultInterface,
	}
	return defaultKey
}

func getAvailableUtunDevice() string {
	// 使用 ifconfig 命令列出所有网络接口
	cmd := utils2.GetRunCommand("ifconfig")
	output, err := cmd.Output()
	if err != nil {
		// 如果执行命令失败，返回一个默认值并打印错误（或记录日志）
		fmt.Printf("Error running ifconfig: %v\n", err)
		return "utun99" // Fallback default
	}

	existingUtuns := make(map[int]bool)
	// 使用正则表达式匹配 utunX 接口
	re := regexp.MustCompile(`utun(\d+):`)
	matches := re.FindAllStringSubmatch(string(output), -1)

	for _, match := range matches {
		if len(match) > 1 {
			if num, err := strconv.Atoi(match[1]); err == nil {
				existingUtuns[num] = true
			}
		}
	}

	// 从0开始查找第一个可用的utun序列号
	for i := 0; i < 100; i++ { // 尝试最多100个，防止无限循环
		if !existingUtuns[i] {
			return fmt.Sprintf("utun%d", i)
		}
	}

	// 如果所有utun0-99都被占用，返回一个高序号的默认值
	fmt.Println("Warning: All utun0-99 devices are in use. Returning utun999.")
	return "utun999"
}

// getAvailableWintunDevice 查找可用的 Wintun 设备名称，例如 Wintun, Wintun1, ...
// Windows下没有一个通用的命令行工具像ifconfig那样列出所有虚拟网卡。
// Wintun设备通常在适配器列表中显示，但名称可能不完全是"WintunX"。
// 最可靠的方法通常是查询注册表或使用PowerShell。
// 这里的实现是一个简化的示例，可能需要根据Wintun库的具体行为调整。
// 理想情况下，Wintun库本身提供创建唯一名称的API。
func getAvailableWintunDevice() string {
	// 在Windows上，我们可以尝试使用Get-NetAdapter来查看网络适配器，
	// 但其名称可能不规律。这里假设你的Wintun设备会包含"Wintun"字样。
	// 更精确的检查需要Wintun API或Windows网络适配器管理API。
	//
	// 这是一个尝试性的PowerShell命令，可能需要管理员权限才能运行
	// 且输出格式需要仔细解析。
	cmd := utils2.GetRunCommand("powershell.exe", "-Command", "Get-NetAdapter | Select-Object -ExpandProperty Name")
	output, err := cmd.Output()
	if err != nil {
		fmt.Printf("Error running Get-NetAdapter PowerShell command: %v\n", err)
		return "Wintun" // Fallback default
	}

	existingAdapters := strings.Split(string(output), "\n")
	existingWintuns := make(map[int]bool)

	// 匹配 Wintun 或 WintunX 这样的名称
	// 注意：Wintun库可能默认只创建一个"Wintun"接口，后续创建可能需要显式名称。
	// 这里假设可以有Wintun1, Wintun2等
	re := regexp.MustCompile(`Wintun(\d*)`) // 匹配 "Wintun" 或 "WintunN"

	for _, adapterName := range existingAdapters {
		match := re.FindStringSubmatch(strings.TrimSpace(adapterName))
		if len(match) > 0 {
			if match[1] == "" { // "Wintun" (没有数字后缀)
				existingWintuns[0] = true
			} else if num, err := strconv.Atoi(match[1]); err == nil {
				existingWintuns[num] = true
			}
		}
	}

	// 从0开始查找第一个可用的Wintun序列号
	for i := 0; i < 100; i++ {
		if !existingWintuns[i] {
			if i == 0 {
				return "Wintun" // 如果 Wintun0 可用，返回 "Wintun"
			}
			return fmt.Sprintf("Wintun%d", i)
		}
	}

	fmt.Println("Warning: All Wintun0-99 devices are in use. Returning Wintun999.")
	return "Wintun999"
}

// getDefaultTunName 根据操作系统返回默认的 TUN 设备名称
func getDefaultTunName() string {
	switch runtime.GOOS {
	case "darwin":
		return getAvailableUtunDevice()
	case "linux":
		return "tun0"
	case "windows":
		return getAvailableWintunDevice()
	default:
		return "tun0"
	}
}
