//go:build unix

package engine

import (
	"net/url"
	"nursor.org/nursorgate/client/server/tun/core/device"
	"nursor.org/nursorgate/client/server/tun/core/device/tun"
)

func parseTUN(u *url.URL, mtu uint32) (device.Device, error) {
	return tun.Open(u.Host, mtu)
}
