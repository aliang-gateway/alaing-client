package tcp

import (
	"bytes"
	"context"
	"io"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"aliang.one/nursorgate/common/logger"
	"aliang.one/nursorgate/inbound/tun/buffer"
	M "aliang.one/nursorgate/inbound/tun/metadata"
)

const (
	// defaultRelayBufferSize is the buffer size for bidirectional relay
	// This matches the size used in tun/buffer
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
func (r *DefaultRelayManager) Relay(ctx context.Context, originConn, remoteConn net.Conn, metadata *M.Metadata) (*RelayStats, error) {
	stats := &RelayStats{StartedAt: time.Now()}

	// Use a WaitGroup to wait for both directions to complete
	wg := sync.WaitGroup{}
	wg.Add(2)

	var firstResponseNano int64
	var firstResponseSet int32

	requestCapture := newPayloadCaptureBuffer(128 * 1024)
	responseCapture := newPayloadCaptureBuffer(128 * 1024)

	var clientToServerBytes int64
	var serverToClientBytes int64

	markFirstResponse := func() {
		if atomic.CompareAndSwapInt32(&firstResponseSet, 0, 1) {
			atomic.StoreInt64(&firstResponseNano, time.Now().UnixNano())
		}
	}

	// Start concurrent unidirectional streams
	go r.relayStream(remoteConn, originConn, "client->server", &wg, requestCapture, nil, &clientToServerBytes, ctx)
	go r.relayStream(originConn, remoteConn, "server->client", &wg, responseCapture, markFirstResponse, &serverToClientBytes, ctx)

	// Wait for both directions to complete
	wg.Wait()

	stats.CompletedAt = time.Now()
	if firstNs := atomic.LoadInt64(&firstResponseNano); firstNs > 0 {
		stats.FirstResponseAt = time.Unix(0, firstNs)
	}
	stats.ClientToServerByte = atomic.LoadInt64(&clientToServerBytes)
	stats.ServerToClientByte = atomic.LoadInt64(&serverToClientBytes)
	stats.RequestPayload = requestCapture.Bytes()
	stats.ResponsePayload = responseCapture.Bytes()

	return stats, nil
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
	payloadCapture *payloadCaptureBuffer,
	onFirstData func(),
	byteCounter *int64,
	_ context.Context,
) {
	defer wg.Done()

	// Get buffer from pool
	buf := buffer.Get(defaultRelayBufferSize)
	defer buffer.Put(buf)

	// Copy data with timeout handling
	countingDst := &countingWriter{writer: dst, capture: payloadCapture, onFirstData: onFirstData}
	_, err := io.CopyBuffer(countingDst, src, buf)
	if byteCounter != nil {
		atomic.AddInt64(byteCounter, countingDst.written)
	}
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

type payloadCaptureBuffer struct {
	mu    sync.Mutex
	buf   bytes.Buffer
	limit int
}

func newPayloadCaptureBuffer(limit int) *payloadCaptureBuffer {
	if limit <= 0 {
		limit = 128 * 1024
	}
	return &payloadCaptureBuffer{limit: limit}
}

func (p *payloadCaptureBuffer) Write(data []byte) {
	if p == nil || len(data) == 0 {
		return
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	remaining := p.limit - p.buf.Len()
	if remaining <= 0 {
		return
	}

	if len(data) > remaining {
		data = data[:remaining]
	}
	_, _ = p.buf.Write(data)
}

func (p *payloadCaptureBuffer) Bytes() []byte {
	if p == nil {
		return nil
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	b := p.buf.Bytes()
	out := make([]byte, len(b))
	copy(out, b)
	return out
}

type countingWriter struct {
	writer      io.Writer
	capture     *payloadCaptureBuffer
	onFirstData func()
	written     int64
}

func (w *countingWriter) Write(p []byte) (int, error) {
	if w.onFirstData != nil && len(p) > 0 {
		w.onFirstData()
		w.onFirstData = nil
	}
	if w.capture != nil && len(p) > 0 {
		w.capture.Write(p)
	}
	n, err := w.writer.Write(p)
	if n > 0 {
		w.written += int64(n)
	}
	return n, err
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
	_, _ = manager.Relay(ctx, originConn, remoteConn, metadata)
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
