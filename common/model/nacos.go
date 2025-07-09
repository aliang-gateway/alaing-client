package model

import (
	"encoding/json"
	"fmt"
	"strings"

	"nursor.org/nursorgate/common/config"

	"github.com/nacos-group/nacos-sdk-go/clients/config_client"
	"github.com/nacos-group/nacos-sdk-go/vo"
)

var allowProxyDomain *AllowProxyDomain

type AllowProxyDomain struct {
	ToGateDomains  []string `json:"toGateDomains"`
	ToCursorDomain []string `json:"toCursorDomain"`
	DenyDomains    []string `json:"denyDomains"`
}

// 是否允许经过Gate，用作配置route的时候，windows上目前用不着
func (a *AllowProxyDomain) IsAllowToGate(domain string) bool {
	if len(a.ToGateDomains) == 0 {
		return strings.Contains(domain, "cursor")
	}
	for _, d := range a.ToGateDomains {
		if strings.Contains(domain, d) {
			return true
		}
	}
	return false
}

// 是否允许发送到我的nursorgate
func (a *AllowProxyDomain) IsAllowToCursor(domain string) bool {
	for _, d := range a.DenyDomains {
		if strings.Contains(domain, d) {
			return false
		}
	}
	if len(a.ToCursorDomain) == 0 {
		return strings.Contains(domain, "cursor")
	}
	for _, d := range a.ToCursorDomain {
		if strings.Contains(domain, d) {
			return true
		}
	}
	return false
}

func NewAllowProxyDomain() *AllowProxyDomain {
	if allowProxyDomain == nil {
		allowProxyDomain = &AllowProxyDomain{
			ToGateDomains:  []string{},
			DenyDomains:    []string{},
			ToCursorDomain: []string{"cursor.sh", "cursor.com"},
		}
		nacosClient, err := config.NewNacosClient(
			"http://local-nacos-config.nursor.org",
			"5afe4eb9-d3ee-4b37-a072-7ea04421467a",
			80,
		)
		err = allowProxyDomain.SyncFromNacos(
			nacosClient.GetConfigClient(),
			"nursor-user-door", // 配置ID
			"DEFAULT_GROUP",    // 配置分组
		)
		if err != nil {
			fmt.Println(err.Error())
		}
	}
	return allowProxyDomain
}

// 从Nacos获取配置并监听更新
func (a *AllowProxyDomain) SyncFromNacos(client config_client.IConfigClient, dataId, group string) error {
	// 监听配置变化
	err := client.ListenConfig(vo.ConfigParam{
		DataId: dataId,
		Group:  group,
		OnChange: func(namespace, group, dataId, data string) {
			var newConfig AllowProxyDomain
			if err := json.Unmarshal([]byte(data), &newConfig); err == nil {
				*a = newConfig
			}
		},
	})

	if err != nil {
		return err
	}

	// 获取初始配置
	content, err := client.GetConfig(vo.ConfigParam{
		DataId: dataId,
		Group:  group,
	})

	if err != nil {
		return err
	}

	return json.Unmarshal([]byte(content), a)
}
