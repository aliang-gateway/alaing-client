package tunnel

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"sync"
	"time"

	"nursor.org/nursorgate/client/inbound/cert"
	"nursor.org/nursorgate/client/server/helper"
	"nursor.org/nursorgate/client/server/tun/buffer"
	"nursor.org/nursorgate/client/server/tun/core/adapter"
	"nursor.org/nursorgate/client/server/tun/tunnel/statistic"
	"nursor.org/nursorgate/common/logger"
	"nursor.org/nursorgate/common/model"

	M "nursor.org/nursorgate/client/server/tun/metadata"
)

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

	var remoteConn net.Conn
	var err error
	var newOriginConn net.Conn

	if metadata.DstPort == 443 {
		serverName, sniBuf, err := helper.ExtractSNI(originConn)
		nursorRouter := model.NewAllowProxyDomain()
		if err != nil {
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
				tlsConf := cert.CreateTlsConfigForHost(serverName)
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
				if helper.IsCursorProxyEnabled {
					handleTlsConnect(tlsConn, req)
				} else {
					remoteConn, err = GetDoorProxy().DialContextWithServerName(ctx, metadata, serverName)
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
			remoteConn, err = GetDoorProxy().DialContextWithServerName(ctx, metadata, serverName)
			if err != nil {
				logger.Error(fmt.Sprintf("failure in connenct to anydoor %v", err))
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
