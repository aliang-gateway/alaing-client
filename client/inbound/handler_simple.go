package inbound

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/netip"
	"sync"
	"time"

	"nursor.org/nursorgate/client/outbound"
	"nursor.org/nursorgate/client/server/tun/buffer"
	"nursor.org/nursorgate/client/server/tun/metadata"
	"nursor.org/nursorgate/client/server/tun/proxy"
	"nursor.org/nursorgate/client/server/tun/tunnel"
	"nursor.org/nursorgate/client/utils"
	"nursor.org/nursorgate/common/logger"
)

const tcpWaitTimeout = time.Second * 60 * 10

func HandleTLSConnectionSimple(tlsConn *tls.Conn, req *http.Request) {
	logger.Info("parse ", req.Host)
	var isHttp2 = true
	alpnVersion := tlsConn.ConnectionState().NegotiatedProtocol
	if alpnVersion != "h2" {
		isHttp2 = false
	}
	outBoundClient, err := outbound.NewHttp2ProxyClient(utils.GetServerHost(), req.Host, isHttp2)
	if err != nil {
		logger.Error(err.Error())
		return
	}
	pipe(tlsConn, outBoundClient.Conn)
}

func HandleTLSConnectionSimpleWithoutDecrypt(conn net.Conn, sni string, ip string, req *http.Request) {
	targetPort := uint16(443)
	if req.URL.Scheme == "http" {
		targetPort = uint16(80)
	} else {
		targetPort = uint16(443)
	}
	// 解析目标域名
	targetHost := sni
	// 移除端口号（如果有的话）
	if host, _, err := net.SplitHostPort(targetHost); err == nil {
		targetHost = host
	}

	logger.Info("Target host:", targetHost)

	// 解析域名到IP地址
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 使用默认DNS解析器
	resolver := tunnel.GetDefaultResolver()
	if resolver == nil {
		// 如果没有默认解析器，创建一个直连的解析器
		directDialer := proxy.NewDirect()
		resolver = tunnel.NewDNSResolver("8.8.8.8:53", directDialer, 5*time.Second, 5*time.Minute)
	}

	ips, err := resolver.LookupA(ctx, targetHost)
	if err != nil {
		logger.Error("DNS lookup failed for", targetHost, ":", err.Error())
		return
	}

	if len(ips) == 0 {
		logger.Error("No IP addresses found for", targetHost)
		return
	}

	// 使用第一个IP地址
	targetIP := ips[0]
	logger.Info("Resolved", targetHost, "to", targetIP.String())
	// 将net.IP转换为netip.Addr
	var dstIP netip.Addr
	if ipv4 := targetIP.To4(); ipv4 != nil {
		dstIP = netip.AddrFrom4([4]byte{ipv4[0], ipv4[1], ipv4[2], ipv4[3]})
	} else if ipv6 := targetIP.To16(); ipv6 != nil {
		var arr [16]byte
		copy(arr[:], ipv6)
		dstIP = netip.AddrFrom16(arr)
	} else {
		logger.Error("Invalid IP address:", targetIP.String())
		return
	}

	// 使用已修复的 VLESS + REALITY + Vision 实现
	outboundVless, err := proxy.NewVLESSWithReality(
		"103.255.209.43:443",                          // 服务器地址
		"d9868dc7-3547-4195-95f1-5455748e7706",        // UUID
		"www.cloudflare.com",                          // SNI
		"2cLV-hIMZlfDWUc0ESMUCnKYhDVEQL4WGzydSspzfEw", // REALITY PublicKey
		"838f28591b",                                  // REALITY ShortID
	)
	if err != nil {
		logger.Error(err.Error(), sni)
		return
	}
	// 创建Metadata结构体
	meta := &metadata.Metadata{
		Network:  metadata.TCP,
		DstIP:    dstIP,
		HostName: targetHost,
		DstPort:  targetPort,
	}

	outboundVlessClient, err := outboundVless.DialContext(context.Background(), meta)
	if err != nil {
		logger.Error("Failed to dial VLESS proxy:", err.Error())
		return
	}
	logger.Info("parse ", sni)

	pipe(conn, outboundVlessClient)
}

// pipe copies data to & from provided net.Conn(s) bidirectionally.
func pipe(origin, remote net.Conn) {
	logger.Info("Starting pipe between origin:", origin.RemoteAddr(), "and remote:", remote.RemoteAddr())
	logger.Info("Origin type:", fmt.Sprintf("%T", origin), "Remote type:", fmt.Sprintf("%T", remote))

	wg := sync.WaitGroup{}
	wg.Add(2)

	go unidirectionalStream(remote, origin, "origin->remote", &wg)
	go unidirectionalStream(origin, remote, "remote->origin", &wg)

	wg.Wait()
	logger.Info("Pipe completed")
}

func unidirectionalStream(dst, src net.Conn, dir string, wg *sync.WaitGroup) {
	defer wg.Done()
	logger.Info("Starting unidirectional stream:", dir, "from", src.RemoteAddr(), "to", dst.RemoteAddr())
	buf := buffer.Get(buffer.RelayBufferSize)
	n, err := io.CopyBuffer(dst, src, buf)
	if err != nil {
		logger.Error("[TCP] copy data for", dir, "error:", err, "bytes copied:", n)
	} else {
		logger.Info("[TCP] copy data for", dir, "success, bytes copied:", n)
	}
	buffer.Put(buf)
	// Do the upload/download side TCP half-close.
	if cr, ok := src.(interface{ CloseRead() error }); ok {
		cr.CloseRead()
	}
	if cw, ok := dst.(interface{ CloseWrite() error }); ok {
		cw.CloseWrite()
	}
	// Set TCP half-close timeout.
	dst.SetReadDeadline(time.Now().Add(tcpWaitTimeout))
}

// determinePortFromMetadata 通过Metadata结构体和连接信息判断目标端口
func determinePortFromMetadata(conn net.Conn) uint16 {
	// 通过连接的远程端口判断
	if remoteAddr := conn.RemoteAddr(); remoteAddr != nil {
		if tcpAddr, ok := remoteAddr.(*net.TCPAddr); ok {
			switch tcpAddr.Port {
			case 443:
				logger.Info("Detected HTTPS request from remote port 443, using destination port 443")
				return 443
			case 80:
				logger.Info("Detected HTTP request from remote port 80, using destination port 80")
				return 80
			}
		}
	}
	return 80
}
