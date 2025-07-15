package test

import (
	"context"
	"nursor.org/nursorgate/client/server/tun/metadata"
	"nursor.org/nursorgate/client/server/tun/proxy"
	"testing"
)

func TestHysteria1(t *testing.T) {
	d, err := proxy.NewHysteriaDialer("lisi", "IW6gUxtuG46FURELO08p9L9I3GtHtfh1")
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	m1 := metadata.Metadata{}
	d.DialContext(ctx, &m1)
	return
}
