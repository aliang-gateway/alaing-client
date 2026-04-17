package tcp

import (
	"errors"
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

func (c *closeOnceConn) CloseRead() error {
	if c == nil || c.Conn == nil {
		return nil
	}
	if cr, ok := c.Conn.(interface{ CloseRead() error }); ok {
		return cr.CloseRead()
	}
	return errors.New("CloseRead is not implemented")
}

func (c *closeOnceConn) CloseWrite() error {
	if c == nil || c.Conn == nil {
		return nil
	}
	if cw, ok := c.Conn.(interface{ CloseWrite() error }); ok {
		return cw.CloseWrite()
	}
	return errors.New("CloseWrite is not implemented")
}
