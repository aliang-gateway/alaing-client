package utils

import (
	"fmt"
	"net"
	"runtime"
	"strings"

	"nursor.org/nursorgate/client/utils"
)

// GetDefaultInterface 获取系统默认的网络接口
func GetDefaultInterface() (string, error) {
	switch runtime.GOOS {
	case "windows":
		return getWindowsDefaultInterface()
	case "darwin":
		return getDarwinDefaultInterface()
	case "linux":
		return getLinuxDefaultInterface()
	default:
		return "", fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
	}
}

// getWindowsDefaultInterface 获取 Windows 默认网络接口
func getWindowsDefaultInterface() (string, error) {
	// 使用 netsh 命令获取默认路由接口
	cmd := utils.GetRunCommand("netsh", "interface", "ipv4", "show", "route")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to get routes: %w", err)
	}

	// 自动检测并转换编码
	outputStr, err := utils.AutoConvertEncoding(output)
	if err != nil {
		return "", fmt.Errorf("failed to convert encoding: %w", err)
	}

	// 解析输出找到默认路由（0.0.0.0/0）
	lines := strings.Split(outputStr, "\n")
	var defaultRouteIndex string
	for _, line := range lines {
		if strings.Contains(line, "0.0.0.0/0") {
			fields := strings.Fields(line)
			if len(fields) >= 5 {
				// 在 Windows 输出中，索引是第 5 个字段
				defaultRouteIndex = fields[4]
				break
			}
		}
	}

	if defaultRouteIndex == "" {
		return "", fmt.Errorf("no default route found")
	}

	// 获取所有网络接口的状态
	cmd = utils.GetRunCommand("netsh", "interface", "ipv4", "show", "interfaces")
	output, err = cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to get interfaces: %w", err)
	}

	// 自动检测并转换接口信息编码
	outputStr, err = utils.AutoConvertEncoding(output)
	if err != nil {
		return "", fmt.Errorf("failed to convert encoding: %w", err)
	}

	// 解析接口信息
	lines = strings.Split(outputStr, "\n")
	var connectedInterfaces []string
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) >= 5 {
			// 检查是否是目标索引且状态为 connected
			if fields[0] == defaultRouteIndex && fields[3] == "connected" {
				// 接口名称是最后一个字段
				interfaceName := fields[len(fields)-1]
				// 排除虚拟接口和回环接口
				if !strings.Contains(strings.ToLower(interfaceName), "loopback") &&
					!strings.Contains(strings.ToLower(interfaceName), "pseudo") &&
					!strings.Contains(strings.ToLower(interfaceName), "virtual") &&
					!strings.Contains(strings.ToLower(interfaceName), "vethernet") {
					connectedInterfaces = append(connectedInterfaces, interfaceName)
				}
			}
		}
	}

	// 如果找到了连接的接口，返回第一个
	if len(connectedInterfaces) > 0 {
		return connectedInterfaces[0], nil
	}

	// 如果没有找到合适的接口，尝试获取所有活动的网络接口
	interfaces, err := net.Interfaces()
	if err != nil {
		return "", fmt.Errorf("failed to get interfaces: %w", err)
	}

	for _, iface := range interfaces {
		// 检查接口是否启用且不是回环接口
		if iface.Flags&net.FlagUp != 0 && iface.Flags&net.FlagLoopback == 0 {
			// 尝试获取接口的 IP 地址
			addrs, err := iface.Addrs()
			if err != nil {
				continue
			}
			for _, addr := range addrs {
				// 检查是否是 IPv4 地址
				if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() && ipnet.IP.To4() != nil {
					// 排除虚拟接口
					if !strings.Contains(strings.ToLower(iface.Name), "virtual") &&
						!strings.Contains(strings.ToLower(iface.Name), "vethernet") {
						return iface.Name, nil
					}
				}
			}
		}
	}

	return "", fmt.Errorf("no suitable network interface found")
}

// getDarwinDefaultInterface 获取 macOS 默认网络接口
func getDarwinDefaultInterface() (string, error) {
	// 使用 route 命令获取默认路由接口
	cmd := utils.GetRunCommand("route", "-n", "get", "default")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to get default route: %w", err)
	}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "interface:") {
			fields := strings.Fields(line)
			if len(fields) > 1 {
				return fields[len(fields)-1], nil
			}
		}
	}

	// 如果没有找到默认路由，尝试获取活动的网络接口
	interfaces, err := net.Interfaces()
	if err != nil {
		return "", fmt.Errorf("failed to get interfaces: %w", err)
	}

	for _, iface := range interfaces {
		// 检查接口是否启用且不是回环接口
		if iface.Flags&net.FlagUp != 0 && iface.Flags&net.FlagLoopback == 0 {
			// 尝试获取接口的 IP 地址
			addrs, err := iface.Addrs()
			if err != nil {
				continue
			}
			for _, addr := range addrs {
				// 检查是否是 IPv4 地址
				if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() && ipnet.IP.To4() != nil {
					return iface.Name, nil
				}
			}
		}
	}

	return "", fmt.Errorf("no suitable network interface found")
}

// getLinuxDefaultInterface 获取 Linux 默认网络接口
func getLinuxDefaultInterface() (string, error) {
	// 使用 ip 命令获取默认路由接口
	cmd := utils.GetRunCommand("ip", "route", "get", "8.8.8.8")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to get default route: %w", err)
	}

	// 解析输出找到接口名称
	fields := strings.Fields(string(output))
	for i, field := range fields {
		if field == "dev" && i+1 < len(fields) {
			return fields[i+1], nil
		}
	}

	// 如果没有找到默认路由，尝试获取活动的网络接口
	interfaces, err := net.Interfaces()
	if err != nil {
		return "", fmt.Errorf("failed to get interfaces: %w", err)
	}

	for _, iface := range interfaces {
		// 检查接口是否启用且不是回环接口
		if iface.Flags&net.FlagUp != 0 && iface.Flags&net.FlagLoopback == 0 {
			// 尝试获取接口的 IP 地址
			addrs, err := iface.Addrs()
			if err != nil {
				continue
			}
			for _, addr := range addrs {
				// 检查是否是 IPv4 地址
				if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() && ipnet.IP.To4() != nil {
					return iface.Name, nil
				}
			}
		}
	}

	return "", fmt.Errorf("no suitable network interface found")
}
