package fdbased

import (
	"errors"

	"nursor.org/nursorgate/client/server/tun/core/device"
)

func Open(name string, mtu uint32, offset int) (device.Device, error) {
	return nil, errors.ErrUnsupported
}
