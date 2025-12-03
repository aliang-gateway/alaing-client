package tcp

import (
	"context"
	"io"
	"net"
	"sync"
	"time"

	"nursor.org/nursorgate/common/logger"
	"nursor.org/nursorgate/inbound/tun/buffer"
	M "nursor.org/nursorgate/inbound/tun/metadata"
)

const (
	// defaultRelayBufferSize is the buffer size for bidirectional relay
	// This matches the size used in inbound/tun/buffer
	defaultRelayBufferSize = 32 * 1024
)

// DefaultRelayManager implements the RelayManager interface.
// It handles bidirectional data piping between two connections with:
// - Concurrent unidirectional streams
// - Buffer pooling for efficiency
// - Graceful shutdown and cleanup
// - TCP half-close handling
type DefaultRelayManager struct {
	// No fields needed - buffer pooling is handled globally
}

// NewDefaultRelayManager creates a new relay manager
func NewDefaultRelayManager() *DefaultRelayManager {
	return &DefaultRelayManager{}
}

// Relay implements the RelayManager interface.
// It establishes bidirectional data flow between two connections.
func (r *DefaultRelayManager) Relay(ctx context.Context, originConn, remoteConn net.Conn, metadata *M.Metadata) error {
	// Use a WaitGroup to wait for both directions to complete
	wg := sync.WaitGroup{}
	wg.Add(2)

	// Start concurrent unidirectional streams
	go r.relayStream(originConn, remoteConn, "client->server", &wg, ctx)
	go r.relayStream(remoteConn, originConn, "server->client", &wg, ctx)

	// Wait for both directions to complete
	wg.Wait()

	return nil
}

// relayStream copies data from src to dst in one direction.
// It handles:
// - Buffer management and pooling
// - TCP half-close (CloseRead/CloseWrite)
// - Timeout handling
// - Error logging
func (r *DefaultRelayManager) relayStream(
	dst net.Conn,
	src net.Conn,
	direction string,
	wg *sync.WaitGroup,
	ctx context.Context,
) {
	defer wg.Done()

	// Get buffer from pool
	buf := buffer.Get(defaultRelayBufferSize)
	defer buffer.Put(buf)

	// Copy data with timeout handling
	_, err := io.CopyBuffer(dst, src, buf)
	if err != nil && err != io.EOF {
		logger.Debug("relay copy error [" + direction + "]: " + err.Error())
	}

	// Perform TCP half-close to signal end of stream
	// Try to close the read side on src (source stops sending)
	if cr, ok := src.(interface{ CloseRead() error }); ok {
		cr.CloseRead()
	}

	// Try to close the write side on dst (destination stops accepting)
	if cw, ok := dst.(interface{ CloseWrite() error }); ok {
		cw.CloseWrite()
	}

	// Set a read deadline so we don't wait forever for the other side to close
	dst.SetReadDeadline(time.Now().Add(time.Duration(DefaultTCPWaitTimeout) * time.Second))
}

// SimpleRelayFunc provides a simple pipe function compatible with existing code.
// It creates a default relay manager and uses it to relay data.
func SimpleRelayFunc(ctx context.Context, originConn, remoteConn net.Conn) {
	// Create a simple relay manager (for backward compatibility)
	manager := NewDefaultRelayManager()

	// Create minimal metadata for relay
	metadata := &M.Metadata{
		Network: M.TCP,
	}

	// Relay the connections (ignore error for backward compatibility)
	manager.Relay(ctx, originConn, remoteConn, metadata)
}

// PipeConnections is a convenience function that mimics the old pipe() function
// for backward compatibility during migration.
func PipeConnections(origin, remote net.Conn) {
	// Simple inline version without context
	wg := sync.WaitGroup{}
	wg.Add(2)

	go func() {
		defer wg.Done()
		buf := buffer.Get(defaultRelayBufferSize)
		defer buffer.Put(buf)

		io.CopyBuffer(remote, origin, buf)

		// Half-close on source side
		if cr, ok := origin.(interface{ CloseRead() error }); ok {
			cr.CloseRead()
		}
		// Half-close on destination side
		if cw, ok := remote.(interface{ CloseWrite() error }); ok {
			cw.CloseWrite()
		}
		remote.SetReadDeadline(time.Now().Add(time.Duration(DefaultTCPWaitTimeout) * time.Second))
	}()

	go func() {
		defer wg.Done()
		buf := buffer.Get(defaultRelayBufferSize)
		defer buffer.Put(buf)

		io.CopyBuffer(origin, remote, buf)

		// Half-close on source side
		if cr, ok := remote.(interface{ CloseRead() error }); ok {
			cr.CloseRead()
		}
		// Half-close on destination side
		if cw, ok := origin.(interface{ CloseWrite() error }); ok {
			cw.CloseWrite()
		}
		origin.SetReadDeadline(time.Now().Add(time.Duration(DefaultTCPWaitTimeout) * time.Second))
	}()

	wg.Wait()
}
