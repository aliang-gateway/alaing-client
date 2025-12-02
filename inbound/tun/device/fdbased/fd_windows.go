package fdbased

import (
	"errors"

	"nursor.org/nursorgate/inbound/tun/device"
)

func Open(name string, mtu uint32, offset int) (device.Device, error) {
	return nil, errors.ErrUnsupported
}
