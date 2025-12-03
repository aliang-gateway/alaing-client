package engine

import (
	"errors"
	"fmt"
	"net"
	"net/netip"
	"time"

	"github.com/sagernet/gvisor/pkg/tcpip/stack"
	"nursor.org/nursorgate/common/logger"
	"nursor.org/nursorgate/inbound/tun/runner/utils"
	"nursor.org/nursorgate/outbound/proxy"
	"nursor.org/nursorgate/outbound/proxy/direct"

	"github.com/docker/go-units"
	"github.com/google/shlex"
	"nursor.org/nursorgate/inbound/tun"
	"nursor.org/nursorgate/inbound/tun/device"
	"nursor.org/nursorgate/inbound/tun/dialer"
	"nursor.org/nursorgate/inbound/tun/option"
	"nursor.org/nursorgate/inbound/tun/tunnel"
	config "nursor.org/nursorgate/processor/config"
	proxyRegistry "nursor.org/nursorgate/processor/proxy"
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

	// 从注册中心获取默认代理
	_defaultProxy, err = proxyRegistry.GetRegistry().GetDefault()
	if err != nil {
		// 如果没有配置，使用直连代理作为后备
		_defaultProxy = direct.NewDirect()
		logger.Warn("No default proxy configured, using direct connection")
	}

	// 设置代理到 tunnel 的 dialer（用于 direct dialing）
	tunnel.T().SetDialer(_defaultProxy)

	// 从注册中心获取门代理（可选，用于 DNS 和特殊路由）
	doorProxy, err := proxyRegistry.GetRegistry().GetDoor()
	if err != nil {
		logger.Debug(fmt.Sprintf("No door proxy configured: %v", err))
		doorProxy = nil
	}

	// 如果有门代理，创建 DNS resolver
	if doorProxy != nil {
		defaultResolver := tunnel.NewDNSResolver("8.8.8.8:53", doorProxy, 5*time.Second, 5*time.Minute)
		tunnel.SetDefaultResolver(defaultResolver)
		logger.Info(fmt.Sprintf("DNS resolver configured with door proxy: %s", doorProxy.Addr()))
	}

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
