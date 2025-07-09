package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"nursor.org/nursorgate/client/utils"

	"nursor.org/nursorgate/client/server"
	"nursor.org/nursorgate/common/config"
	"nursor.org/nursorgate/common/model"
)

func main() {
	// RunWindowsDesktop()
	RunBackground()
}

func RunBackground() {
	go server.StartHttpServer()
	nacosClient, err := config.NewNacosClient(
		"http://nacos-config.nursor.org",
		"5afe4eb9-d3ee-4b37-a072-7ea04421467a",
		80,
	)
	if err != nil {
		panic("failed to create nacos client: " + err.Error())
	}
	allowDomain := model.NewAllowProxyDomain()
	err = allowDomain.SyncFromNacos(
		nacosClient.GetConfigClient(),
		"nursor-user-door", // 配置ID
		"DEFAULT_GROUP",    // 配置分组
	)
	if err != nil {
		fmt.Println(err.Error())
	}
	//utils.SetServerHost("ai-gateway.nursor.org:8889")
	utils.SetServerHost("192.140.163.38:12235")

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh
}
