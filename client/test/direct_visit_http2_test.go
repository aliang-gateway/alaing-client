package test

import (
	"crypto/tls"
	"net"
	"testing"

	"golang.org/x/net/http2"
	"nursor.org/nursorgate/client/inbound/cert"
	"nursor.org/nursorgate/common/logger"
)

func TestDirectVisitHttp3000(t *testing.T) {
	conn, err := net.Dial("tcp", "172.16.202.34:3000")
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	tlsConf := cert.CreateTlsConfigForHost("172.16.202.34")
	tlsConn := tls.Client(conn, tlsConf)
	if err := tlsConn.Handshake(); err != nil {
		logger.Error("TLS handshake with client failed:", err)
		return
	}
	f := http2.NewFramer(tlsConn, tlsConn)
	f.WriteSettings(http2.Setting{ID: http2.SettingMaxConcurrentStreams, Val: 100})
	f.WriteSettingsAck()
	fr, err := f.ReadFrame()
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := fr.(*http2.SettingsFrame); !ok {
		t.Fatal("not settings frame")
	}
	logger.Info("read settings frame")

}
