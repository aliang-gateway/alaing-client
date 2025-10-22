package proxy

import (
	"context"
	"crypto/tls"
	"fmt"
	"math/rand"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	stls "github.com/sagernet/sing-box/common/tls"
	sopt "github.com/sagernet/sing-box/option"
	vlessSingBox "github.com/sagernet/sing-vmess/vless"
	"github.com/sagernet/sing/common"
	"github.com/sagernet/sing/common/json/badoption"
	smeta "github.com/sagernet/sing/common/metadata"
	"nursor.org/nursorgate/client/server/tun/dialer"
	M "nursor.org/nursorgate/client/server/tun/metadata"
	"nursor.org/nursorgate/client/server/tun/proxy/proto"
	"nursor.org/nursorgate/common/logger"
)

// VLESS 使用简化的 VLESS 实现，参考 xray-core 设计
type VLESS struct {
	*Base
	server    string
	uuid      string
	uuidBytes []byte
	sni       string
	flow      string
	reality   *RealityConfig
	client    *vlessSingBox.Client

	// 连接池管理，参考 xray-core 的连接复用机制
	connPool *ConnectionPool
	mu       sync.RWMutex
}

// ConnectionPool 连接池，参考 xray-core 的连接管理
type ConnectionPool struct {
	connections chan *PooledConnection
	maxSize     int
	mu          sync.RWMutex
}

// PooledConnection 池化连接
type PooledConnection struct {
	Conn        net.Conn
	LastUsed    time.Time
	IsAvailable bool
	mu          sync.RWMutex
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
	shortIDStr := "5f2b,8e5607,4bfe0963,8c593074,335fad66be5a,98fd,b219,f3,1bc5bd29,ccc7ab,5b6933f199,865f,751b8836,eb,530abf,3de2da,8f,e5af86261f33,08,3ec3,4f58,d0,fa9063"
	shortIDSArray := strings.Split(shortIDStr, ",")
	shortID := shortIDSArray[rand.Intn(len(shortIDSArray))]
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
	// 解析 UUID
	//parsedUUID, err := uuid.Parse(config.UUID)
	//if err != nil {
	//	return nil, fmt.Errorf("invalid UUID: %w", err)
	//}

	// 使用 SingBox 兼容的 logger
	singBoxLogger := logger.GetSingBoxLogger()
	client, err := vlessSingBox.NewClient(config.UUID, config.Flow, singBoxLogger)
	if err != nil {
		return nil, fmt.Errorf("failed to create vless client: %w", err)
	}

	v := &VLESS{
		Base: &Base{
			addr:  fmt.Sprintf("%s:%d", config.Server, config.ServerPort),
			proto: proto.VLESS,
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

// Get 从连接池获取连接，参考 xray-core 连接获取机制
func (cp *ConnectionPool) Get() *PooledConnection {
	for {
		select {
		case conn := <-cp.connections:
			// 检查连接是否仍然可用且未过期
			if conn != nil && conn.IsAvailable && time.Since(conn.LastUsed) < 30*time.Second {
				conn.mu.Lock()
				conn.LastUsed = time.Now()
				conn.mu.Unlock()
				return conn
			}
			// 不合格则关闭并继续取下一个
			if conn != nil && conn.Conn != nil {
				conn.Conn.Close()
			}
			// 继续循环尝试从池中取下一个
			continue
		default:
			// 池当前没有更多连接了
			return nil
		}
	}
}

// Put 将连接放回连接池，参考 xray-core 连接回收机制
func (cp *ConnectionPool) Put(conn *PooledConnection) {
	if conn == nil || !conn.IsAvailable {
		return
	}

	select {
	case cp.connections <- conn:
		// 成功放回池中
	default:
		// 池已满，关闭连接
		conn.Conn.Close()
	}
}

// DialContext 实现 Proxy 接口的 DialContext 方法，参考 xray-core 连接复用机制
func (v *VLESS) DialContext(ctx context.Context, metadata *M.Metadata) (net.Conn, error) {
	v.mu.RLock()
	defer v.mu.RUnlock()

	fmt.Printf("DEBUG: 创建新连接到 %s:%d\n", metadata.HostName, metadata.DstPort)
	conn, err := v.establishNewConnection(ctx, metadata)
	if err != nil {
		return nil, err
	}

	return conn, nil
}

// establishNewConnection 建立新连接并准备握手数据，参考 xray-core 连接建立流程
func (v *VLESS) establishNewConnection(ctx context.Context, metadata *M.Metadata) (net.Conn, error) {
	// 连接到 VLESS 服务器，使用默认 dialer
	fmt.Printf("DEBUG: 连接到 VLESS 服务器 %s\n", v.Addr())
	conn, err := dialer.DialContext(ctx, "tcp", v.Addr())
	if err != nil {
		return nil, fmt.Errorf("failed to connect to VLESS server: %w", err)
	}

	// 如果启用了 REALITY，进行 REALITY 握手
	if v.reality != nil && v.reality.Enabled {
		fmt.Printf("DEBUG: 开始 REALITY 握手，SNI: %s\n", v.sni)

		// 使用自动生成的 TLS 配置
		defaultOptions := v.GetDefaultTLSOptions()

		// 提取服务器地址（不带端口）
		serverAddr := v.server
		if idx := strings.Index(serverAddr, ":"); idx != -1 {
			serverAddr = serverAddr[:idx]
		}

		fmt.Printf("DEBUG: TLS ServerAddr: %s, SNI: %s\n", serverAddr, defaultOptions.ServerName)
		tlsConf, err := stls.NewClient(ctx, serverAddr, common.PtrValueOrDefault(defaultOptions))
		if err != nil {
			return nil, fmt.Errorf("failed to create TLS client: %w", err)
		}

		tlsConn, err := stls.ClientHandshake(ctx, conn, tlsConf)
		if err != nil {
			return nil, fmt.Errorf("failed to handshake with VLESS server: %w", err)
		}
		fmt.Printf("DEBUG: REALITY TLS 握手完成\n")

		// 创建目标地址
		// 重要：如果有域名，优先使用域名而不是IP，这样目标服务器才能正确处理TLS SNI
		destination := smeta.Socksaddr{
			Port: metadata.DstPort,
		}

		// 优先使用 Fqdn（域名）
		if metadata.HostName != "" {
			destination.Fqdn = metadata.HostName
			fmt.Printf("DEBUG: VLESS 目标（使用域名）: %s:%d\n", metadata.HostName, metadata.DstPort)
		} else {
			destination.Addr = metadata.DstIP
			fmt.Printf("DEBUG: VLESS 目标（使用IP）: %s:%d\n", metadata.DstIP, metadata.DstPort)
		}

		visionConn, err := v.client.DialEarlyConn(tlsConn, destination)
		if err != nil {
			return nil, fmt.Errorf("failed to dial early connection: %w", err)
		}

		fmt.Printf("DEBUG: VLESS 连接建立成功，类型: %T\n", visionConn)
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
