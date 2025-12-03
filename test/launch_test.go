package test

import (
	"testing"

	httpServer "nursor.org/nursorgate/app/http"
	"nursor.org/nursorgate/runner"
)

func TestLaunch(t *testing.T) {
	runner.Start()
}

func TestServer(t *testing.T) {
	httpServer.StartHttpServer()
}
