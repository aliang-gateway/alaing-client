package tun

import (
	"fmt"
	"net"
	"runtime"
	"strings"

	"bytes"
	"io"

	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
	"nursor.org/nursorgate/client/utils"
	"nursor.org/nursorgate/common/logger"
)

// convertGBKToUTF8 将 GBK 编码转换为 UTF-8
func convertGBKToUTF8(s string) (string, error) {
	reader := transform.NewReader(bytes.NewReader([]byte(s)), simplifiedchinese.GBK.NewDecoder())
	d, err := io.ReadAll(reader)
	if err != nil {
		return "", err
	}
	return string(d), nil
}

func ConfigureTunInterface(ifname string) error {
	logger.Info("[INFO] Configuring TUN interface on %s", runtime.GOOS)
	switch runtime.GOOS {
	case "windows":
		return configureWindowsTunInterface(ifname)
	case "darwin":
		return configureDarwinTunInterface(ifname)
	case "linux":
		return configureLinuxTunInterface(ifname)
	default:
		return fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
	}
}

func ConfigureTunRoute() error {
	logger.Info(fmt.Sprintf("[INFO] Configuring TUN routes on %s", runtime.GOOS))
	switch runtime.GOOS {
	case "windows":
		return configureWindowsTunRoute()
	case "darwin":
		return configureDarwinTunRoute()
	case "linux":
		return configureLinuxTunRoute()
	default:
		return fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
	}
}

func configureWindowsTunInterface(ifname string) error {
	// 使用 netsh 命令配置接口
	commands := [][]string{
		{"netsh", "interface", "ipv4", "set", "address", "name=" + ifname, "static", "10.0.0.1", "255.255.255.0"},
		{"netsh", "interface", "ipv4", "set", "interface", "name=" + ifname, "metric=1"},
		{"netsh", "interface", "ipv4", "set", "interface", "name=" + ifname, "admin=enabled"},
	}

	for _, cmd := range commands {
		if err := utils.RunCommand(cmd[0], cmd[1:]...); err != nil {
			return fmt.Errorf("netsh command failed: %w", err)
		}
	}
	return nil
}

func configureDarwinTunInterface(ifname string) error {
	if err := utils.RunCommand("ifconfig", ifname, "10.0.0.1", "10.0.0.2", "up"); err != nil {
		return fmt.Errorf("ifconfig failed: %w", err)
	}
	return nil
}

func configureLinuxTunInterface(ifname string) error {
	commands := [][]string{
		{"ip", "addr", "add", "10.0.0.1/24", "dev", ifname},
		{"ip", "link", "set", "dev", ifname, "up"},
		{"ip", "route", "add", "10.0.0.0/24", "dev", ifname},
	}

	for _, cmd := range commands {
		if err := utils.RunCommand(cmd[0], cmd[1:]...); err != nil {
			return fmt.Errorf("ip command failed: %w", err)
		}
	}
	return nil
}

func configureWindowsTunRoute() error {
	// 保存当前默认路由
	cmd := utils.GetRunCommand("netsh", "interface", "ipv4", "show", "route")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to get current routes: %w", err)
	}

	// 转换输出编码
	//outputStr, err := convertGBKToUTF8(string(output))
	outputStr, _ := utils.AutoConvertEncoding(output)
	if err != nil {
		return fmt.Errorf("failed to convert encoding: %w", err)
	}

	// 解析输出找到默认路由
	lines := strings.Split(outputStr, "\n")
	var defaultGateway string
	var defaultRouteMetric int = 999999 // 设置一个较大的初始值

	// 跳过表头
	startParsing := false
	for _, line := range lines {
		// 跳过空行
		if strings.TrimSpace(line) == "" {
			continue
		}

		// 找到表头后的分隔线
		if strings.Contains(line, "-------") {
			startParsing = true
			continue
		}

		// 开始解析路由表
		if startParsing {
			fields := strings.Fields(line)
			if len(fields) >= 6 {
				// 检查是否是默认路由 (0.0.0.0/0)
				if fields[3] == "0.0.0.0/0" {
					// 解析跃点数
					metric := 0
					fmt.Sscanf(fields[2], "%d", &metric)

					// 选择跃点数最小的路由作为默认路由
					if metric < defaultRouteMetric {
						defaultRouteMetric = metric
						defaultGateway = fields[5] // 网关/接口名称在最后一列
					}
				}
			}
		}
	}

	// 如果没有找到默认路由，尝试使用 ipconfig 命令
	if defaultGateway == "" {
		cmd = utils.GetRunCommand("ipconfig")
		output, err = cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("failed to get ipconfig: %w", err)
		}

		outputStr, err = convertGBKToUTF8(string(output))
		if err != nil {
			return fmt.Errorf("failed to convert ipconfig encoding: %w", err)
		}

		// 查找默认网关
		lines = strings.Split(outputStr, "\n")
		for _, line := range lines {
			if strings.Contains(line, "默认网关") || strings.Contains(line, "Default Gateway") {
				fields := strings.Fields(line)
				for i, field := range fields {
					if field == ":" && i+1 < len(fields) {
						defaultGateway = fields[i+1]
						break
					}
				}
			}
		}
	}

	if defaultGateway == "" {
		// 如果仍然找不到默认网关，尝试使用 route print 命令
		cmd = utils.GetRunCommand("route", "print")
		output, err = cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("failed to get route print: %w", err)
		}

		outputStr, err = convertGBKToUTF8(string(output))
		if err != nil {
			return fmt.Errorf("failed to convert route print encoding: %w", err)
		}

		lines = strings.Split(outputStr, "\n")
		for _, line := range lines {
			if strings.Contains(line, "0.0.0.0") && strings.Contains(line, "0.0.0.0") {
				fields := strings.Fields(line)
				if len(fields) >= 4 {
					defaultGateway = fields[2]
					break
				}
			}
		}
	}

	if defaultGateway == "" {
		return fmt.Errorf("无法找到默认网关，请检查网络连接")
	}

	logger.Info("找到默认网关: %s (跃点数: %d)", defaultGateway, defaultRouteMetric)

	// 删除现有默认路由
	commands := [][]string{
		{"route", "delete", "0.0.0.0"},
		{"route", "delete", "128.0.0.0"},
	}

	for _, cmd := range commands {
		if err := utils.RunCommand(cmd[0], cmd[1:]...); err != nil {
			logger.Error("删除路由失败: %v", err)
		}
	}

	// 添加新的路由
	commands = [][]string{
		{"route", "add", "0.0.0.0", "mask", "128.0.0.0", "10.0.0.1", "metric", "1"},
		{"route", "add", "128.0.0.0", "mask", "128.0.0.0", "10.0.0.1", "metric", "1"},
		// 添加回原默认网关的路由，但优先级较低
		{"route", "add", "0.0.0.0", "mask", "0.0.0.0", defaultGateway, "metric", "2"},
	}

	for _, cmd := range commands {
		if err := utils.RunCommand(cmd[0], cmd[1:]...); err != nil {
			return fmt.Errorf("添加路由失败: %w", err)
		}
	}

	return nil
}

func configureDarwinTunRoute() error {
	// nursorRouter := model.NewAllowProxyDomain()
	// allowdToGateUrls := nursorRouter.ToGateDomains

	var routes [][]string
	hasFakeIP := false
	// for _, domain := range allowdToGateUrls {
	// 	domains := strings.Split(domain, ":")
	// 	ips, err := net.LookupIP(domains[0])
	// 	if err != nil {
	// 		fmt.Printf("Failed to resolve domain %s: %v\n", domain, err)
	// 		continue
	// 	}

	// 	for _, ip := range ips {
	// 		if ipv4 := ip.To4(); ipv4 != nil {
	// 			if isFakeIP(ipv4) {
	// 				hasFakeIP = true
	// 			}
	// 			// 添加路由，仅针对 fakeip
	// 			routes = append(routes, []string{
	// 				"route", "-n", "add", "-host", ipv4.String(), "10.0.0.1",
	// 			})
	// 		}
	// 	}
	// }

	// 如果没有匹配到 fakeip，就 fallback 用默认劫持
	if !hasFakeIP {
		routes = [][]string{
			{"route", "-n", "add", "-net", "1.0.0.0/8", "10.0.0.1"},
			{"route", "-n", "add", "-net", "2.0.0.0/7", "10.0.0.1"},
			{"route", "-n", "add", "-net", "4.0.0.0/6", "10.0.0.1"},
			{"route", "-n", "add", "-net", "8.0.0.0/5", "10.0.0.1"},
			{"route", "-n", "add", "-net", "32.0.0.0/3", "10.0.0.1"},
			{"route", "-n", "add", "-net", "64.0.0.0/2", "10.0.0.1"},
			{"route", "-n", "add", "-net", "128.0.0.0/1", "10.0.0.1"},
			{"route", "-n", "add", "-net", "198.18.0.0/15", "10.0.0.1"},
		}
	}

	// 执行路由命令
	for _, r := range routes {
		if err := utils.RunCommand(r[0], r[1:]...); err != nil {
			return fmt.Errorf("route add failed: %w", err)
		}
	}
	return nil
}

// 判断是否是 198.18.x.x/15
func isFakeIP(ip net.IP) bool {
	return ip[0] == 198 && (ip[1] == 18 || ip[1] == 19)
}

func configureLinuxTunRoute() error {
	// 保存当前默认路由
	cmd := utils.GetRunCommand("ip", "route", "show", "default")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to get current routes: %w", err)
	}

	// 解析输出找到默认路由
	lines := strings.Split(string(output), "\n")
	var defaultGateway string
	for _, line := range lines {
		if strings.Contains(line, "default via") {
			fields := strings.Fields(line)
			for i, field := range fields {
				if field == "via" && i+1 < len(fields) {
					defaultGateway = fields[i+1]
					break
				}
			}
		}
	}

	if defaultGateway == "" {
		return fmt.Errorf("no default gateway found")
	}

	// 删除现有默认路由
	commands := [][]string{
		{"ip", "route", "del", "default"},
	}

	for _, cmd := range commands {
		if err := utils.RunCommand(cmd[0], cmd[1:]...); err != nil {
			logger.Error("Failed to delete route: %v", err)
		}
	}

	// 添加新的路由
	commands = [][]string{
		{"ip", "route", "add", "0.0.0.0/1", "via", "10.0.0.1", "metric", "1"},
		{"ip", "route", "add", "128.0.0.0/1", "via", "10.0.0.1", "metric", "1"},
		// 添加回原默认网关的路由，但优先级较低
		{"ip", "route", "add", "default", "via", defaultGateway, "metric", "2"},
	}

	for _, cmd := range commands {
		if err := utils.RunCommand(cmd[0], cmd[1:]...); err != nil {
			return fmt.Errorf("ip route add failed: %w", err)
		}
	}

	return nil
}
