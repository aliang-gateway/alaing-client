package vless

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"math/rand"
	"net"
	"strconv"
	"strings"
	"sync"

	stls "github.com/sagernet/sing-box/common/tls"
	sopt "github.com/sagernet/sing-box/option"
	vlessSingBox "github.com/sagernet/sing-vmess/vless"
	"github.com/sagernet/sing/common"
	"github.com/sagernet/sing/common/json/badoption"
	smeta "github.com/sagernet/sing/common/metadata"
	"nursor.org/nursorgate/common/logger"
	"nursor.org/nursorgate/inbound/tun/dialer"
	M "nursor.org/nursorgate/inbound/tun/metadata"
	"nursor.org/nursorgate/outbound/proxy"
	"nursor.org/nursorgate/outbound/proxy/proto"
)

// VLESS 使用简化的 VLESS 实现，参考 xray-core 设计
type VLESS struct {
	*proxy.Base
	server  string
	uuid    string
	sni     string
	flow    string
	reality *RealityConfig
	client  *vlessSingBox.Client

	mu sync.RWMutex
}

// RealityConfig REALITY 配置
type RealityConfig struct {
	Enabled   bool   `json:"enabled"`
	PublicKey string `json:"public_key"`
	ShortID   string `json:"short_id"`
}

// VLESSConfig VLESS 配置选项
type VLESSConfig struct {
	Server     string         `json:"server"`
	ServerPort uint16         `json:"server_port"`
	UUID       string         `json:"uuid"`
	Flow       string         `json:"flow"`
	TLS        *TLSConfig     `json:"tls,omitempty"`
	Reality    *RealityConfig `json:"reality,omitempty"`
	PublicKey  string         `json:"public_key"`
	ShortID    string         `json:"short_id"`
}

// TLSConfig TLS 配置
type TLSConfig struct {
	Enabled    bool           `json:"enabled"`
	ServerName string         `json:"server_name"`
	UTLS       *UTLSConfig    `json:"utls,omitempty"`
	Reality    *RealityConfig `json:"reality,omitempty"`
}

// UTLSConfig uTLS 配置
type UTLSConfig struct {
	Enabled     bool   `json:"enabled"`
	Fingerprint string `json:"fingerprint"`
}

// NewVLESS 创建基础 VLESS 客户端
func NewVLESS(server, uuid string) (*VLESS, error) {
	// 解析服务器地址
	host, port := server, uint16(443)
	if idx := strings.Index(server, ":"); idx != -1 {
		host = server[:idx]
		if p, err := strconv.ParseUint(server[idx+1:], 10, 16); err == nil {
			port = uint16(p)
		}
	}

	return NewVLESSWithConfig(&VLESSConfig{
		Server:     host,
		ServerPort: port,
		UUID:       uuid,
	})
}

// NewVLESSWithTLS 创建带 TLS 的 VLESS 客户端
func NewVLESSWithTLS(server, uuid, sni string) (*VLESS, error) {
	// 解析服务器地址
	host, port := server, uint16(443)
	if idx := strings.Index(server, ":"); idx != -1 {
		host = server[:idx]
		if p, err := strconv.ParseUint(server[idx+1:], 10, 16); err == nil {
			port = uint16(p)
		}
	}

	return NewVLESSWithConfig(&VLESSConfig{
		Server:     host,
		ServerPort: port,
		UUID:       uuid,
		TLS: &TLSConfig{
			Enabled:    true,
			ServerName: sni,
		},
	})
}

// NewVLESSWithVision 创建带 Vision 流的 VLESS 客户端
func NewVLESSWithVision(server, uuid, sni string) (*VLESS, error) {
	// 解析服务器地址
	host, port := server, uint16(443)
	if idx := strings.Index(server, ":"); idx != -1 {
		host = server[:idx]
		if p, err := strconv.ParseUint(server[idx+1:], 10, 16); err == nil {
			port = uint16(p)
		}
	}

	return NewVLESSWithConfig(&VLESSConfig{
		Server:     host,
		ServerPort: port,
		UUID:       uuid,
		Flow:       "xtls-rprx-vision",
		TLS: &TLSConfig{
			Enabled:    true,
			ServerName: sni,
		},
	})
}

// NewVLESSWithReality 创建带 REALITY 的 VLESS 客户端
func NewVLESSWithReality(server, uuid, sni, publicKey string) (*VLESS, error) {
	return NewVLESSWithRealityAndShortID(server, uuid, sni, publicKey, "")
}

// NewVLESSWithRealityAndShortID 创建带 REALITY 和指定 ShortID 的 VLESS 客户端
func NewVLESSWithRealityAndShortID(server, uuid, sni, publicKey, shortID string) (*VLESS, error) {
	// 如果没有提供 shortID，从默认列表随机选择
	if shortID == "" {
		shortIDStr := "ef,b79e62,7d87a3,f4bfb2,ecdc,048cc1,be,872a9cb601,4e642a,d0a4cc,6a37c85b4d,facf,e2e46bb5,5fe83d984b7c,884c,f2e4c3af,7b79c5,b7a05d,b6920fa248,0975,95,4d3bd40917,57d89cd6ed9a"
		shortIDSArray := strings.Split(shortIDStr, ",")
		shortID = shortIDSArray[rand.Intn(len(shortIDSArray))]
	}
	// 解析服务器地址
	host, port := server, uint16(443)
	if idx := strings.Index(server, ":"); idx != -1 {
		host = server[:idx]
		if p, err := strconv.ParseUint(server[idx+1:], 10, 16); err == nil {
			port = uint16(p)
		}
	}

	return NewVLESSWithConfig(&VLESSConfig{
		Server:     host,
		ServerPort: port,
		UUID:       uuid,
		Flow:       "xtls-rprx-vision",
		TLS: &TLSConfig{
			Enabled:    true,
			ServerName: sni,
			Reality: &RealityConfig{
				Enabled:   true,
				PublicKey: publicKey,
				ShortID:   shortID,
			},
		},
	})
}

// NewVLESSWithConfig 使用配置创建 VLESS 客户端，参考 xray-core 初始化
func NewVLESSWithConfig(config *VLESSConfig) (*VLESS, error) {

	// 使用 SingBox 兼容的 logger
	singBoxLogger := logger.GetSingBoxLogger()
	client, err := vlessSingBox.NewClient(config.UUID, config.Flow, singBoxLogger)
	if err != nil {
		return nil, fmt.Errorf("failed to create vless client: %w", err)
	}

	v := &VLESS{
		Base: &proxy.Base{
			Address:  fmt.Sprintf("%s:%d", config.Server, config.ServerPort),
			Protocol: proto.VLESS,
		},
		server: config.Server,
		uuid:   config.UUID,
		//uuidBytes: parsedUUID[:], // 转换为字节数组
		client: client,
	}

	if config.TLS != nil && config.TLS.Enabled {
		v.sni = config.TLS.ServerName
		if config.TLS.Reality != nil {
			v.reality = config.TLS.Reality
		}
	}
	v.flow = config.Flow

	return v, nil
}

// DialContext 实现 Proxy 接口的 DialContext 方法，参考 xray-core 连接复用机制
func (v *VLESS) DialContext(ctx context.Context, metadata *M.Metadata) (net.Conn, error) {
	v.mu.RLock()
	defer v.mu.RUnlock()

	conn, err := v.establishNewConnection(ctx, metadata)
	if err != nil {
		return nil, err
	}

	return conn, nil
}

// establishNewConnection 建立新连接并准备握手数据，参考 xray-core 连接建立流程
func (v *VLESS) establishNewConnection(ctx context.Context, metadata *M.Metadata) (net.Conn, error) {
	// 连接到 VLESS 服务器，使用默认 dialer
	logger.Debug(fmt.Sprintf("DEBUG: 连接到 VLESS 服务器 %s\n", v.Addr()))
	conn, err := dialer.DialContext(ctx, "tcp", v.Addr())
	if err != nil {
		return nil, fmt.Errorf("failed to connect to VLESS server: %w", err)
	}

	// 如果启用了 REALITY，进行 REALITY 握手
	if v.reality != nil && v.reality.Enabled {
		logger.Debug(fmt.Sprintf("DEBUG: 开始 REALITY 握手，SNI: %s\n", v.sni))

		// 使用自动生成的 TLS 配置
		defaultOptions := v.GetDefaultTLSOptions()

		// 提取服务器地址（不带端口）
		serverAddr := v.server
		if idx := strings.Index(serverAddr, ":"); idx != -1 {
			serverAddr = serverAddr[:idx]
		}

		logger.Debug(fmt.Sprintf("DEBUG: TLS ServerAddr: %s, SNI: %s\n", serverAddr, defaultOptions.ServerName))
		tlsConf, err := stls.NewClient(ctx, serverAddr, common.PtrValueOrDefault(defaultOptions))
		if err != nil {
			return nil, fmt.Errorf("failed to create TLS client: %w", err)
		}

		tlsConn, err := stls.ClientHandshake(ctx, conn, tlsConf)
		if err != nil {
			logger.Error(fmt.Sprintf("[VLESS] TLS握手失败 目标: %s:%d, 错误: %v", metadata.HostName, metadata.DstPort, err))
			return nil, fmt.Errorf("failed to handshake with VLESS server: %w", err)
		}
		logger.Info(fmt.Sprintf("[VLESS] ✓ TLS握手成功 SNI: %s, 目标: %s:%d", v.sni, metadata.HostName, metadata.DstPort))

		// 创建目标地址
		// 重要：如果有域名，优先使用域名而不是IP，这样目标服务器才能正确处理TLS SNI
		destination := smeta.Socksaddr{
			Port: metadata.DstPort,
		}

		// 优先使用 Fqdn（域名）
		if metadata.HostName != "" {
			destination.Fqdn = metadata.HostName
			logger.Debug(fmt.Sprintf("DEBUG: VLESS 目标（使用域名）: %s:%d\n", metadata.HostName, metadata.DstPort))
		} else {
			destination.Addr = metadata.DstIP
			logger.Debug(fmt.Sprintf("DEBUG: VLESS 目标（使用IP）: %s:%d\n", metadata.DstIP, metadata.DstPort))
		}

		visionConn, err := v.client.DialEarlyConn(tlsConn, destination)
		if err != nil {
			// 关键诊断：记录错误的具体类型
			errMsg := fmt.Sprintf("[VLESS] ❌ Early Dial失败 目标: %s:%d, 错误: %v", metadata.HostName, metadata.DstPort, err)

			if err == io.EOF {
				errMsg += " [EOF - 可能原因: 服务器不支持0-RTT或握手失败]"
			} else if err.Error() == "context canceled" {
				errMsg += " [Context取消]"
			}

			logger.Error(errMsg)
			return nil, fmt.Errorf("failed to dial early connection: %w", err)
		}

		logger.Info(fmt.Sprintf("[VLESS] ✓ Early Dial成功 目标: %s:%d, 连接类型: %T", metadata.HostName, metadata.DstPort, visionConn))
		return visionConn, nil
	}

	// 如果启用了 TLS，进行 TLS 握手
	if v.sni != "" {
		tlsConfig := &tls.Config{
			ServerName:         v.sni,
			InsecureSkipVerify: true,
			MinVersion:         tls.VersionTLS12,
			MaxVersion:         tls.VersionTLS13,
		}
		conn = tls.Client(conn, tlsConfig)
	}

	fmt.Printf("DEBUG: 连接建立成功\n")
	return conn, nil
}

// DialUDP 实现 Proxy 接口的 DialUDP 方法
func (v *VLESS) DialUDP(metadata *M.Metadata) (net.PacketConn, error) {
	// VLESS 的 UDP 支持需要先建立 TCP 连接
	// 这里返回一个错误，表示不支持 UDP
	return nil, fmt.Errorf("VLESS UDP not supported")
}

// GetDefaultTLSOptions 从 VLESS 配置生成 Sing-box 的 OutboundTLSOptions
// 包含完整的 TLS、REALITY、UTLS 等配置
func (v *VLESS) GetDefaultTLSOptions() *sopt.OutboundTLSOptions {
	// 如果没有 SNI，返回 nil（表示不使用 TLS）
	if v.sni == "" {
		return nil
	}

	opts := &sopt.OutboundTLSOptions{
		Enabled:    true,
		ServerName: v.sni,
		Insecure:   false, // REALITY 不需要跳过证书验证
	}

	// 配置 TLS 版本
	// Vision 模式推荐使用 TLS 1.3
	if v.flow == "xtls-rprx-vision" {
		opts.MinVersion = "1.3"
		opts.MaxVersion = "1.3"
	} else {
		opts.MinVersion = "1.2"
		opts.MaxVersion = "1.3"
	}

	// 配置 ALPN（应用层协议协商）
	// HTTP/2 和 HTTP/1.1 是常用的协议
	opts.ALPN = badoption.Listable[string]([]string{"h2", "http/1.1"})

	// 如果启用了 REALITY，配置 REALITY 选项
	if v.reality != nil && v.reality.Enabled {
		opts.Reality = &sopt.OutboundRealityOptions{
			Enabled:   true,
			PublicKey: v.reality.PublicKey,
			ShortID:   v.reality.ShortID,
		}

		// REALITY 通常使用 uTLS 来模拟真实浏览器指纹
		opts.UTLS = &sopt.OutboundUTLSOptions{
			Enabled:     true,
			Fingerprint: "chrome", // 模拟 Chrome 浏览器
		}
	}

	// 推荐的密码套件（优先使用更安全的）
	opts.CipherSuites = badoption.Listable[string]([]string{
		"TLS_AES_128_GCM_SHA256",
		"TLS_AES_256_GCM_SHA384",
		"TLS_CHACHA20_POLY1305_SHA256",
		"TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256",
		"TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256",
	})

	return opts
}

// GetSimpleTLSOptions 获取简化的 TLS 配置（用于快速配置）
func (v *VLESS) GetSimpleTLSOptions() *sopt.OutboundTLSOptions {
	if v.sni == "" {
		return nil
	}

	opts := &sopt.OutboundTLSOptions{
		Enabled:    true,
		ServerName: v.sni,
		Insecure:   true,
		MinVersion: "1.2",
		MaxVersion: "1.3",
	}

	// 如果有 REALITY 配置
	if v.reality != nil && v.reality.Enabled {
		opts.Reality = &sopt.OutboundRealityOptions{
			Enabled:   true,
			PublicKey: v.reality.PublicKey,
			ShortID:   v.reality.ShortID,
		}
	}

	return opts
}

// GetCustomTLSOptions 获取自定义的 TLS 配置
// 允许覆盖特定字段
func (v *VLESS) GetCustomTLSOptions(customize func(*sopt.OutboundTLSOptions)) *sopt.OutboundTLSOptions {
	opts := v.GetDefaultTLSOptions()
	if opts != nil && customize != nil {
		customize(opts)
	}
	return opts
}
