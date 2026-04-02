//go:build windows

package ipc

import (
	"fmt"
	"net"
	"time"

	"github.com/Microsoft/go-winio"
)

// windowsTransport implements Transport using Named Pipe.
type windowsTransport struct {
	path string
}

// NewTransport creates a new Named Pipe transport.
func NewTransport() Transport {
	return &windowsTransport{path: SocketPath()}
}

func (t *windowsTransport) Listen() (net.Listener, error) {
	listener, err := winio.ListenPipe(t.path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to listen on named pipe %s: %w", t.path, err)
	}

	return listener, nil
}

func (t *windowsTransport) Dial() (net.Conn, error) {
	timeout := 2 * time.Second
	conn, err := winio.DialPipe(t.path, &timeout)
	if err != nil {
		return nil, fmt.Errorf("failed to dial named pipe %s: %w", t.path, err)
	}
	return conn, nil
}

func (t *windowsTransport) SocketPath() string {
	return t.path
}
