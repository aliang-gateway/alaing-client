package proxy

import (
	"context"
	"crypto/tls"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	xnet "github.com/xtls/xray-core/common/net"
	"github.com/xtls/xray-core/common/protocol"
	xuuid "github.com/xtls/xray-core/common/uuid"
	vless "github.com/xtls/xray-core/proxy/vless"
	vencl "github.com/xtls/xray-core/proxy/vless/encoding"
	reality "github.com/xtls/xray-core/transport/internet/reality"

	M "nursor.org/nursorgate/client/server/tun/metadata"
	"nursor.org/nursorgate/client/server/tun/proxy/proto"
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
	conn        net.Conn
	lastUsed    time.Time
	isAvailable bool
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
func NewVLESSWithReality(server, uuid, sni, publicKey, shortID string) (*VLESS, error) {
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
	parsedUUID, err := uuid.Parse(config.UUID)
	if err != nil {
		return nil, fmt.Errorf("invalid UUID: %w", err)
	}

	v := &VLESS{
		Base: &Base{
			addr:  fmt.Sprintf("%s:%d", config.Server, config.ServerPort),
			proto: proto.VLESS,
		},
		server:    config.Server,
		uuid:      config.UUID,
		uuidBytes: parsedUUID[:], // 转换为字节数组
	}

	if config.TLS != nil {
		v.sni = config.TLS.ServerName
		if config.TLS.Reality != nil {
			v.reality = config.TLS.Reality
		}
	}
	v.flow = config.Flow

	// 初始化连接池，参考 xray-core 的连接管理
	v.connPool = NewConnectionPool(5) // 默认最大5个连接

	return v, nil
}

// NewConnectionPool 创建连接池
func NewConnectionPool(maxSize int) *ConnectionPool {
	return &ConnectionPool{
		connections: make(chan *PooledConnection, maxSize),
		maxSize:     maxSize,
	}
}

// Get 从连接池获取连接，参考 xray-core 连接获取机制
func (cp *ConnectionPool) Get() *PooledConnection {
	for {
		select {
		case conn := <-cp.connections:
			// 检查连接是否仍然可用且未过期
			if conn != nil && conn.isAvailable && time.Since(conn.lastUsed) < 30*time.Second {
				conn.mu.Lock()
				conn.lastUsed = time.Now()
				conn.mu.Unlock()
				return conn
			}
			// 不合格则关闭并继续取下一个
			if conn != nil && conn.conn != nil {
				conn.conn.Close()
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
	if conn == nil || !conn.isAvailable {
		return
	}

	select {
	case cp.connections <- conn:
		// 成功放回池中
	default:
		// 池已满，关闭连接
		conn.conn.Close()
	}
}

// DialContext 实现 Proxy 接口的 DialContext 方法，参考 xray-core 连接复用机制
func (v *VLESS) DialContext(ctx context.Context, metadata *M.Metadata) (net.Conn, error) {
	v.mu.RLock()
	defer v.mu.RUnlock()

	// 尝试从连接池获取已握手的连接
	if pooledConn := v.connPool.Get(); pooledConn != nil {
		fmt.Printf("DEBUG: 复用连接池中的连接\n")
		return v.wrapConnectionForTarget(pooledConn, metadata), nil
	}

	// 连接池为空，创建新连接
	fmt.Printf("DEBUG: 连接池为空，创建新连接\n")
	conn, err := v.establishNewConnection(ctx, metadata)
	if err != nil {
		return nil, err
	}

	// 将新连接放入连接池（如果池未满）
	v.connPool.Put(conn)

	return v.wrapConnectionForTarget(conn, metadata), nil
}

// establishNewConnection 建立新连接并完成握手，参考 xray-core 连接建立流程
func (v *VLESS) establishNewConnection(ctx context.Context, metadata *M.Metadata) (*PooledConnection, error) {
	// 连接到 VLESS 服务器
	conn, err := net.DialTimeout("tcp", v.Addr(), 10*time.Second)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to VLESS server: %w", err)
	}

	// 如果启用了 REALITY，进行 REALITY 握手
	if v.reality != nil && v.reality.Enabled {
		// 直接使用 Xray-core 的 UClient 方法完成 REALITY 握手
		realityConn, err := v.performXrayRealityHandshake(ctx, conn, metadata)
		if err != nil {
			conn.Close()
			return nil, fmt.Errorf("failed to perform Xray REALITY handshake: %w", err)
		}

		// 发送 VLESS 握手并返回包装后的连接（若 Vision 则包裹编码/解码）
		wrappedConn, err := v.sendVLESSHandshake(ctx, realityConn, metadata)
		if err != nil {
			fmt.Printf("DEBUG: VLESS 握手失败: %v\n", err)
			realityConn.Close()
			return nil, fmt.Errorf("VLESS handshake failed: %w", err)
		}

		fmt.Printf("DEBUG: VLESS 握手成功\n")
		return &PooledConnection{
			conn:        wrappedConn,
			lastUsed:    time.Now(),
			isAvailable: true,
		}, nil
	}

	// 如果启用了 TLS，进行 TLS 握手
	if v.sni != "" {
		tlsConfig := &tls.Config{
			ServerName:         v.sni,
			InsecureSkipVerify: true,
			MinVersion:         tls.VersionTLS12, // 强制使用 TLS 1.2
			MaxVersion:         tls.VersionTLS12, // 避免 TLS 1.3
		}
		conn = tls.Client(conn, tlsConfig)
	}

	// 发送 VLESS 握手并返回包装后的连接
	wrappedConn, err := v.sendVLESSHandshake(ctx, conn, metadata)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to send VLESS handshake: %w", err)
	}

	return &PooledConnection{
		conn:        wrappedConn,
		lastUsed:    time.Now(),
		isAvailable: true,
	}, nil
}

// wrapConnectionForTarget 为连接包装目标地址信息，参考 xray-core 连接包装
func (v *VLESS) wrapConnectionForTarget(pooledConn *PooledConnection, metadata *M.Metadata) net.Conn {
	return &VLESSWrappedConn{
		PooledConnection: pooledConn,
		targetAddr:       metadata.DestinationAddress(),
		targetPort:       metadata.DstPort,
		hasSetTarget:     false,
	}
}

// VLESSWrappedConn 包装连接，支持动态目标地址设置
type VLESSWrappedConn struct {
	*PooledConnection
	targetAddr   string
	targetPort   uint16
	hasSetTarget bool
	mu           sync.RWMutex
}

func (vc *VLESSWrappedConn) Write(p []byte) (int, error) {
	// 第一次写入时设置目标地址（如果需要）
	vc.mu.Lock()
	if !vc.hasSetTarget {
		// VLESS 协议在握手时已经设置了目标地址，这里不需要再次设置
		vc.hasSetTarget = true
	}
	vc.mu.Unlock()

	return vc.conn.Write(p)
}

func (vc *VLESSWrappedConn) Read(p []byte) (int, error) {
	return vc.conn.Read(p)
}

func (vc *VLESSWrappedConn) Close() error {
	// 连接关闭时，标记为不可用但不关闭底层连接（用于连接池复用）
	vc.mu.Lock()
	vc.isAvailable = false
	vc.mu.Unlock()
	return nil // 不关闭底层连接，让连接池管理
}

// 实现 net.Conn 接口的其他方法
func (vc *VLESSWrappedConn) LocalAddr() net.Addr {
	return vc.conn.LocalAddr()
}

func (vc *VLESSWrappedConn) RemoteAddr() net.Addr {
	return vc.conn.RemoteAddr()
}

func (vc *VLESSWrappedConn) SetDeadline(t time.Time) error {
	return vc.conn.SetDeadline(t)
}

func (vc *VLESSWrappedConn) SetReadDeadline(t time.Time) error {
	return vc.conn.SetReadDeadline(t)
}

func (vc *VLESSWrappedConn) SetWriteDeadline(t time.Time) error {
	return vc.conn.SetWriteDeadline(t)
}

// performXrayRealityHandshake 使用 Xray-core 的 UClient 完成 REALITY 握手
func (v *VLESS) performXrayRealityHandshake(ctx context.Context, conn net.Conn, metadata *M.Metadata) (net.Conn, error) {
	if v.reality == nil || !v.reality.Enabled {
		return nil, fmt.Errorf("REALITY not enabled")
	}

	shortIDBytes := v.parseShortID(v.reality.ShortID)

	// PublicKey 为 base64url 编码字符串，需要解码为原始字节
	pubKeyBytes, err := base64.RawURLEncoding.DecodeString(v.reality.PublicKey)
	if err != nil {
		return nil, fmt.Errorf("invalid REALITY public key: %w", err)
	}

	cfg := &reality.Config{
		ServerName: v.sni,
		PublicKey:  pubKeyBytes,
		ShortId:    shortIDBytes[:],
	}
	server, port, _ := xnet.SplitHostPort(v.Addr())
	portInt, _ := xnet.PortFromString(port)
	dest := xnet.TCPDestination(xnet.ParseAddress(server), portInt)

	fmt.Printf("DEBUG: 使用 Xray-core reality.UClient 握手, SNI=%s, ShortID=%x\n", v.sni, shortIDBytes)
	realityConn, err := reality.UClient(conn, cfg, ctx, dest)
	if err != nil {
		fmt.Printf("DEBUG: REALITY UClient 握手失败: %v\n", err)
		return nil, err
	}
	fmt.Printf("DEBUG: REALITY UClient 握手成功\n")
	return realityConn, nil
}

// parseShortID 解析 ShortID 字符串为字节数组
func (v *VLESS) parseShortID(shortID string) [8]byte {
	var result [8]byte

	// 如果 ShortID 是十六进制字符串
	if len(shortID) == 12 {
		// 填充到 16 个字符（8 字节）
		paddedShortID := shortID + "0000"
		bytes, err := hex.DecodeString(paddedShortID)
		if err != nil {
			// 如果解析失败，使用原始字符串
			shortIDBytes := []byte(shortID)
			if len(shortIDBytes) > 8 {
				copy(result[:], shortIDBytes[:8])
			} else {
				copy(result[:], shortIDBytes)
			}
			return result
		}
		if len(bytes) == 8 {
			copy(result[:], bytes)
			return result
		}
	}

	// 如果 ShortID 是其他格式，尝试填充到 8 字节
	shortIDBytes := []byte(shortID)
	if len(shortIDBytes) > 8 {
		copy(result[:], shortIDBytes[:8])
	} else {
		copy(result[:], shortIDBytes)
	}

	return result
}

// DialUDP 实现 Proxy 接口的 DialUDP 方法
func (v *VLESS) DialUDP(metadata *M.Metadata) (net.PacketConn, error) {
	// VLESS 的 UDP 支持需要先建立 TCP 连接
	// 这里返回一个错误，表示不支持 UDP
	return nil, fmt.Errorf("VLESS UDP not supported")
}

// sendVLESSHandshake 发送 VLESS 握手
func (v *VLESS) sendVLESSHandshake(ctx context.Context, conn net.Conn, metadata *M.Metadata) (net.Conn, error) {
	// 1) 构造用户（MemoryAccount + protocol.ID）
	uParsed, err := xuuid.ParseString(v.uuid)
	if err != nil {
		return nil, fmt.Errorf("invalid uuid: %w", err)
	}
	user := &protocol.MemoryUser{
		Account: &vless.MemoryAccount{
			ID:   protocol.NewID(uParsed),
			Flow: v.flow,
		},
	}
	fmt.Printf("DEBUG: VLESS 用户信息 - UUID: %s, Flow: %s\n", v.uuid, v.flow)

	targetHost := metadata.DstIP.String()

	fmt.Printf("DEBUG: 目标主机: %s, 端口: %d\n", targetHost, metadata.DstPort)
	addr := xnet.ParseAddress(targetHost)
	port := xnet.Port(metadata.DstPort)

	// 3) 请求头
	req := &protocol.RequestHeader{
		Version: vencl.Version,
		User:    user,
		Command: protocol.RequestCommandTCP,
		Address: addr,
		Port:    port,
	}
	fmt.Printf("DEBUG: VLESS 请求头 - Version: %d, Command: %d, Address: %s, Port: %d\n",
		req.Version, req.Command, req.Address.String(), req.Port)

	// 4) Addons（Vision 流需要在 Addons 中设置 Flow）
	addons := &vencl.Addons{}
	if v.flow == "xtls-rprx-vision" {
		addons.Flow = "xtls-rprx-vision"
	}
	fmt.Printf("DEBUG: VLESS Addons - Flow: %s\n", addons.Flow)

	// 5) 编码并写入请求头
	if err := vencl.EncodeRequestHeader(conn, req, addons); err != nil {
		return nil, err
	}
	fmt.Printf("DEBUG: VLESS 请求头发送成功\n")

	// 6) Vision 流不需要读取响应头，直接开始数据传输
	if v.flow == "xtls-rprx-vision" {
		fmt.Printf("DEBUG: Vision 流不需要响应头，直接开始数据传输\n")
	} else {
		// 非 Vision 流需要读取响应头
		fmt.Printf("DEBUG: 开始读取 VLESS 响应头...\n")
		responseAddons, err := vencl.DecodeResponseHeader(conn, req)
		if err != nil {
			fmt.Printf("DEBUG: VLESS 响应头解码失败: %v\n", err)
			return nil, fmt.Errorf("VLESS handshake failed: %w", err)
		}

		fmt.Printf("DEBUG: VLESS 响应头解码成功\n")
		if responseAddons != nil {
			fmt.Printf("DEBUG: 响应 Addons: %+v\n", responseAddons)
		}
	}

	// 7) 返回握手成功的连接
	fmt.Printf("DEBUG: VLESS 握手完成，返回连接\n")
	return conn, nil
}

// 旧的 vlessWrappedConn 已移除，使用新的 VLESSWrappedConn

// GetConfig 获取当前配置
func (v *VLESS) GetConfig() *VLESSConfig {
	return &VLESSConfig{
		Server: v.server,
		UUID:   v.uuid,
		Flow:   v.flow,
		TLS: &TLSConfig{
			Enabled:    v.sni != "",
			ServerName: v.sni,
			Reality:    v.reality,
		},
	}
}

// String 返回字符串表示
func (v *VLESS) String() string {
	config := v.GetConfig()
	data, _ := json.Marshal(config)
	return fmt.Sprintf("VLESS(%s)", string(data))
}
