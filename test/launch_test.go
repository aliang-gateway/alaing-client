package test

import (
	"testing"

	httpServer "nursor.org/nursorgate/app/http"
	"nursor.org/nursorgate/inbound/http"
	"nursor.org/nursorgate/runner"
)

func TestLaunch(t *testing.T) {
	runner.Start()
}

func TestServer(t *testing.T) {
	httpServer.StartHttpServer()
}

func TestMitmHttp(t *testing.T) {
	http.StartMitmHttp()
}
