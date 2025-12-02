package out

import (
	"crypto/tls"
)

// WrappedTLSConn 包装 tls.Conn，支持数据回放
type WrappedTLSConn struct {
	*tls.Conn        // 嵌入 tls.Conn
	Buf       []byte // 用于存储回放的数据
}

// Read 实现数据回放逻辑
func (w *WrappedTLSConn) Read(b []byte) (int, error) {
	if len(w.Buf) > 0 {
		n := copy(b, w.Buf)
		w.Buf = w.Buf[n:]
		if len(w.Buf) == 0 {
			w.Buf = nil
		}
		return n, nil
	}
	return w.Conn.Read(b)
}
