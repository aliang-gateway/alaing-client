package shadowsocks

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/netip"
	"strconv"
	"strings"
	"sync"

	"github.com/xjasonlyu/tun2socks/v2/transport/shadowsocks/core"

	"nursor.org/nursorgate/common/logger"
	"nursor.org/nursorgate/inbound/tun/dialer"
	M "nursor.org/nursorgate/inbound/tun/metadata"
	"nursor.org/nursorgate/outbound/proxy"
	"nursor.org/nursorgate/outbound/proxy/proto"
)

// Shadowsocks Shadowsocks 代理实现
// 基于 tun2socks 的核心加密库，提供改进的错误处理和稳定性
type Shadowsocks struct {
	*proxy.Base

	method   string
	password string
	username string
	cipher   core.Cipher
	obfsMode string
	obfsHost string

	mu sync.RWMutex
}

// ShadowsocksConfig Shadowsocks 配置选项
type ShadowsocksConfig struct {
	Server   string `json:"server"`
	Port     uint16 `json:"port"`
	Method   string `json:"method"`
	Password string `json:"password"`
	Username string `json:"username,omitempty"`
	ObfsMode string `json:"obfs_mode"`
	ObfsHost string `json:"obfs_host"`
}

// NewShadowsocks 创建 Shadowsocks 客户端
// 参数：server, method, password, username, obfsMode, obfsHost
func NewShadowsocks(server, method, password, username, obfsMode, obfsHost string) (*Shadowsocks, error) {
	// 解析服务器地址
	host, port := server, uint16(8388)
	if idx := strings.Index(server, ":"); idx != -1 {
		host = server[:idx]
		if p, err := strconv.ParseUint(server[idx+1:], 10, 16); err == nil {
			port = uint16(p)
		}
	}

	return NewShadowsocksWithConfig(&ShadowsocksConfig{
		Server:   host,
		Port:     port,
		Method:   method,
		Password: password,
		Username: username,
		ObfsMode: obfsMode,
		ObfsHost: obfsHost,
	})
}

// NewShadowsocksWithConfig 使用配置创建 Shadowsocks 客户端
func NewShadowsocksWithConfig(config *ShadowsocksConfig) (*Shadowsocks, error) {
	// 验证加密方式和密码
	if config.Method == "" {
		return nil, errors.New("shadowsocks method cannot be empty")
	}
	if config.Password == "" {
		return nil, errors.New("shadowsocks password cannot be empty")
	}

	// 创建密码对应的 Cipher
	cipher, err := core.PickCipher(config.Method, nil, config.Password)
	if err != nil {
		logger.Error(fmt.Sprintf("[Shadowsocks] ✗ 加密方式初始化失败 %s: %v", config.Method, err))
		return nil, fmt.Errorf("failed to initialize cipher: %w", err)
	}

	ss := &Shadowsocks{
		Base: &proxy.Base{
			Address:  fmt.Sprintf("%s:%d", config.Server, config.Port),
			Protocol: proto.Shadowsocks,
		},
		method:   config.Method,
		password: config.Password,
		username: config.Username,
		cipher:   cipher,
		obfsMode: config.ObfsMode,
		obfsHost: config.ObfsHost,
	}

	obfsInfo := ""
	if config.ObfsMode != "" {
		obfsInfo = fmt.Sprintf(", obfs: %s", config.ObfsMode)
	}

	userInfo := "-"
	if config.Username != "" {
		userInfo = config.Username
	}

	logger.Info(fmt.Sprintf("[Shadowsocks] ✓ 客户端已创建 服务器: %s:%d, 用户: %s, 加密方式: %s%s", config.Server, config.Port, userInfo, config.Method, obfsInfo))

	return ss, nil
}

// DialContext 实现 Proxy 接口的 DialContext 方法
func (ss *Shadowsocks) DialContext(ctx context.Context, metadata *M.Metadata) (c net.Conn, err error) {
	ss.mu.RLock()
	defer ss.mu.RUnlock()

	logger.Debug(fmt.Sprintf("[Shadowsocks] 连接到 Shadowsocks 服务器 %s, 目标: %s:%d", ss.Addr(), metadata.HostName, metadata.DstPort))

	// 连接到 Shadowsocks 服务器
	var connErr error
	c, connErr = dialer.DialContext(ctx, "tcp", ss.Addr())
	if connErr != nil {
		logger.Error(fmt.Sprintf("[Shadowsocks] ✗ 连接失败 %s: %v", ss.Addr(), connErr))
		return nil, fmt.Errorf("failed to connect to shadowsocks server: %w", connErr)
	}

	// 设置 KeepAlive
	proxy.SetKeepAlive(c)

	// 延迟清理连接（如果出错）
	defer func(c net.Conn) {
		proxy.SafeConnClose(c, err)
	}(c)

	// 使用加密包装连接
	c = ss.cipher.StreamConn(c)

	// 构建 SOCKS5 地址
	socksAddr := proxy.SerializeSocksAddr(metadata)

	// 调试日志：显示metadata和SOCKS5地址详情
	logger.Debug(fmt.Sprintf("[Shadowsocks] DEBUG - metadata.DstIP: %v (valid: %v), DstPort: %d, HostName: %s",
		metadata.DstIP, metadata.DstIP.IsValid(), metadata.DstPort, metadata.HostName))
	logger.Debug(fmt.Sprintf("[Shadowsocks] DEBUG - socksAddr bytes: %v (len: %d)", socksAddr, len(socksAddr)))

	// 发送目标地址信息到服务器
	n, err := c.Write(socksAddr)
	logger.Debug(fmt.Sprintf("[Shadowsocks] DEBUG - Write result: n=%d, err=%v", n, err))

	if err != nil {
		logger.Error(fmt.Sprintf("[Shadowsocks] ✗ 发送地址信息失败: %v", err))
		return nil, fmt.Errorf("failed to write socks addr: %w", err)
	}

	userInfo := "-"
	if ss.username != "" {
		userInfo = ss.username
	}

	logger.Info(fmt.Sprintf("[Shadowsocks] ✓ TCP连接成功 用户: %s, 目标: %s:%d", userInfo, metadata.HostName, metadata.DstPort))

	return c, err
}

// DialUDP 实现 Proxy 接口的 DialUDP 方法
func (ss *Shadowsocks) DialUDP(metadata *M.Metadata) (net.PacketConn, error) {
	ss.mu.RLock()
	defer ss.mu.RUnlock()

	logger.Debug(fmt.Sprintf("[Shadowsocks] 创建 UDP 连接到 %s, 目标: %s:%d", ss.Addr(), metadata.HostName, metadata.DstPort))

	// 创建本地 UDP 套接字
	pc, err := dialer.ListenPacket("udp", "")
	if err != nil {
		logger.Error(fmt.Sprintf("[Shadowsocks] ✗ UDP 监听失败: %v", err))
		return nil, fmt.Errorf("failed to listen packet: %w", err)
	}

	// 解析服务器地址
	udpAddr, err := net.ResolveUDPAddr("udp", ss.Addr())
	if err != nil {
		pc.Close()
		logger.Error(fmt.Sprintf("[Shadowsocks] ✗ 解析 UDP 地址失败 %s: %v", ss.Addr(), err))
		return nil, fmt.Errorf("resolve udp address %s: %w", ss.Addr(), err)
	}

	// 使用加密包装 UDP 连接
	pc = ss.cipher.PacketConn(pc)

	logger.Info(fmt.Sprintf("[Shadowsocks] ✓ UDP连接成功 目标: %s:%d", metadata.HostName, metadata.DstPort))

	return &ssPacketConn{
		PacketConn: pc,
		rAddr:      udpAddr,
		metadata:   metadata,
	}, nil
}

// ssPacketConn 包装 UDP 连接，处理 SOCKS5 地址格式
type ssPacketConn struct {
	net.PacketConn
	rAddr    net.Addr
	metadata *M.Metadata
}

// WriteTo 实现 PacketConn 接口
func (pc *ssPacketConn) WriteTo(b []byte, addr net.Addr) (n int, err error) {
	logger.Debug(fmt.Sprintf("[Shadowsocks UDP] 写入数据到 %v", addr))

	// 构建 SOCKS5 格式的 UDP 数据包
	var packet []byte

	// 如果是 M.Addr 类型，使用元数据构建地址
	if ma, ok := addr.(*M.Addr); ok {
		socksAddr := proxy.SerializeSocksAddr(ma.Metadata())
		packet = make([]byte, len(socksAddr)+len(b))
		copy(packet, socksAddr)
		copy(packet[len(socksAddr):], b)
	} else if udpAddr, ok := addr.(*net.UDPAddr); ok {
		// 将 net.IP 转换为 netip.Addr
		ip, err := netip.ParseAddr(udpAddr.IP.String())
		if err != nil {
			ip = netip.IPv4Unspecified()
		}
		socksAddr := proxy.SerializeSocksAddr(&M.Metadata{
			DstIP:    ip,
			DstPort:  uint16(udpAddr.Port),
			HostName: "",
		})
		packet = make([]byte, len(socksAddr)+len(b))
		copy(packet, socksAddr)
		copy(packet[len(socksAddr):], b)
	} else {
		// 默认情况，直接发送
		packet = b
	}

	// 写入加密后的数据到服务器
	return pc.PacketConn.WriteTo(packet, pc.rAddr)
}

// ReadFrom 实现 PacketConn 接口
func (pc *ssPacketConn) ReadFrom(b []byte) (n int, addr net.Addr, err error) {
	n, _, err = pc.PacketConn.ReadFrom(b)
	if err != nil {
		return 0, nil, err
	}

	logger.Debug(fmt.Sprintf("[Shadowsocks UDP] 读取数据 %d 字节", n))

	// 返回原始请求的目标地址
	if pc.metadata != nil {
		// 返回正确的 net.Addr 类型
		ip := pc.metadata.DstIP
		port := pc.metadata.DstPort
		return n, &net.UDPAddr{
			IP:   net.ParseIP(ip.String()),
			Port: int(port),
		}, nil
	}

	return n, pc.rAddr, nil
}
