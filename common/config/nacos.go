package config

import (
	"fmt"

	"github.com/nacos-group/nacos-sdk-go/clients"
	"github.com/nacos-group/nacos-sdk-go/clients/config_client"
	"github.com/nacos-group/nacos-sdk-go/common/constant"
	"github.com/nacos-group/nacos-sdk-go/vo"
	"nursor.org/nursorgate/common/cache"
)

var nacosConfig *NacosConfig

type NacosConfig struct {
	client config_client.IConfigClient
}

// NewNacosClient initializes a Nacos configuration client.
// The cache and log directories are automatically set to ~/.nonelane (or NURSOR_CACHE_DIR if set)
// with 0777 permissions to allow all users to access cached configurations.
func NewNacosClient(endpoint, namespaceId string, port uint64) (*NacosConfig, error) {
	// Get the cache directory (creates it if needed)
	_, err := cache.GetCacheDir()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize cache directory: %w", err)
	}

	// Get or create nacos subdirectories
	nacosCache, err := cache.GetCacheSubdir("nacos/cache")
	if err != nil {
		return nil, fmt.Errorf("failed to initialize nacos cache directory: %w", err)
	}

	nacosLog, err := cache.GetCacheSubdir("nacos/log")
	if err != nil {
		return nil, fmt.Errorf("failed to initialize nacos log directory: %w", err)
	}

	// Create ServerConfig
	sc := []constant.ServerConfig{
		{
			Scheme: "https",
			IpAddr: endpoint,
			Port:   port,
		},
	}

	// Create ClientConfig with cache and log directories
	cc := constant.ClientConfig{
		NamespaceId:         namespaceId, // 命名空间ID
		TimeoutMs:           5000,
		NotLoadCacheAtStart: false,      // Allow loading from cache for better performance
		LogDir:              nacosLog,   // Use ~/.nonelane/nacos/log
		CacheDir:            nacosCache, // Use ~/.nonelane/nacos/cache
		CustomLogger:        nil,
		LogLevel:            "error",
	}

	// Create configuration client
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
