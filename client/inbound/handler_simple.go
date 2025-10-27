package inbound

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/netip"
	"strconv"
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
	// 从 req.Host 或 sni 中解析目标主机和端口
	targetHost := sni
	targetPort := uint16(443) // 默认端口

	// 优先从 req.Host 解析（CONNECT 请求的格式是 host:port）
	if req.Host != "" {
		if host, portStr, err := net.SplitHostPort(req.Host); err == nil {
			targetHost = host
			if port, err := strconv.ParseUint(portStr, 10, 16); err == nil {
				targetPort = uint16(port)
			}
		} else {
			// 没有端口号，使用整个 req.Host 作为主机名
			targetHost = req.Host
			// 根据 scheme 判断端口
			if req.URL != nil && req.URL.Scheme == "http" {
				targetPort = 80
			}
		}
	} else if sni != "" {
		// 从 sni 中解析
		if host, portStr, err := net.SplitHostPort(sni); err == nil {
			targetHost = host
			if port, err := strconv.ParseUint(portStr, 10, 16); err == nil {
				targetPort = uint16(port)
			}
		} else {
			targetHost = sni
		}
	}

	logger.Info("Target host:", targetHost, "port:", targetPort)
	logger.Info("Request details - Method:", req.Method, "Host:", req.Host, "SNI:", sni)

	// 解析域名到IP地址
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 使用默认DNS解析器
	resolver := tunnel.GetDefaultResolver()
	if resolver == nil {
		// 如果没有默认解析器，创建一个直连的解析器
		directDialer := proxy.NewDirect()
		resolver = tunnel.NewDNSResolver("223.5.5.5:53", directDialer, 5*time.Second, 5*time.Minute)
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

	outboundVless, err := proxy.NewVLESSWithReality(
		"node1.nursor.org:443",
		"d9868dc7-3547-4195-95f1-5455748e7706",
		"www.microsoft.com",
		"h1h7T-tqXyGaI0teh7i7kHu1qRLTT5HibTZcu30YtSs",
	)
	if err != nil {
		logger.Error(err.Error(), sni)
		return
	}

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

	// 创建Metadata结构体
	meta := &metadata.Metadata{
		Network:  metadata.TCP,
		DstIP:    dstIP,
		HostName: targetHost,
		DstPort:  targetPort,
	}

	logger.Info("Creating VLESS connection with metadata - Host:", targetHost, "IP:", dstIP, "Port:", targetPort)

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
