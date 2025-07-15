package proxy

import (
	"context"
	"errors"
	"net"
	"sync"
	"time"

	M "nursor.org/nursorgate/client/server/tun/metadata"
	"nursor.org/nursorgate/client/server/tun/proxy/proto"

	hysteria "github.com/apernet/hysteria/core/v2/client"
)

// 确保实现 proxy.Dialer 接口
var _ interface {
	DialContext(context.Context, *M.Metadata) (net.Conn, error)
	DialUDP(*M.Metadata) (net.PacketConn, error)
} = (*HysteriaDialer)(nil)

type HysteriaDialer struct {
	config *hysteria.Config

	client hysteria.Client
	once   sync.Once
	err    error
}

// NewHysteriaDialer 创建 Dialer 并建立连接
func NewHysteriaDialer(config *hysteria.Config) (*HysteriaDialer, error) {
	h := &HysteriaDialer{
		config: config,
	}
	h.once.Do(func() {
		h.client, _, h.err = hysteria.NewClient(config)
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
	return proto.Hy
}

func (h *HysteriaDialer) Addr() string {
	return h.config.ServerAddr.String()
}

type hysteriaUDPConn struct {
	session hysteria.HyUDPConn
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
