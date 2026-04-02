package tunnel

import (
	"context"
	"fmt"
	"runtime/debug"
	"sync"
	"time"

	"aliang.one/nursorgate/common/logger"
	"aliang.one/nursorgate/inbound/tun/adapter"
	"aliang.one/nursorgate/outbound/proxy"
	"aliang.one/nursorgate/processor/statistic"

	"go.uber.org/atomic"
)

const (
	// tcpConnectTimeout is the default timeout for TCP handshakes.
	tcpConnectTimeout = 30 * time.Second
	// tcpWaitTimeout implements a TCP half-close timeout.
	tcpWaitTimeout = 60 * time.Second
	// udpSessionTimeout is the default timeout for UDP sessions.
	udpSessionTimeout = 60 * time.Second

	// tcpWorkerCount is the number of TCP worker goroutines
	tcpWorkerCount = 100
	// udpWorkerCount is the number of UDP worker goroutines
	udpWorkerCount = 50
)

var _ adapter.TransportHandler = (*Tunnel)(nil)

type Tunnel struct {
	// Unbuffered TCP/UDP queues.
	tcpQueue chan adapter.TCPConn
	udpQueue chan adapter.UDPConn

	// UDP session timeout.
	udpTimeout *atomic.Duration

	// Internal proxy.Dialer for Tunnel.
	dialerMu sync.RWMutex
	dialer   proxy.Dialer

	// Where the Tunnel statistics are sent to.
	manager *statistic.Manager

	procOnce   sync.Once
	procCancel context.CancelFunc
}

func New(dialer proxy.Dialer, manager *statistic.Manager) *Tunnel {
	return &Tunnel{
		tcpQueue:   make(chan adapter.TCPConn, 512),
		udpQueue:   make(chan adapter.UDPConn, 128),
		udpTimeout: atomic.NewDuration(udpSessionTimeout),
		dialer:     dialer,
		manager:    manager,
		procCancel: func() { /* nop */ },
	}
}

// TCPIn return fan-in TCP queue.
func (t *Tunnel) TCPIn() chan<- adapter.TCPConn {
	return t.tcpQueue
}

// UDPIn return fan-in UDP queue.
func (t *Tunnel) UDPIn() chan<- adapter.UDPConn {
	return t.udpQueue
}

func (t *Tunnel) HandleTCP(conn adapter.TCPConn) {
	t.TCPIn() <- conn
}

func (t *Tunnel) HandleUDP(conn adapter.UDPConn) {
	t.UDPIn() <- conn
}

func (t *Tunnel) process(ctx context.Context) {
	defer func() {
		if r := recover(); r != nil {
			logger.Error(fmt.Sprintf("Recovered from panic in Tunnel.process: %v", r))
			debug.PrintStack()
		}
	}()

	// Start TCP worker pool
	for i := 0; i < tcpWorkerCount; i++ {
		go func(workerID int) {
			defer func() {
				if r := recover(); r != nil {
					logger.Error(fmt.Sprintf("TCP worker %d panic: %v", workerID, r))
					debug.PrintStack()
				}
			}()
			for {
				select {
				case conn := <-t.tcpQueue:
					t.handleTCPConn(conn)
				case <-ctx.Done():
					return
				}
			}
		}(i)
	}

	// Start UDP worker pool
	for i := 0; i < udpWorkerCount; i++ {
		go func(workerID int) {
			defer func() {
				if r := recover(); r != nil {
					logger.Error(fmt.Sprintf("UDP worker %d panic: %v", workerID, r))
					debug.PrintStack()
				}
			}()
			for {
				select {
				case conn := <-t.udpQueue:
					t.handleUDPConn(conn)
				case <-ctx.Done():
					return
				}
			}
		}(i)
	}

	// Wait for context cancellation
	<-ctx.Done()
}

// ProcessAsync can be safely called multiple times, but will only be effective once.
func (t *Tunnel) ProcessAsync() {
	t.procOnce.Do(func() {
		ctx, cancel := context.WithCancel(context.Background())
		t.procCancel = cancel
		go t.process(ctx)
	})
}

// Close closes the Tunnel and releases its resources.
func (t *Tunnel) Close() {
	t.procCancel()
}

func (t *Tunnel) Dialer() proxy.Dialer {
	t.dialerMu.RLock()
	d := t.dialer
	t.dialerMu.RUnlock()
	return d
}

func (t *Tunnel) SetDialer(dialer proxy.Dialer) {
	t.dialerMu.Lock()
	t.dialer = dialer
	t.dialerMu.Unlock()
}

func (t *Tunnel) SetUDPTimeout(timeout time.Duration) {
	t.udpTimeout.Store(timeout)
}
