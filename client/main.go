package main

import (
	"os"
	"os/signal"
	"syscall"

	"nursor.org/nursorgate/client/utils"

	_ "github.com/sagernet/reality"
	_ "github.com/sagernet/sing-box"
	"nursor.org/nursorgate/client/server"
	"nursor.org/nursorgate/common/logger"
	"nursor.org/nursorgate/common/model"
)

func main() {
	// RunWindowsDesktop()
	RunBackground()
}

func RunBackground() {
	go server.StartHttpServer()
	model.NewAllowProxyDomain()
	logger.SetLogLevel(logger.DEBUG)

	utils.SetServerHost("test-ai-gateway.nursor.org:18889")

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh
}
