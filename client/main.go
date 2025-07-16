package main

import (
	"os"
	"os/signal"
	"syscall"

	"nursor.org/nursorgate/client/utils"

	"nursor.org/nursorgate/client/server"
	"nursor.org/nursorgate/common/model"
)

func main() {
	// RunWindowsDesktop()
	RunBackground()
}

func RunBackground() {
	go server.StartHttpServer()
	//go server.StartMitmHttp()
	model.NewAllowProxyDomain()

	//utils.SetServerHost("ai-gateway.nursor.org:8889")
	utils.SetServerHost("192.140.163.38:12235")

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh
}
