package test

import (
	"os"
	"testing"
	"time"

	"github.com/nacos-group/nacos-sdk-go/clients"
	"github.com/nacos-group/nacos-sdk-go/common/constant"
	"github.com/nacos-group/nacos-sdk-go/vo"
	"nursor.org/nursorgate/common/model"
)

func TestNacos(t *testing.T) {
	// 创建ServerConfig
	sc := []constant.ServerConfig{
		{
			IpAddr: "local-nacos.nursor.org",
			Port:   80,
			Scheme: "http",
		},
	}

	// 创建ClientConfig
	cc := constant.ClientConfig{
		TimeoutMs:           10000, // 增加超时时间
		NotLoadCacheAtStart: true,
		LogDir:              "./tmp/nacos/log", // 使用相对路径
		CacheDir:            "./tmp/nacos/cache",
		LogLevel:            "debug",
		Username:            "",                                     // 如果需要认证，添加用户名
		Password:            "",                                     // 如果需要认证，添加密码
		NamespaceId:         "9976d63d-759b-491b-897a-df311cd8ebc5", // 使用空字符串表示默认命名空间
	}

	// 确保日志和缓存目录存在
	os.MkdirAll("./tmp/nacos/log", 0755)
	os.MkdirAll("./tmp/nacos/cache", 0755)

	// 创建客户端
	client, err := clients.NewConfigClient(
		vo.NacosClientParam{
			ClientConfig:  &cc,
			ServerConfigs: sc,
		},
	)
	if err != nil {
		t.Fatalf("创建nacos客户端失败: %v", err)
	}

	// 先尝试发布配置
	success, err := client.PublishConfig(vo.ConfigParam{
		DataId:  "nursor-user-door",
		Group:   "DEFAULT_GROUP",
		Content: `{"toGateDomains":["example.com"],"toCursorDomain":["cursor.example.com"]}`,
	})
	if err != nil {
		t.Fatalf("发布配置失败: %v", err)
	}
	if !success {
		t.Fatal("发布配置返回失败")
	}

	// 等待一下确保配置已经生效
	time.Sleep(time.Second)

	// 获取配置
	content, err := client.GetConfig(vo.ConfigParam{
		DataId: "nursor-user-door",
		Group:  "DEFAULT_GROUP",
	})

	t.Logf("尝试连接Nacos服务器: %s:%d", sc[0].IpAddr, sc[0].Port)
	if err != nil {
		t.Fatalf("获取配置失败: %v", err)
	}

	t.Logf("获取到的配置内容: %s", content)
	t.Logf("尝试连接Nacos服务器: %s:%d", sc[0].IpAddr, sc[0].Port)
	if err != nil {
		t.Fatalf("获取配置失败: %v", err)
	}

	t.Logf("获取到的配置内容: %s", content)

	// 测试配置监听
	err = client.ListenConfig(vo.ConfigParam{
		DataId: "nursor-user-door",
		Group:  "DEFAULT_GROUP",
		OnChange: func(namespace, group, dataId, data string) {
			t.Logf("配置发生变化: %s", data)
		},
	})
	if err != nil {
		t.Fatalf("监听配置失败: %v", err)
	}

	allowDomain := &model.AllowProxyDomain{}
	err = allowDomain.SyncFromNacos(
		client,
		"nursor-user-door",
		"DEFAULT_GROUP",
	)
	if err != nil {
		t.Fatalf("同步配置失败: %v", err)
	}
	t.Logf("同步配置成功")
}
