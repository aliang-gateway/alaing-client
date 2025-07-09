package inbound

import (
	"bytes"
	"io"

	"github.com/nacos-group/nacos-sdk-go/common/logger"
)

// loggingReader 记录读取的原始数据
type loggingReader struct {
	r      io.Reader
	buffer bytes.Buffer // 存储最近读取的数据
}

func (lr *loggingReader) Read(p []byte) (n int, err error) {
	n, err = lr.r.Read(p)
	if n > 0 {
		//lr.buffer.Write()
		lr.buffer.Write(p[:n])
		logger.Infof("Read raw data (%d bytes): %x", n, p[:n])
	}
	return n, err
}

func (lr *loggingReader) Bytes() []byte {
	return lr.buffer.Bytes()
}

// loggingWriter 记录写入的原始数据
type loggingWriter struct {
	w      io.Writer
	buffer bytes.Buffer // 存储最近写入的数据
}

func (lw *loggingWriter) Write(p []byte) (n int, err error) {
	n, err = lw.w.Write(p)
	if n > 0 {
		lw.buffer.Write(p[:n])
		logger.Infof("Wrote raw data (%d bytes): %x", n, p[:n])
	}
	return n, err
}

func (lw *loggingWriter) Bytes() []byte {
	return lw.buffer.Bytes()
}
