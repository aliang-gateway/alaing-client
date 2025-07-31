package test

import (
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
	model.NewAllowProxyDomain()
	utils.SetServerHost("api2.nursor.org:12235")
	mytun.Start()
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh
}
