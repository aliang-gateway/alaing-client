package test

import (
	"testing"

	"golang.org/x/net/http2"
	"nursor.org/nursorgate/client/outbound"
)

func TestDirectHttp2(t *testing.T) {
	conn, err := outbound.NewDirectHttp2Client("www.grok.com:443")
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	conn.Write([]byte(http2.ClientPreface))
	buf := make([]byte, 4096)
	n, err := conn.Read(buf)
	if err != nil {
		t.Fatal(err)
	}
	respStr := string(buf[:n])
	t.Log(respStr)

}
