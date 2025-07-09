package test

import (
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"testing"
	"time"

	"golang.zx2c4.com/wireguard/tun"
	mytun "nursor.org/nursorgate/client/server/tun"
	"nursor.org/nursorgate/client/utils"
	"nursor.org/nursorgate/common/config"
	"nursor.org/nursorgate/common/model"
)

func TestTunWindows(t *testing.T) {
	// 定义 TUN 设备名称
	ifname := "MyTUN"

	// 创建 TUN 设备
	dev, err := tun.CreateTUN(ifname, 0)
	if err != nil {
		log.Fatalf("Failed to create TUN device: %v", err)
	}
	defer dev.Close()

	// 获取设备的 LUID
	nativeTunDevice, ok := dev.(*tun.NativeTun)
	if !ok {
		log.Fatalf("Device is not a NativeTun")
	}
	luid := nativeTunDevice.LUID()

	// 设置 IP 地址和子网掩码
	addr := &net.IPNet{
		IP:   net.ParseIP("10.0.0.1"),
		Mask: net.CIDRMask(24, 32), // 255.255.255.0
	}

	print(luid)

	log.Printf("TUN device %s created with IP %s", ifname, addr.String())

	// 保持程序运行以观察效果
	time.Sleep(30 * time.Second)
}

func TestTunWindows2(t *testing.T) {
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
	utils.SetServerHost("192.140.163.38:12235")
	mytun.Start()
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh
}
