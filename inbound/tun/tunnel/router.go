package tunnel

import "nursor.org/nursorgate/outbound/proxy"

var (
	defaultProxy proxy.Proxy
	nursorProxy  *proxy.Proxy
)

func SetDefaultProxy(newProxy proxy.Proxy) {
	defaultProxy = newProxy
}

func SetDoorProxy(newProxy proxy.Proxy) {
	nursorProxy = &newProxy
}

func GetDefaultProxy() proxy.Proxy {
	return defaultProxy
}

func GetDoorProxy() *proxy.Proxy {
	return nursorProxy
}
