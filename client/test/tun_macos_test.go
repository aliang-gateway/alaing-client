package test

import (
	"fmt"
	"log"
	"testing"

	"nursor.org/nursorgate/client/server/tun"
	mytun "nursor.org/nursorgate/client/server/tun"
	"nursor.org/nursorgate/client/utils"
	"nursor.org/nursorgate/common/config"
	"nursor.org/nursorgate/common/model"
)

func TestCreatTun(t *testing.T) {
	nacosClient, err := config.NewNacosClient(
		"http://local-nacos-config.nursor.org",
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
	utils.SetServerHost("ai-gateway.nursor.org:8889")

	tun.Start()
}

func TestTunMacos3(t *testing.T) {
	gw, err := mytun.GetDefaultGateway()
	if err != nil {
		log.Fatalf("Failed to get default gateway: %v", err)
	}
	log.Printf("Default gateway: %s", gw)
}
