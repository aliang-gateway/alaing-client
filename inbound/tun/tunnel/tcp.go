package tunnel

import (
	"bufio"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"nursor.org/nursorgate/common/logger"
	"nursor.org/nursorgate/common/model"
	"nursor.org/nursorgate/inbound/tun/adapter"
	"nursor.org/nursorgate/inbound/tun/buffer"
	M "nursor.org/nursorgate/inbound/tun/metadata"
	cert_client "nursor.org/nursorgate/processor/cert/client"
	proxyRegistry "nursor.org/nursorgate/processor/proxy"
	"nursor.org/nursorgate/processor/statistic"
	tcphandler "nursor.org/nursorgate/processor/tcp"
	tls_helper "nursor.org/nursorgate/processor/tls"
	watcher "nursor.org/nursorgate/processor/watcher"
)

// getTCPHandler safely gets the TCP handler, with error recovery
func getTCPHandler() tcphandler.TCPConnHandler {
	defer func() {
		if r := recover(); r != nil {
			logger.Error(fmt.Sprintf("Panic in getTCPHandler: %v", r))
		}
	}()
	return tcphandler.GetHandler()
}

func detectDoH(tlsConn *tls.Conn) bool {
	// 读取HTTP请求头
	reader := bufio.NewReader(tlsConn)
	line, _, _ := reader.ReadLine()

	// 检查请求路径
	if strings.Contains(string(line), "/dns-query") ||
		strings.Contains(string(line), "/resolve") {
		return true
	}
	return false
}

func isDoHProvider(serverName string) bool {
	dohProviders := []string{
		"dns.google",
		"cloudflare-dns.com",
		"doh.opendns.com",
		"doh.quad9.net",
		"doh.cleanbrowsing.org",
		"1.1.1.1",
		"8.8.8.8",
		"9.9.9.9",
	}

	for _, provider := range dohProviders {
		if strings.Contains(serverName, provider) {
			return true
		}
	}
	return false
}

func (t *Tunnel) handleTCPConn(originConn adapter.TCPConn) {
	defer originConn.Close()

	id := originConn.ID()
	metadata := &M.Metadata{
		Network: M.TCP,
		SrcIP:   parseTCPIPAddress(id.RemoteAddress),
		SrcPort: id.RemotePort,
		DstIP:   parseTCPIPAddress(id.LocalAddress),
		DstPort: id.LocalPort,
	}

	ctx, cancel := context.WithTimeout(context.Background(), tcpConnectTimeout)
	defer cancel()

	// Use unified TCP handler from processor/tcp
	handler := getTCPHandler()
	if handler != nil {
		err := handler.Handle(ctx, originConn, metadata)
		if err != nil {
			logger.Debug(fmt.Sprintf("TCP handler error: %v", err))
		}
		return
	}

	// Fallback to legacy implementation if handler not available
	logger.Debug("TCP handler not available, using legacy implementation")

	var remoteConn net.Conn
	var err error
	var newOriginConn net.Conn

	if metadata.DstPort == 443 {
		serverName, sniBuf, err := tls_helper.ExtractSNI(originConn)
		metadata.HostName = serverName
		nursorRouter := model.NewAllowProxyDomain()
		if err != nil {
			if isDoHProvider(serverName) {
				logger.Info("[DoH] 检测到DoH流量，目标:", serverName)
				// 特殊处理DoH流量
				return
			}
			logger.Debug("SNI extraction error:", err)
			newOriginConn = &WrappedConn{
				Buf:  sniBuf,
				Conn: originConn,
			}
		} else {
			logger.Debug("Extracted SNI:", serverName)
			var req = &http.Request{
				Host: serverName,
			}
			newOriginConn = &WrappedConn{
				Buf:  sniBuf,
				Conn: originConn,
			}

			if nursorRouter.IsAllowToCursor(serverName) {
				tlsConf := cert_client.CreateTlsConfigForHost(serverName)
				tlsConn := tls.Server(newOriginConn, tlsConf)
				if err := tlsConn.Handshake(); err != nil {
					if errors.Is(err, io.EOF) {
						logger.Debug("client close the handshake connection", serverName)
					} else {
						logger.Error(fmt.Sprintf("TLS handshake failed: %s, %v", serverName, err))
					}
					return
				}
				state := tlsConn.ConnectionState()
				logger.Debug("TLS handshake successful. Protocol:", state.NegotiatedProtocol, "Version:", state.Version)
				if watcher.IsCursorProxyEnabled {
					handleTlsConnect(tlsConn, req)
				} else {
					doorProxy, err := proxyRegistry.GetRegistry().GetDoor()
			if err != nil {
				logger.Error(fmt.Sprintf("door proxy not available: %v", err))
				return
			}
			remoteConn, err = doorProxy.DialContext(ctx, metadata)
					if err != nil {
						logger.Error(fmt.Sprintf("failure in connenct to anydoor %v", err))
						return
					}
					// watcherConn := helper.NewWatcherWrapConn(tlsConn)
					pipe(tlsConn, remoteConn)
				}
				return
			}
		}

		if nursorRouter.IsAllowToAnyDoor(serverName) {
			doorProxy, err := proxyRegistry.GetRegistry().GetDoor()
			if err != nil {
				logger.Error(fmt.Sprintf("door proxy not available: %v", err))
				return
			}
			remoteConn, err = doorProxy.DialContext(ctx, metadata)
			if err != nil {
				logger.Error(fmt.Sprintf("%s failure in connenct to anydoor %v", serverName, err))
				return
			}
		} else {
			// 直连
			remoteConn, err = t.Dialer().DialContext(ctx, metadata)
			if err != nil {
				logger.Debug(fmt.Printf("[TCP] dial %s: %v", metadata.DestinationAddress(), err))
				return
			}
		}

	} else {
		// 直连
		remoteConn, err = t.Dialer().DialContext(ctx, metadata)
		if err != nil {
			logger.Debug(fmt.Printf("[TCP] dial %s: %v", metadata.DestinationAddress(), err))
			return
		}
		newOriginConn = originConn
	}

	metadata.MidIP, metadata.MidPort = parseNetAddr(remoteConn.LocalAddr())

	remoteConn = statistic.NewTCPTracker(remoteConn, metadata, t.manager)
	defer remoteConn.Close()

	pipe(newOriginConn, remoteConn)
}

// pipe copies data to & from provided net.Conn(s) bidirectionally.
func pipe(origin, remote net.Conn) {
	wg := sync.WaitGroup{}
	wg.Add(2)

	go unidirectionalStream(remote, origin, "origin->remote", &wg)
	go unidirectionalStream(origin, remote, "remote->origin", &wg)

	wg.Wait()
}

func unidirectionalStream(dst, src net.Conn, dir string, wg *sync.WaitGroup) {
	defer wg.Done()
	buf := buffer.Get(buffer.RelayBufferSize)
	if _, err := io.CopyBuffer(dst, src, buf); err != nil {
		//log.Debugf("[TCP] copy data for %s: %v", dir, err)
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
