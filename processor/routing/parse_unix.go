//go:build unix

package routing

import (
	"net/url"
	"nursor.org/nursorgate/inbound/tun/device"
	"nursor.org/nursorgate/inbound/tun/device/tun"
)

func parseTUN(u *url.URL, mtu uint32) (device.Device, error) {
	return tun.Open(u.Host, mtu)
}
