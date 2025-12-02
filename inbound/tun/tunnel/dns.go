package tunnel

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/netip"
	"strings"
	"sync"
	"time"

	"github.com/miekg/dns"

	M "nursor.org/nursorgate/inbound/tun/metadata"
	"nursor.org/nursorgate/outbound/proxy"
)

var defaultResolver *DNSResolver

func SetDefaultResolver(resolver *DNSResolver) {
	defaultResolver = resolver
}

func GetDefaultResolver() *DNSResolver {
	return defaultResolver
}

// DNSResolver 提供可指定 dialer 与 DNS 服务器的解析能力，并内置简单缓存。
type DNSResolver struct {
	// dnsServer 形如 "8.8.8.8:53" 或 "1.1.1.1"（未带端口则默认 53）
	dnsServer string
	dialer    proxy.Dialer
	timeout   time.Duration
	maxTTL    time.Duration

	mu    sync.RWMutex
	cache map[string]cacheEntry // key: qname|qtype
}

type cacheEntry struct {
	expiresAt time.Time
	ips       []net.IP
}

// NewDNSResolver 创建一个解析器。
// 说明：
// - dnsServer 推荐直接填写 IP:53，避免本地再解析带来循环依赖。
// - dialer 例如传入 doorProxy（Hysteria2）或 defaultProxy，根据你的路由需求选择。
// - timeout 为单次解析的超时时间，maxTTL 为缓存的最大生存期（会与应答 TTL 取较小值）。
func NewDNSResolver(dnsServer string, dialer proxy.Dialer, timeout, maxTTL time.Duration) *DNSResolver {
	if !strings.Contains(dnsServer, ":") {
		dnsServer = net.JoinHostPort(dnsServer, "53")
	}
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	if maxTTL <= 0 {
		maxTTL = 5 * time.Minute
	}
	return &DNSResolver{
		dnsServer: dnsServer,
		dialer:    dialer,
		timeout:   timeout,
		maxTTL:    maxTTL,
		cache:     make(map[string]cacheEntry),
	}
}

// LookupA 解析 A 记录并返回 IPv4 列表（带缓存）。
func (r *DNSResolver) LookupA(ctx context.Context, qname string) ([]net.IP, error) {
	return r.lookup(ctx, qname, dns.TypeA)
}

// LookupAAAA 解析 AAAA 记录并返回 IPv6 列表（带缓存）。
func (r *DNSResolver) LookupAAAA(ctx context.Context, qname string) ([]net.IP, error) {
	return r.lookup(ctx, qname, dns.TypeAAAA)
}

func (r *DNSResolver) lookup(ctx context.Context, qname string, qtype uint16) ([]net.IP, error) {
	key := fmt.Sprintf("%s|%d", strings.ToLower(strings.TrimSuffix(qname, ".")), qtype)

	// 读缓存
	r.mu.RLock()
	if ce, ok := r.cache[key]; ok && time.Now().Before(ce.expiresAt) {
		r.mu.RUnlock()
		return cloneIPs(ce.ips), nil
	}
	r.mu.RUnlock()

	// 实际查询（使用 TCP/53，避免 PacketConn 适配复杂度）
	ips, ttl, err := r.exchangeOverTCP(ctx, ensureDotSuffix(qname), qtype)
	if err != nil {
		return nil, err
	}
	if len(ips) == 0 {
		return nil, errors.New("no answer")
	}

	// 写缓存
	life := ttl
	if life <= 0 || life > r.maxTTL {
		life = r.maxTTL
	}
	r.mu.Lock()
	r.cache[key] = cacheEntry{expiresAt: time.Now().Add(life), ips: cloneIPs(ips)}
	r.mu.Unlock()
	return ips, nil
}

// exchangeOverTCP 通过指定 dialer 建立到 DNS 服务器的 TCP 连接并完成一次查询。
func (r *DNSResolver) exchangeOverTCP(ctx context.Context, qname string, qtype uint16) ([]net.IP, time.Duration, error) {
	host, portStr, err := net.SplitHostPort(r.dnsServer)
	if err != nil {
		return nil, 0, fmt.Errorf("invalid dnsServer: %w", err)
	}
	port, err := parsePort(portStr)
	if err != nil {
		return nil, 0, err
	}

	ip := net.ParseIP(host)
	if ip == nil {
		// 为避免循环依赖，这里不再解析域名 DNS 服务器；要求传入 IP。
		return nil, 0, fmt.Errorf("dnsServer must be an IP: %s", r.dnsServer)
	}

	md := &M.Metadata{ // 目标为 DNS 服务器
		Network: M.TCP,
		DstIP:   toNetip(ip),
		DstPort: uint16(port),
	}

	// 带超时的上下文
	var cancel context.CancelFunc
	if deadline, ok := ctx.Deadline(); !ok || time.Until(deadline) > r.timeout {
		ctx, cancel = context.WithTimeout(ctx, r.timeout)
		defer cancel()
	}

	conn, err := r.dialer.DialContext(ctx, md)
	if err != nil {
		return nil, 0, err
	}
	defer func() { _ = conn.Close() }()

	c := &dns.Conn{Conn: conn}
	// 某些连接类型可能不支持 deadline，这里忽略错误以兼容
	_ = c.SetWriteDeadline(time.Now().Add(r.timeout))
	_ = c.SetReadDeadline(time.Now().Add(r.timeout))

	msg := new(dns.Msg)
	msg.SetQuestion(qname, qtype)
	msg.RecursionDesired = true

	if err := c.WriteMsg(msg); err != nil {
		return nil, 0, err
	}
	resp, err := c.ReadMsg()
	if err != nil {
		return nil, 0, err
	}
	if resp == nil || resp.Rcode != dns.RcodeSuccess {
		return nil, 0, fmt.Errorf("dns rcode: %d", resp.Rcode)
	}

	var (
		ips []net.IP
		ttl uint32 = 0
	)
	// 取最小 TTL 作为缓存时间
	minTTL := func(a, b uint32) uint32 {
		if a == 0 {
			return b
		}
		if b == 0 {
			return a
		}
		if a < b {
			return a
		}
		return b
	}

	for _, rr := range resp.Answer {
		switch v := rr.(type) {
		case *dns.A:
			ips = append(ips, v.A)
			ttl = minTTL(ttl, v.Hdr.Ttl)
		case *dns.AAAA:
			ips = append(ips, v.AAAA)
			ttl = minTTL(ttl, v.Hdr.Ttl)
		}
	}

	return ips, time.Duration(ttl) * time.Second, nil
}

// 工具函数
func ensureDotSuffix(s string) string {
	if strings.HasSuffix(s, ".") {
		return s
	}
	return s + "."
}

func cloneIPs(src []net.IP) []net.IP {
	out := make([]net.IP, len(src))
	for i := range src {
		if src[i] != nil {
			b := make([]byte, len(src[i]))
			copy(b, src[i])
			out[i] = net.IP(b)
		}
	}
	return out
}

func parsePort(s string) (int, error) {
	p, err := net.LookupPort("tcp", s)
	if err == nil {
		return p, nil
	}
	// s 不是命名端口，尝试数值
	var n int
	_, scanErr := fmt.Sscanf(s, "%d", &n)
	if scanErr != nil || n <= 0 || n > 65535 {
		return 0, fmt.Errorf("invalid port: %s", s)
	}
	return n, nil
}

func toNetip(ip net.IP) netip.Addr {
	if v4 := ip.To4(); v4 != nil {
		return netip.AddrFrom4([4]byte{v4[0], v4[1], v4[2], v4[3]})
	}
	if v6 := ip.To16(); v6 != nil {
		var arr [16]byte
		copy(arr[:], v6)
		return netip.AddrFrom16(arr)
	}
	return netip.IPv4Unspecified()
}
