// Package tun provides TUN which implemented device.Device interface.
package tun

import (
	"nursor.org/nursorgate/client/server/tun/core/device"
)

const Driver = "tun"

func (t *TUN) Type() string {
	return Driver
}

var _ device.Device = (*TUN)(nil)
