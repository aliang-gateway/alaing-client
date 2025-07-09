package config

import (
	"github.com/nacos-group/nacos-sdk-go/clients"
	"github.com/nacos-group/nacos-sdk-go/clients/config_client"
	"github.com/nacos-group/nacos-sdk-go/common/constant"
	"github.com/nacos-group/nacos-sdk-go/vo"
)

var nacosConfig *NacosConfig

type NacosConfig struct {
	client config_client.IConfigClient
}

// 初始化Nacos客户端
func NewNacosClient(endpoint, namespaceId string, port uint64) (*NacosConfig, error) {
	// 创建ServerConfig
	sc := []constant.ServerConfig{
		{
			Scheme: "https",
			IpAddr: endpoint,
			Port:   port,
		},
	}

	// 创建ClientConfig
	cc := constant.ClientConfig{
		NamespaceId:         namespaceId, // 命名空间ID
		TimeoutMs:           5000,
		NotLoadCacheAtStart: true,
		// LogDir:              "/dev/null",
		// CacheDir:            "/dev/null",
		CustomLogger: nil,
		LogLevel:     "error",
	}

	// 创建配置客户端
	client, err := clients.NewConfigClient(
		vo.NacosClientParam{
			ClientConfig:  &cc,
			ServerConfigs: sc,
		},
	)

	if err != nil {
		return nil, err
	}

	return &NacosConfig{
		client: client,
	}, nil
}

func (n *NacosConfig) GetConfigClient() config_client.IConfigClient {
	return n.client
}

func GetNacosConfig() *NacosConfig {
	if nacosConfig == nil {
		nacosConfig, err := NewNacosClient("local-nacos.nursor.org", "9976d63d-759b-491b-897a-df311cd8ebc5", 80)
		if err != nil {
			panic(err)
		}
		return nacosConfig
	}
	return nacosConfig
}
