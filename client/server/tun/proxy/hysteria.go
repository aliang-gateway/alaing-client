package proxy

import (
	"context"
	"errors"
	"fmt"
	"github.com/apernet/hysteria/extras/v2/obfs"
	"net"
	"sync"
	"time"

	M "nursor.org/nursorgate/client/server/tun/metadata"
	"nursor.org/nursorgate/client/server/tun/proxy/proto"
	"nursor.org/nursorgate/common/logger"

	"github.com/apernet/hysteria/core/v2/client"
	// hysteria "github.com/apernet/hysteria/core/v2/client"
)

// 确保实现 proxy.Dialer 接口
var _ interface {
	DialContext(context.Context, *M.Metadata) (net.Conn, error)
	DialUDP(*M.Metadata) (net.PacketConn, error)
} = (*HysteriaDialer)(nil)

type HysteriaDialer struct {
	config *client.Config

	client client.Client
	once   sync.Once
	err    error
}

type adaptiveConnFactory struct {
	NewFunc    func(addr net.Addr) (net.PacketConn, error)
	Obfuscator obfs.Obfuscator // nil if no obfuscation
}

func (f *adaptiveConnFactory) New(addr net.Addr) (net.PacketConn, error) {
	if f.Obfuscator == nil {
		return f.NewFunc(addr)
	} else {
		conn, err := f.NewFunc(addr)
		if err != nil {
			return nil, err
		}
		return obfs.WrapPacketConn(conn, f.Obfuscator), nil
	}
}

func applyToUDPConn(c *net.UDPConn) error {

	return nil
}

func getDefaultConnFactory(salamada string) client.ConnFactory {
	var ob obfs.Obfuscator
	ob, _ = obfs.NewSalamanderObfuscator([]byte(salamada))

	// Inner PacketConn
	var newFunc func(addr net.Addr) (net.PacketConn, error)
	newFunc = func(addr net.Addr) (net.PacketConn, error) {
		uconn, err := net.ListenUDP("udp", nil)
		if err != nil {
			return nil, err
		}
		return uconn, nil
	}
	// Obfuscation
	return &adaptiveConnFactory{
		NewFunc:    newFunc,
		Obfuscator: ob,
	}

}

func BuildHysteriaClientConfig(username, password string) (*client.Config, error) {
	server := "8.209.245.103:1443"

	addr, err := net.ResolveUDPAddr("udp", server)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve server address: %w", err)
	}

	return &client.Config{
		ConnFactory: getDefaultConnFactory("2hKDWT79uWNIJuRMS5jqFNyOtSIf05Oc"),
		ServerAddr:  addr,
		Auth:        fmt.Sprintf("%s:%s", username, password),
		TLSConfig: client.TLSConfig{
			ServerName:         "node1.nursor.org",
			InsecureSkipVerify: true,
		},
		QUICConfig: client.QUICConfig{
			InitialStreamReceiveWindow:     8388608,
			MaxStreamReceiveWindow:         8388608,
			InitialConnectionReceiveWindow: 20971520,
			MaxConnectionReceiveWindow:     20971520,
			MaxIdleTimeout:                 20 * time.Second,
			KeepAlivePeriod:                20 * time.Second,
			DisablePathMTUDiscovery:        false,
		},
		BandwidthConfig: client.BandwidthConfig{
			MaxTx: 1024 * 5,
			MaxRx: 1024 * 5,
		},
		FastOpen: true,
	}, nil
}

// NewHysteriaDialer 创建 Dialer 并建立连接
func NewHysteriaDialer(username, password string) (*HysteriaDialer, error) {
	config, err := BuildHysteriaClientConfig(username, password)
	if err != nil {
		logger.Error("failed to build hysteria client config: %v", err)
		return nil, err
	}
	h := &HysteriaDialer{
		config: config,
	}
	h.once.Do(func() {
		h.client, _, h.err = client.NewClient(config)
	})
	return h, h.err
}

func (h *HysteriaDialer) DialContext(ctx context.Context, m *M.Metadata) (net.Conn, error) {
	if h.client == nil {
		return nil, errors.New("Hysteria client not initialized")
	}
	target := m.DstIP.String()
	type result struct {
		conn net.Conn
		err  error
	}
	ch := make(chan result, 1)
	go func() {
		conn, err := h.client.TCP(target)
		ch <- result{conn, err}
	}()
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case res := <-ch:
		return res.conn, res.err
	}
}

func (h *HysteriaDialer) DialUDP(m *M.Metadata) (net.PacketConn, error) {
	if h.client == nil {
		return nil, errors.New("Hysteria client not initialized")
	}
	session, err := h.client.UDP()
	if err != nil {
		return nil, err
	}
	return &hysteriaUDPConn{
		session: session,
		raddr:   &net.UDPAddr{IP: net.IP(m.DstIP.AsSlice()), Port: int(m.DstPort)},
	}, nil
}

func (h *HysteriaDialer) Proto() proto.Proto {
	return proto.HY2
}

func (h *HysteriaDialer) Addr() string {
	return h.config.ServerAddr.String()
}

type hysteriaUDPConn struct {
	session client.HyUDPConn
	raddr   net.Addr
}

func (c *hysteriaUDPConn) ReadFrom(b []byte) (n int, addr net.Addr, err error) {
	data, src, err := c.session.Receive()
	if err != nil {
		return 0, nil, err
	}
	copy(b, data)
	return len(data), &net.UDPAddr{IP: net.ParseIP(src), Port: 0}, nil
}

func (c *hysteriaUDPConn) WriteTo(b []byte, addr net.Addr) (n int, err error) {
	err = c.session.Send(b, addr.String())
	if err != nil {
		return 0, err
	}
	return len(b), nil
}

func (c *hysteriaUDPConn) Close() error {
	return c.session.Close()
}

func (c *hysteriaUDPConn) LocalAddr() net.Addr {
	return &net.UDPAddr{IP: net.IPv4zero, Port: 0}
}

func (c *hysteriaUDPConn) SetDeadline(t time.Time) error {
	return nil
}

func (c *hysteriaUDPConn) SetReadDeadline(t time.Time) error {
	return nil
}

func (c *hysteriaUDPConn) SetWriteDeadline(t time.Time) error {
	return nil
}
