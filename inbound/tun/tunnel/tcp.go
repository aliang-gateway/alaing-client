package tunnel

import (
	"context"
	"fmt"

	"nursor.org/nursorgate/common/logger"
	"nursor.org/nursorgate/inbound/tun/adapter"
	M "nursor.org/nursorgate/inbound/tun/metadata"
	tcphandler "nursor.org/nursorgate/processor/tcp"
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

	// Handler should always be available - legacy fallback removed
	logger.Error("TCP handler not available")
}
