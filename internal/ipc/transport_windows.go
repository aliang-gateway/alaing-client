//go:build windows

package ipc

import (
	"fmt"
	"net"
	"os"
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
	// Remove existing pipe file if it exists
	os.Remove(t.path)

	// Create a listener on the named pipe
	listener, err := net.Listen("pipe", t.path)
	if err != nil {
		return nil, fmt.Errorf("failed to listen on named pipe %s: %w", t.path, err)
	}

	return listener, nil
}

func (t *windowsTransport) Dial() (net.Conn, error) {
	// Dial the named pipe
	conn, err := net.Dial("pipe", t.path)
	if err != nil {
		return nil, fmt.Errorf("failed to dial named pipe %s: %w", t.path, err)
	}
	return conn, nil
}

func (t *windowsTransport) SocketPath() string {
	return t.path
}
