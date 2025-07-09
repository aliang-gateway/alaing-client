package tunnel

import "nursor.org/nursorgate/client/server/tun/proxy"

var (
	defaultProxy proxy.Proxy
	nursorProxy  proxy.Proxy
)

func SetDefaultProxy(newProxy proxy.Proxy) {
	defaultProxy = newProxy
}

func SetNursorProxy(newProxy proxy.Proxy) {
	nursorProxy = newProxy
}

func GetDefaultProxy() proxy.Proxy {
	return defaultProxy
}

func GetNursorProxy() proxy.Proxy {
	return nursorProxy
}
