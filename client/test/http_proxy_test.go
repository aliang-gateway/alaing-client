package test

import (
	"fmt"
	"testing"

	"nursor.org/nursorgate/client/server"
	"nursor.org/nursorgate/client/utils"
	"nursor.org/nursorgate/common/config"
	"nursor.org/nursorgate/common/model"
)

func TestHttpProxy(t *testing.T) {
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
	utils.SetServerHost("127.0.0.1:8082")
	server.StartMitmHttpSimple()
}
