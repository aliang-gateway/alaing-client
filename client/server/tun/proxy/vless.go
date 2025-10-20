package proxy

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
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

	"nursor.org/nursorgate/client/server/tun/dialer"
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
		Flow:       "xtls-rprx-vision", // ✅ 启用 Vision，使用 Xray-core 的实现
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

	if config.TLS != nil && config.TLS.Enabled {
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
	// 连接池为空，创建新连接
	fmt.Printf("DEBUG: 连接池为空，创建新连接\n")
	conn, err := v.establishNewConnection(ctx, metadata)
	if err != nil {
		return nil, err
	}

	// 准备握手数据
	handshakeData, err := v.prepareVLESSHandshake(ctx, metadata)
	if err != nil {
		conn.Conn.Close()
		return nil, fmt.Errorf("failed to prepare handshake for new connection: %w", err)
	}
	return v.wrapConnectionForTarget(conn, metadata, handshakeData), nil
}

// establishNewConnection 建立新连接并准备握手数据，参考 xray-core 连接建立流程
func (v *VLESS) establishNewConnection(ctx context.Context, metadata *M.Metadata) (*PooledConnection, error) {
	// 连接到 VLESS 服务器，使用默认 dialer
	conn, err := dialer.DialContext(ctx, "tcp", v.Addr())
	if err != nil {
		return nil, fmt.Errorf("failed to connect to VLESS server: %w", err)
	}

	// 如果启用了 REALITY，进行 REALITY 握手
	if v.reality != nil && v.reality.Enabled {
		fmt.Printf("DEBUG: 开始 REALITY 握手，SNI: %s\n", v.sni)
		// 使用 Xray-core 的 UClient 方法完成 REALITY 握手
		realityConn, err := v.performXrayRealityHandshake(ctx, conn, metadata)
		if err != nil {
			conn.Close()
			return nil, fmt.Errorf("failed to perform Xray REALITY handshake: %w", err)
		}

		fmt.Printf("DEBUG: REALITY 握手成功\n")
		return &PooledConnection{
			Conn:        realityConn,
			LastUsed:    time.Now(),
			IsAvailable: true,
		}, nil
	}

	// 如果启用了 TLS，进行 TLS 握手
	if v.sni != "" {
		fmt.Printf("DEBUG: 开始 TLS 握手，SNI: %s\n", v.sni)
		tlsConfig := &tls.Config{
			ServerName:         v.sni,
			InsecureSkipVerify: true,
			MinVersion:         tls.VersionTLS13,
			MaxVersion:         tls.VersionTLS13,
		}
		conn = tls.Client(conn, tlsConfig)
		fmt.Printf("DEBUG: TLS 握手成功\n")
	}

	fmt.Printf("DEBUG: 连接建立成功\n")
	return &PooledConnection{
		Conn:        conn,
		LastUsed:    time.Now(),
		IsAvailable: true,
	}, nil
}

// wrapConnectionForTarget 为连接包装目标地址信息，参考 xray-core 连接包装
func (v *VLESS) wrapConnectionForTarget(pooledConn *PooledConnection, metadata *M.Metadata, handshakeData []byte) net.Conn {
	wrapped := &VLESSWrappedConn{
		PooledConnection: pooledConn,
		TargetAddr:       metadata.DestinationAddress(),
		TargetPort:       metadata.DstPort,
		HasSetTarget:     false,
		HandshakeData:    handshakeData,
		HandshakeSent:    false,
		flow:             v.flow,
	}

	// 如果启用了 Vision，创建 Vision Reader/Writer
	if v.flow == "xtls-rprx-vision" {
		// 解析 UUID 为字节
		uuidParsed, _ := uuid.Parse(v.uuid)
		wrapped.userUUID = uuidParsed[:]

		// 创建 TrafficState
		wrapped.trafficState = proxy.NewTrafficState(wrapped.userUUID)

		// 创建 buf.Reader/Writer（Xray-core 的接口）
		wrapped.bufReader = buf.NewReader(pooledConn.Conn)
		wrapped.bufWriter = buf.NewWriter(pooledConn.Conn)

		// ⚠️ 不再创建 bufferedReader！
		// 直接使用 bufReader 读取 VLESS Response Header，然后交给 visionReader 读取应用数据
		// 这样可以确保数据流的一致性

		// 使用 Xray-core 的 VisionReader/VisionWriter 包装
		wrapped.visionReader = proxy.NewVisionReader(wrapped.bufReader, wrapped.trafficState, false, context.Background())
		wrapped.visionWriter = proxy.NewVisionWriter(wrapped.bufWriter, wrapped.trafficState, true, context.Background())

		fmt.Printf("DEBUG: Vision 已启用，使用 Xray-core VisionReader/VisionWriter\n")
	}

	return wrapped
}

// VLESSWrappedConn 包装连接，支持动态目标地址设置和延迟握手发送
type VLESSWrappedConn struct {
	*PooledConnection
	TargetAddr   string
	TargetPort   uint16
	HasSetTarget bool

	// 延迟握手相关字段
	HandshakeData []byte // 缓存的握手数据
	HandshakeSent bool   // 握手是否已发送

	// 响应头处理相关字段
	ResponseHeaderRead bool // VLESS 响应头是否已读取
	mu                 sync.RWMutex

	// Vision 相关字段（使用 Xray-core 的实现）
	flow         string
	userUUID     []byte
	trafficState *proxy.TrafficState
	visionReader *proxy.VisionReader
	visionWriter *proxy.VisionWriter
	bufReader    buf.Reader
	bufWriter    buf.Writer

	// 缓冲区：保存 Vision Reader 读取但未消费的数据
	readBuffer buf.MultiBuffer
}

func (vc *VLESSWrappedConn) Write(p []byte) (int, error) {
	vc.mu.Lock()
	defer vc.mu.Unlock()

	// 如果握手数据还没有发送，先发送握手数据
	if !vc.HandshakeSent && len(vc.HandshakeData) > 0 {
		handshakeLen := len(vc.HandshakeData)
		fmt.Printf("DEBUG: 发送延迟的VLESS握手数据，长度: %d bytes, payload: %d bytes\n", handshakeLen, len(p))

		// ⚠️ 重要：VLESS 握手数据必须直接写入底层连接，不能经过 Vision！
		// Vision 只处理通过 VLESS 隧道传输的应用数据（TLS）

		// 1. 先发送 VLESS 握手（不经过 Vision）
		totalWritten := 0
		for totalWritten < handshakeLen {
			n, err := vc.Conn.Write(vc.HandshakeData[totalWritten:])
			if err != nil {
				fmt.Printf("DEBUG: VLESS 握手发送失败: %v\n", err)
				return 0, err
			}
			totalWritten += n
		}
		fmt.Printf("DEBUG: VLESS 握手发送成功: %d bytes\n", handshakeLen)

		vc.HandshakeSent = true
		vc.HandshakeData = nil

		// 2. 然后发送 payload（通过 Vision 如果启用的话）
		if len(p) > 0 {
			if vc.visionWriter != nil {
				// ⚠️ 检查是否应该切换到 Direct Copy
				if vc.trafficState != nil &&
					!vc.trafficState.Outbound.IsPadding &&
					vc.trafficState.Outbound.UplinkWriterDirectCopy {
					fmt.Printf("DEBUG: Vision Writer 已停止 padding，直接写入应用数据\n")
					vc.visionWriter = nil
					return vc.Conn.Write(p)
				}

				fmt.Printf("DEBUG: 使用 Vision Writer 发送应用数据: %d bytes\n", len(p))
				buffer := buf.New()
				buffer.Write(p)
				mb := buf.MultiBuffer{buffer}

				err := vc.visionWriter.WriteMultiBuffer(mb)
				if err != nil {
					fmt.Printf("DEBUG: Vision Write 失败: %v\n", err)
					return 0, err
				}

				// 写入后检查是否需要切换（IsPadding 变为 false 后才切换）
				if vc.trafficState != nil &&
					!vc.trafficState.Outbound.IsPadding &&
					vc.trafficState.Outbound.UplinkWriterDirectCopy {
					fmt.Printf("DEBUG: Vision Writer 已完成 padding，下次 Write 将直接写入\n")
					vc.visionWriter = nil
				}

				return len(p), nil
			} else {
				// 标准写入
				return vc.Conn.Write(p)
			}
		}

		return 0, nil
	}

	// 握手已发送，直接发送数据
	if vc.visionWriter != nil {
		// ⚠️ 检查是否应该切换到 Direct Copy
		// 注意：VisionWriter 在内部已经停止 padding 后，我们才切换
		if vc.trafficState != nil &&
			!vc.trafficState.Outbound.IsPadding &&
			vc.trafficState.Outbound.UplinkWriterDirectCopy {
			fmt.Printf("DEBUG: Vision Writer 已停止 padding 且 Direct Copy 已启用，切换到直接写入\n")
			vc.visionWriter = nil
			// 当前数据直接写入
			return vc.Conn.Write(p)
		}

		// 使用 Vision Writer
		buffer := buf.New()
		buffer.Write(p)
		mb := buf.MultiBuffer{buffer}
		err := vc.visionWriter.WriteMultiBuffer(mb)
		if err != nil {
			return 0, err
		}

		// 写入后检查是否需要切换（IsPadding 变为 false 后才切换）
		if vc.trafficState != nil &&
			!vc.trafficState.Outbound.IsPadding &&
			vc.trafficState.Outbound.UplinkWriterDirectCopy {
			fmt.Printf("DEBUG: Vision Writer 已完成 padding，下次 Write 将直接写入\n")
			vc.visionWriter = nil
		}

		return len(p), nil
	}

	// 标准写入
	return vc.Conn.Write(p)
}

func (vc *VLESSWrappedConn) Read(p []byte) (int, error) {
	// ⚠️ 重要：VLESS 响应头必须读取，但不能经过 Vision！
	// 在 Vision 模式下，必须从 bufReader 读取（因为 visionReader 基于它）
	// 在标准模式下，直接从 Conn 读取
	if !vc.ResponseHeaderRead {
		var err error
		var version byte
		var addonLen int

		if vc.bufReader != nil {
			// Vision 模式：从 bufReader 读取响应头
			// 需要读取 2 + addonLen 字节，但 addonLen 在第2个字节，所以分两次读取
			vc.Conn.SetReadDeadline(time.Now().Add(3 * time.Second))
			defer vc.Conn.SetReadDeadline(time.Time{})

			// 第一次读取：读取前2字节（version + addonLen）
			mb, err := vc.bufReader.ReadMultiBuffer()
			if err != nil {
				return 0, fmt.Errorf("read vless response header: %w", err)
			}
			if mb.IsEmpty() {
				return 0, fmt.Errorf("read vless response header: empty buffer")
			}

			// 从 MultiBuffer 中提取数据
			firstBuf := mb[0]
			if firstBuf.Len() < 2 {
				buf.ReleaseMulti(mb)
				return 0, fmt.Errorf("read vless response header: buffer too small")
			}

			version = firstBuf.Byte(0)
			addonLen = int(firstBuf.Byte(1))

			// 如果有 addons，继续读取
			if addonLen > 0 {
				// 检查第一个 buffer 是否包含足够的数据
				if firstBuf.Len() >= int32(2+addonLen) {
					// 数据已经在 buffer 中
					addonBytes := firstBuf.BytesRange(2, int32(2+addonLen))
					fmt.Printf("DEBUG: VLESS resp addons: %x\n", addonBytes)
				} else {
					// 需要读取更多数据
					// 先释放已读取的
					buf.ReleaseMulti(mb)

					// 读取 addons
					addonsMB, err := vc.bufReader.ReadMultiBuffer()
					if err != nil {
						return 0, fmt.Errorf("read vless addons: %w", err)
					}
					if !addonsMB.IsEmpty() && addonsMB[0] != nil {
						fmt.Printf("DEBUG: VLESS resp addons: %x\n", addonsMB[0].Bytes())
					}
					buf.ReleaseMulti(addonsMB)
				}
			} else {
				// 没有 addons，释放 buffer
				buf.ReleaseMulti(mb)
			}
		} else {
			// 标准模式：直接从 Conn 读取
			header := make([]byte, 2)
			vc.Conn.SetReadDeadline(time.Now().Add(3 * time.Second))
			_, err = io.ReadFull(vc.Conn, header)
			vc.Conn.SetReadDeadline(time.Time{})

			if err != nil {
				return 0, fmt.Errorf("read vless response header: %w", err)
			}

			version = header[0]
			addonLen = int(header[1])

			if addonLen > 0 {
				addons := make([]byte, addonLen)
				if _, err = io.ReadFull(vc.Conn, addons); err != nil {
					return 0, fmt.Errorf("read vless addons: %w", err)
				}
				fmt.Printf("DEBUG: VLESS resp addons: %x\n", addons)
			}
		}

		vc.ResponseHeaderRead = true
		fmt.Printf("DEBUG: VLESS resp header ok, version=%d addons=%d\n", version, addonLen)

		// 响应头读取完成后，才开始使用 Vision Reader
		if vc.visionReader != nil {
			fmt.Printf("DEBUG: VLESS 响应头读取完成，现在切换到 Vision Reader\n")
		}
	}

	// 响应头读取完成后，应用数据才使用 Vision
	if vc.visionReader != nil {
		var totalRead int

		// 1. 先从缓冲区读取（如果有的话）
		if !vc.readBuffer.IsEmpty() {
			for len(vc.readBuffer) > 0 && totalRead < len(p) {
				buffer := vc.readBuffer[0]
				if buffer != nil {
					n := copy(p[totalRead:], buffer.Bytes())
					totalRead += n

					if int32(n) < buffer.Len() {
						// p 已满，但 buffer 还有数据，保留剩余数据
						buffer.Advance(int32(n))
						break
					} else {
						// buffer 已全部复制，释放并移除
						buffer.Release()
						vc.readBuffer = vc.readBuffer[1:]
					}
				} else {
					vc.readBuffer = vc.readBuffer[1:]
				}
			}

			if totalRead > 0 {
				fmt.Printf("DEBUG: 从缓冲区读取 %d bytes\n", totalRead)
				return totalRead, nil
			}
		}

		// 2. 缓冲区为空，从 Vision Reader 读取新数据
		fmt.Printf("DEBUG: 从 Vision Reader 读取新数据...\n")
		mb, err := vc.visionReader.ReadMultiBuffer()
		if err != nil {
			fmt.Printf("DEBUG: Vision Read 错误: %v\n", err)
			return 0, err
		}
		if mb.IsEmpty() {
			fmt.Printf("DEBUG: Vision Read 返回空数据\n")
			return 0, io.EOF
		}

		// 检查第一个 buffer 的前 16 字节，看是否是 UserUUID（Vision 格式）
		if len(mb) > 0 && mb[0] != nil && mb[0].Len() >= 16 {
			firstBytes := mb[0].BytesTo(16)
			if bytes.Equal(firstBytes, vc.userUUID) {
				fmt.Printf("DEBUG: ✅ 检测到 Vision padding 格式（以 UserUUID 开头）\n")
			} else {
				fmt.Printf("DEBUG: ⚠️ 服务器返回数据不是 Vision 格式！前16字节: %x\n", firstBytes)
				fmt.Printf("DEBUG: 期望的 UserUUID: %x\n", vc.userUUID)
				fmt.Printf("DEBUG: 🔧 检测到 Vision 不对称模式：\n")
				fmt.Printf("    - 客户端→服务器：继续使用 Vision (服务器要求)\n")
				fmt.Printf("    - 服务器→客户端：禁用 Vision (服务器不使用)\n")

				// ⚠️ 重要：只禁用 Read 方向的 Vision，保持 Write 方向
				// 因为服务器可能只在接收时使用 Vision，发送时不使用
				vc.visionReader = nil
				// visionWriter 保持启用！服务器需要接收 Vision 格式

				// 当前读取的数据是原始数据，直接使用
			}
		}

		// 3. 复制数据到 p
		for _, buffer := range mb {
			if buffer != nil && totalRead < len(p) {
				n := copy(p[totalRead:], buffer.Bytes())
				totalRead += n

				if int32(n) < buffer.Len() {
					// p 已满，但 buffer 还有数据，保存到缓冲区
					buffer.Advance(int32(n))
					vc.readBuffer = append(vc.readBuffer, buffer)
					fmt.Printf("DEBUG: buffer 数据过多，保存 %d bytes 到缓冲区\n", buffer.Len())
				} else {
					// buffer 已全部复制，释放
					buffer.Release()
				}
			} else if buffer != nil {
				// p 已满，保存整个 buffer 到缓冲区
				vc.readBuffer = append(vc.readBuffer, buffer)
			}
		}

		fmt.Printf("DEBUG: Vision Read 成功: %d bytes (请求 %d bytes, 缓冲区剩余 %d buffers)\n",
			totalRead, len(p), len(vc.readBuffer))

		// ⚠️ 检查是否应该切换到 Direct Copy（在返回数据之后）
		if vc.trafficState != nil && vc.trafficState.Outbound.DownlinkReaderDirectCopy {
			fmt.Printf("DEBUG: Vision 检测到 Direct Copy 标志，下次 Read 将直接从底层连接读取\n")
			// 设置标志：下次 Read 时不再使用 Vision
			vc.visionReader = nil
			// 注意：当前读取的数据仍然会正常返回
		}

		if totalRead == 0 {
			fmt.Printf("DEBUG: 警告：Vision Read 返回了数据但复制了 0 字节\n")
			return 0, io.EOF
		}

		return totalRead, nil
	}

	// 标准读取
	return vc.Conn.Read(p)
}

func (vc *VLESSWrappedConn) Close() error {
	// 连接关闭时，标记为不可用并关闭底层连接
	vc.mu.Lock()
	vc.IsAvailable = false
	vc.mu.Unlock()

	// 真正关闭底层连接
	if vc.Conn != nil {
		return vc.Conn.Close()
	}
	return nil
}

// 实现 net.Conn 接口的其他方法
func (vc *VLESSWrappedConn) LocalAddr() net.Addr {
	return vc.Conn.LocalAddr()
}

func (vc *VLESSWrappedConn) RemoteAddr() net.Addr {
	return vc.Conn.RemoteAddr()
}

func (vc *VLESSWrappedConn) SetDeadline(t time.Time) error {
	return vc.Conn.SetDeadline(t)
}

func (vc *VLESSWrappedConn) SetReadDeadline(t time.Time) error {
	return vc.Conn.SetReadDeadline(t)
}

func (vc *VLESSWrappedConn) SetWriteDeadline(t time.Time) error {
	return vc.Conn.SetWriteDeadline(t)
}

// performXrayRealityHandshake 使用 Xray-core 的 UClient 完成 REALITY 握手
func (v *VLESS) performXrayRealityHandshake(ctx context.Context, conn net.Conn, metadata *M.Metadata) (net.Conn, error) {
	if v.reality == nil || !v.reality.Enabled {
		return nil, fmt.Errorf("REALITY not enabled")
	}
	// 解析 ShortID
	shortIDBytes := v.parseShortID(v.reality.ShortID)
	// 验证 ShortID
	allZero := true
	for _, b := range shortIDBytes {
		if b != 0 {
			allZero = false
			break
		}
	}
	if allZero && v.reality.ShortID != "" {
		return nil, fmt.Errorf("REALITY ShortID 解析失败，全为零值。原始值: %s", v.reality.ShortID)
	}

	// PublicKey 为 base64url 编码字符串，需要解码为原始字节
	fmt.Printf("DEBUG: 原始 PublicKey: %s\n", v.reality.PublicKey)
	pubKeyBytes, err := base64.RawURLEncoding.DecodeString(v.reality.PublicKey)
	if err != nil {
		return nil, fmt.Errorf("invalid REALITY public key: %w", err)
	}
	fmt.Printf("DEBUG: PublicKey 长度: %d bytes\n", len(pubKeyBytes))

	// 验证 PublicKey 长度（应该是 32 字节）
	if len(pubKeyBytes) != 32 {
		return nil, fmt.Errorf("invalid REALITY public key length: expected 32 bytes, got %d bytes", len(pubKeyBytes))
	}

	// SpiderY: 反爬虫机制配置，需要至少 10 个值
	spiderY := []int64{
		100, 1000, // [0-1] cookie padding 范围
		10, 100, // [2-3] 并发数范围
		1, 5, // [4-5] 重试次数范围
		100, 1000, // [6-7] 间隔时间范围（毫秒）
		100, 1000, // [8-9] 返回延迟范围（毫秒）
	}

	cfg := &reality.Config{
		ServerName: v.sni,
		PublicKey:  pubKeyBytes,
		ShortId:    shortIDBytes,
		SpiderY:    spiderY, // ✅ 添加 SpiderY 配置，避免 panic
	}
	server, port, _ := xnet.SplitHostPort(v.Addr())
	portInt, _ := xnet.PortFromString(port)
	dest := xnet.TCPDestination(xnet.ParseAddress(server), portInt)

	realityConn, err := reality.UClient(conn, cfg, ctx, dest)
	if err != nil {
		return nil, fmt.Errorf("REALITY handshake failed: %w", err)
	}
	return realityConn, nil
}

func (v *VLESS) parseShortID(shortID string) []byte {
	// REALITY 要求 ShortID 必须是 8 字节
	// 参考 xray-core/transport/internet/reality/config.go:54-56
	result := make([]byte, 8)

	if shortID == "" {
		fmt.Printf("DEBUG: ShortID为空，返回全零的8字节: %x\n", result)
		return result
	}

	// 先尝试hex解码
	b, err := hex.DecodeString(shortID)
	if err != nil {
		// 非hex格式，使用原始字节并填充
		b = []byte(shortID)
		fmt.Printf("DEBUG: 非hex格式ShortID，使用原始字节: %s\n", shortID)
	} else {
		fmt.Printf("DEBUG: HEX格式ShortID=%s, 解码为%d字节: %x\n", shortID, len(b), b)
	}

	// 复制到 8 字节结果中（不足则右侧填零，多余则截断）
	if len(b) >= 8 {
		copy(result, b[:8])
		fmt.Printf("DEBUG: ShortID截断到8字节: %x\n", result)
	} else {
		copy(result, b)
		fmt.Printf("DEBUG: ShortID填充到8字节: %x (原始%d字节)\n", result, len(b))
	}

	return result
}

// DialUDP 实现 Proxy 接口的 DialUDP 方法
func (v *VLESS) DialUDP(metadata *M.Metadata) (net.PacketConn, error) {
	// VLESS 的 UDP 支持需要先建立 TCP 连接
	// 这里返回一个错误，表示不支持 UDP
	return nil, fmt.Errorf("VLESS UDP not supported")
}

// PrepareVLESSHandshake 准备 VLESS 握手数据但不立即发送（公开方法）
func (v *VLESS) PrepareVLESSHandshake(ctx context.Context, metadata *M.Metadata) ([]byte, error) {
	return v.prepareVLESSHandshake(ctx, metadata)
}

// prepareVLESSHandshake 准备 VLESS 握手数据但不立即发送
func (v *VLESS) prepareVLESSHandshake(ctx context.Context, metadata *M.Metadata) ([]byte, error) {
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

	// 优先使用 HostName；为空时降级到 IP
	targetHost := metadata.HostName
	if targetHost == "" {
		targetHost = metadata.DstIP.String()
		if targetHost == "" {
			return nil, fmt.Errorf("no destination host available (hostname and IP are empty)")
		}
	}

	fmt.Printf("DEBUG: 目标 Host: %s, 端口: %d\n", targetHost, metadata.DstPort)
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

	// 5) 编码握手数据到缓冲区而不是直接写入连接
	var buf bytes.Buffer
	if err := vencl.EncodeRequestHeader(&buf, req, addons); err != nil {
		return nil, err
	}

	handshakeData := buf.Bytes()
	fmt.Printf("DEBUG: VLESS 握手数据准备完成，长度: %d bytes\n", len(handshakeData))
	return handshakeData, nil
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
