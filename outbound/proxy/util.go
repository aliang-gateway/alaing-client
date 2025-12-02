package proxy

import (
	"net"
	"time"

	"github.com/xjasonlyu/tun2socks/v2/transport/socks5"
	M "nursor.org/nursorgate/inbound/tun/metadata"
)

const (
	tcpKeepAlivePeriod = 30 * time.Second
)

// setKeepAlive sets tcp keepalive option for tcp connection.
func SetKeepAlive(c net.Conn) {
	if tcp, ok := c.(*net.TCPConn); ok {
		tcp.SetKeepAlive(true)
		tcp.SetKeepAlivePeriod(tcpKeepAlivePeriod)
	}
}

// safeConnClose closes tcp connection safely.
func SafeConnClose(c net.Conn, err error) {
	if c != nil && err != nil {
		c.Close()
	}
}

// serializeSocksAddr 将目的地址序列化为 SOCKS5 地址格式，使用 tun2socks 的 socks5.SerializeAddr
func SerializeSocksAddr(metadata *M.Metadata) []byte {
	// 使用 tun2socks 的 socks5.SerializeAddr 确保格式完全一致
	addr := socks5.SerializeAddr("", metadata.DstIP, metadata.DstPort)
	return []byte(addr)
}
