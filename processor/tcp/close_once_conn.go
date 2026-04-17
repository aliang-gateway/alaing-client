package tcp

import (
	"net"
	"sync"
)

type closeOnceConn struct {
	net.Conn

	closeOnce sync.Once
	closeErr  error
}

func wrapCloseOnceConn(conn net.Conn) net.Conn {
	if conn == nil {
		return nil
	}
	if _, ok := conn.(*closeOnceConn); ok {
		return conn
	}
	return &closeOnceConn{Conn: conn}
}

func (c *closeOnceConn) Close() error {
	if c == nil || c.Conn == nil {
		return nil
	}
	c.closeOnce.Do(func() {
		c.closeErr = c.Conn.Close()
	})
	return c.closeErr
}
