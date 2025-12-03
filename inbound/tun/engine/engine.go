package engine

import (
	"errors"
	"fmt"
	"net"
	"net/netip"
	"time"

	"github.com/sagernet/gvisor/pkg/tcpip/stack"
	"nursor.org/nursorgate/common/logger"
	"nursor.org/nursorgate/outbound/proxy"
	"nursor.org/nursorgate/outbound/proxy/vless"

	"nursor.org/nursorgate/inbound/tun"
	"nursor.org/nursorgate/inbound/tun/device"
	"nursor.org/nursorgate/inbound/tun/dialer"
	"nursor.org/nursorgate/inbound/tun/option"
	"nursor.org/nursorgate/inbound/tun/tunnel"
	"nursor.org/nursorgate/outbound/proxy/direct"
	user "nursor.org/nursorgate/processor/auth"
	config "nursor.org/nursorgate/processor/config"
	proxyConfig "nursor.org/nursorgate/processor/config"
	proxyRegistry "nursor.org/nursorgate/processor/proxy"
	"nursor.org/nursorgate/runner/utils"

	"github.com/docker/go-units"
	"github.com/google/shlex"
)

var (

	// _defaultProxy holds the default proxy for the engine.
	_defaultProxy proxy.Proxy

	// _defaultDevice holds the default device for the engine.
	_defaultDevice device.Device

	// _defaultStack holds the default stack for the engine.
	_defaultStack *stack.Stack
)

// Start starts the default engine up.
func Start() error {
	if err := start(); err != nil {
		logger.Error(fmt.Sprintf("[ENGINE] failed to start: %v", err))
		return err
	}
	return nil
}

// Stop shuts the default engine down.
func Stop() {
	if err := stop(); err != nil {
		logger.Error(fmt.Sprintf("[ENGINE] failed to stop: %v", err))
	}
}

func start() error {
	if config.GetDefaultEngineConf() == nil {
		return errors.New("empty key")
	}

	for _, f := range []func(*config.EngineConf) error{
		general,
		netstack,
	} {
		if err := f(config.GetDefaultEngineConf()); err != nil {
			return err
		}
	}
	return nil
}

func stop() (err error) {
	if _defaultDevice != nil {
		_defaultDevice.Close()
	}
	if _defaultStack != nil {
		_defaultStack.Close()
		_defaultStack.Wait()
	}
	return nil
}

func execCommand(cmd string) error {
	parts, err := shlex.Split(cmd)
	if err != nil {
		return err
	}
	if len(parts) == 0 {
		return errors.New("empty command")
	}
	// cmds := exec.Command(parts[0], parts[1:]...)
	// cmds.SysProcAttr = &syscall.SysProcAttr{
	// 	HideWindow: true,
	// }
	// _, err = cmds.Output()
	err = utils.RunCommand(parts[0], parts[1:]...)
	return err
}

func general(k *config.EngineConf) error {
	//TODO: Auto here
	if k.Interface != "" {
		iface, err := net.InterfaceByName(k.Interface)
		if err != nil {
			return err
		}
		dialer.DefaultInterfaceName.Store(iface.Name)
		dialer.DefaultInterfaceIndex.Store(int32(iface.Index))
	}

	if k.Mark != 0 {
		dialer.DefaultRoutingMark.Store(int32(k.Mark))

	}

	if k.UDPTimeout > 0 {
		if k.UDPTimeout < time.Second {
			return errors.New("invalid udp timeout value")
		}
		tunnel.T().SetUDPTimeout(k.UDPTimeout)
	}
	return nil
}

func netstack(k *config.EngineConf) (err error) {
	if k.Device == "" {
		return errors.New("empty device")
	}

	if k.TUNPreUp != "" {
		print(fmt.Sprintf("[TUN] pre-execute command: `%s`", k.TUNPreUp))
		if preUpErr := execCommand(k.TUNPreUp); preUpErr != nil {
			logger.Info(fmt.Sprintf("[TUN] failed to pre-execute: %s: %v", k.TUNPreUp, preUpErr))
		}
	}

	defer func() {
		if k.TUNPostUp == "" || err != nil {
			return
		}
		print(fmt.Sprintf("[TUN] post-execute command: `%s`", k.TUNPostUp))
		if postUpErr := execCommand(k.TUNPostUp); postUpErr != nil {
			logger.Info(fmt.Sprintf("[TUN] failed to post-execute: %s: %v", k.TUNPostUp, postUpErr))
		}
	}()

	// 优先从注册中心获取默认代理
	_defaultProxy, err = proxyRegistry.GetRegistry().GetDefault()
	if err != nil {
		// 如果注册中心没有，尝试从配置管理器获取
		if defaultProxyFromConfig := proxyConfig.GetDirectProxy(); defaultProxyFromConfig != nil {
			_defaultProxy = defaultProxyFromConfig
		} else {
			// 最后使用直连代理作为后备
			_defaultProxy = direct.NewDirect()
			logger.Warn("No proxy configured, using direct connection")
		}
	}
	// 优先使用配置管理器中的代理
	if defaultProxyFromConfig := proxyConfig.GetDirectProxy(); defaultProxyFromConfig != nil {
		tunnel.SetDefaultProxy(defaultProxyFromConfig)
	} else {
		tunnel.SetDefaultProxy(_defaultProxy)
	}

	// 优先使用配置管理器中的门代理
	var doorProxy proxy.Proxy
	if doorProxyFromConfig := proxyConfig.GetDoorProxy(); doorProxyFromConfig != nil {
		doorProxy = doorProxyFromConfig
		tunnel.SetDoorProxy(doorProxyFromConfig)
	} else {
		// 如果没有配置，使用默认的 VLESS 配置（向后兼容）
		uuid := user.GetUserUUID()
		if uuid == "" {
			uuid = "74cddcdd-6d48-41cf-8e62-902e7c943fe7"
		}
		var err error
		doorProxy, err = vless.NewVLESSWithReality(
			"node1.nursor.org:35001",
			uuid,
			"www.microsoft.com",
			"sAtJcW2xLIUWRE-_7KHGEAtvHx-P1sDbjrrgrt4_XCo",
		)
		if err != nil {
			logger.Error(err)
		} else {
			tunnel.SetDoorProxy(doorProxy)
		}
	}

	// 确保 doorProxy 不为 nil 再创建 DNS resolver
	if doorProxy != nil {
		defaultResolver := tunnel.NewDNSResolver("8.8.8.8:53", doorProxy, 5*time.Second, 5*time.Minute)
		tunnel.SetDefaultResolver(defaultResolver)
	} else {
		logger.Warn("Door proxy is nil, DNS resolver not created")
	}

	tunnel.T().SetDialer(_defaultProxy)

	if _defaultDevice, err = parseDevice(k.Device, uint32(k.MTU)); err != nil {
		return err
	}

	var multicastGroups []netip.Addr
	if multicastGroups, err = parseMulticastGroups(k.MulticastGroups); err != nil {
		return err
	}

	var opts []option.Option
	if k.TCPModerateReceiveBuffer {
		opts = append(opts, option.WithTCPModerateReceiveBuffer(true))
	}

	if k.TCPSendBufferSize != "" {
		size, err := units.RAMInBytes(k.TCPSendBufferSize)
		if err != nil {
			return err
		}
		opts = append(opts, option.WithTCPSendBufferSize(int(size)))
	}

	if k.TCPReceiveBufferSize != "" {
		size, err := units.RAMInBytes(k.TCPReceiveBufferSize)
		if err != nil {
			return err
		}
		opts = append(opts, option.WithTCPReceiveBufferSize(int(size)))
	}

	if _defaultStack, err = tun.CreateStack(&tun.Config{
		LinkEndpoint:     _defaultDevice,
		TransportHandler: tunnel.T(),
		MulticastGroups:  multicastGroups,
		Options:          opts,
	}); err != nil {
		return
	}

	logger.Info(
		fmt.Sprintf("[STACK] %s://%s <-> %s://%s",
			_defaultDevice.Type(), _defaultDevice.Name(),
			_defaultProxy.Proto(), _defaultProxy.Addr(),
		),
	)
	return nil
}
