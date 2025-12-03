package http

import (
	"context"
	"fmt"
	"net"

	"nursor.org/nursorgate/common/logger"
	M "nursor.org/nursorgate/inbound/tun/metadata"
	"nursor.org/nursorgate/processor/tcp"
)

// HandleCONNECTTunnel handles HTTP CONNECT tunneling
// It delegates to processor/tcp for unified TCP handling with routing decisions
func HandleCONNECTTunnel(clientConn net.Conn, metadata *M.Metadata) error {
	logger.Debug(fmt.Sprintf("CONNECT tunnel: hostname=%s, port=%d, dstIP=%s, srcIP=%s",
		metadata.HostName, metadata.DstPort, metadata.DstIP.String(), metadata.SrcIP.String()))

	// Create context for the handler
	ctx := context.Background()

	// Get the unified TCP handler
	handler := tcp.GetHandler()

	// Delegate to processor/tcp for routing and relay
	// The handler will:
	// 1. Detect protocol (TLS on 443, HTTP on 80, direct for others)
	// 2. Route based on domain rules (cursor proxy, door proxy, or direct)
	// 3. Handle SNI extraction if TLS
	// 4. Perform bidirectional relay with statistics
	logger.Debug(fmt.Sprintf("Routing CONNECT through TCP handler for %s:%d", metadata.HostName, metadata.DstPort))
	if err := handler.Handle(ctx, clientConn, metadata); err != nil {
		logger.Error(fmt.Sprintf("TCP handler failed for %s: %v", metadata.HostName, err))
		return err
	}

	logger.Debug(fmt.Sprintf("CONNECT tunnel closed: %s:%d", metadata.HostName, metadata.DstPort))
	return nil
}
