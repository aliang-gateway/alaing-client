package tunnel

import "nursor.org/nursorgate/client/server/tun/proxy"

var (
	defaultProxy proxy.Proxy
	nursorProxy  *proxy.HysteriaDialer
)

func SetDefaultProxy(newProxy proxy.Proxy) {
	defaultProxy = newProxy
}

func SetDoorProxy(newProxy *proxy.HysteriaDialer) {
	nursorProxy = newProxy
}

func GetDefaultProxy() proxy.Proxy {
	return defaultProxy
}

func GetDoorProxy() *proxy.HysteriaDialer {
	return nursorProxy
}
