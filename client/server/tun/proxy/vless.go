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
	"github.com/xtls/xray-core/common/buf"
	xnet "github.com/xtls/xray-core/common/net"
	"github.com/xtls/xray-core/common/protocol"
	xuuid "github.com/xtls/xray-core/common/uuid"
	"github.com/xtls/xray-core/proxy"
	vless "github.com/xtls/xray-core/proxy/vless"
	vencl "github.com/xtls/xray-core/proxy/vless/encoding"
	reality "github.com/xtls/xray-core/transport/internet/reality"

	M "nursor.org/nursorgate/client/server/tun/metadata"
	"nursor.org/nursorgate/client/server/tun/proxy/proto"
)

// VLESS 使用简化的 VLESS 实现
type VLESS struct {
	*Base
	server    string
	uuid      string
	uuidBytes []byte
	sni       string
	flow      string
	reality   *RealityConfig
	mu        sync.RWMutex
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

// NewVLESSWithConfig 使用配置创建 VLESS 客户端
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

	return v, nil
}

// DialContext 实现 Proxy 接口的 DialContext 方法
func (v *VLESS) DialContext(ctx context.Context, metadata *M.Metadata) (net.Conn, error) {
	v.mu.RLock()
	defer v.mu.RUnlock()

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
			return nil, fmt.Errorf("Xray REALITY handshake failed: %w", err)
		}

		// 发送 VLESS 握手并返回包装后的连接（若 Vision 则包裹编码/解码）
		wrappedConn, err := v.sendVLESSHandshake(ctx, realityConn, metadata)
		if err != nil {
			fmt.Printf("DEBUG: VLESS 握手失败: %v\n", err)
			realityConn.Close()
			return nil, fmt.Errorf("VLESS handshake failed: %w", err)
		}

		fmt.Printf("DEBUG: VLESS 握手成功\n")
		return wrappedConn, nil
	}

	// 如果启用了 TLS，进行 TLS 握手
	if v.sni != "" {
		tlsConfig := &tls.Config{
			ServerName:         v.sni,
			InsecureSkipVerify: true,
		}
		conn = tls.Client(conn, tlsConfig)
	}

	// 发送 VLESS 握手并返回包装后的连接
	wrappedConn, err := v.sendVLESSHandshake(ctx, conn, metadata)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to send VLESS handshake: %w", err)
	}

	return wrappedConn, nil
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

	// 目的地址：使用 SNI:443 作为目标
	dest := xnet.TCPDestination(xnet.ParseAddress(v.sni), xnet.Port(443))

	fmt.Printf("DEBUG: 使用 Xray-core reality.UClient 握手, SNI=%s, ShortID=%x\n", v.sni, shortIDBytes)

	realityConn, err := reality.UClient(conn, cfg, ctx, dest)
	if err != nil {
		return nil, err
	}
	return realityConn, nil
}

// performRealityHandshake 执行 REALITY 握手（已废弃，使用 uTLS）
func (v *VLESS) performRealityHandshake(conn net.Conn, metadata *M.Metadata) (net.Conn, error) {
	// 这个方法已经不再使用，现在使用 uTLS 进行 REALITY 握手
	return nil, fmt.Errorf("performRealityHandshake is deprecated, use uTLS instead")
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
	// 使用 xray-core 的 VLESS 编码逻辑，避免手写协议

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

	// 2) 目标地址和端口
	var addr xnet.Address
	if metadata.DstIP.Is4() || metadata.DstIP.Is6() {
		addr = xnet.IPAddress(metadata.DstIP.AsSlice())
	} else {
		addr = xnet.ParseAddress(v.sni)
	}
	port := xnet.Port(metadata.DstPort)

	// 3) 请求头
	req := &protocol.RequestHeader{
		Version: vencl.Version,
		User:    user,
		Command: protocol.RequestCommandTCP,
		Address: addr,
		Port:    port,
	}

	// 4) Addons（Vision 流需要设置为 XRV 才会写入 protobuf 附加字段）
	addons := &vencl.Addons{}
	if v.flow == "xtls-rprx-vision" {
		addons.Flow = vless.XRV
	}

	// 5) 编码并写入请求头
	if err := vencl.EncodeRequestHeader(conn, req, addons); err != nil {
		return nil, err
	}

	// Vision 流量需要对后续数据进行编解码，使用 xray-core 的 EncodeBodyAddons/DecodeBodyAddons
	if v.flow == "xtls-rprx-vision" {
		bw := vencl.EncodeBodyAddons(conn, req, addons, &proxy.TrafficState{}, true, ctx)
		br := vencl.DecodeBodyAddons(conn, req, addons)
		return newVLESSWrappedConn(conn, bw, br), nil
	}

	// 非 Vision，直接返回原连接
	return conn, nil
}

// vlessWrappedConn 将 xray 的 buf.Reader/Writer 适配为 net.Conn
type vlessWrappedConn struct {
	net.Conn
	w       buf.Writer
	r       buf.Reader
	readBuf []byte
}

func newVLESSWrappedConn(underlying net.Conn, w buf.Writer, r buf.Reader) net.Conn {
	return &vlessWrappedConn{Conn: underlying, w: w, r: r}
}

func (vc *vlessWrappedConn) Write(p []byte) (int, error) {
	b := buf.New()
	if _, err := b.Write(p); err != nil {
		b.Release()
		return 0, err
	}
	if err := vc.w.WriteMultiBuffer(buf.MultiBuffer{b}); err != nil {
		return 0, err
	}
	return len(p), nil
}

func (vc *vlessWrappedConn) Read(p []byte) (int, error) {
	// 先消费缓存
	if len(vc.readBuf) > 0 {
		n := copy(p, vc.readBuf)
		vc.readBuf = vc.readBuf[n:]
		return n, nil
	}
	mb, err := vc.r.ReadMultiBuffer()
	if err != nil {
		return 0, err
	}
	defer buf.ReleaseMulti(mb)
	if mb.IsEmpty() {
		return 0, nil
	}
	// 合并为连续字节
	var total int
	for _, b := range mb {
		total += int(b.Len())
	}
	if total <= len(p) {
		off := 0
		for _, b := range mb {
			o := copy(p[off:], b.Bytes())
			off += o
		}
		return total, nil
	}
	vc.readBuf = make([]byte, total)
	off := 0
	for _, b := range mb {
		o := copy(vc.readBuf[off:], b.Bytes())
		off += o
	}
	n := copy(p, vc.readBuf)
	vc.readBuf = vc.readBuf[n:]
	return n, nil
}

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
