package server

import (
	"nursor.org/nursorgate/client/server/tun"
)

func StartTun() error {
	go func() {
		tun.Start()
	}()
	return nil
}
