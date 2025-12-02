package test

import (
	"testing"

	"nursor.org/nursorgate/runner"
	"nursor.org/nursorgate/server"
)

func TestLaunch(t *testing.T) {
	runner.Start()
}

func TestServer(t *testing.T) {
	server.StartHttpServer()
}
