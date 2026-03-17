package tunnel

import "nursor.org/nursorgate/outbound/proxy"

var (
	defaultProxy proxy.Proxy
)

func SetDefaultProxy(newProxy proxy.Proxy) {
	defaultProxy = newProxy
}

func GetDefaultProxy() proxy.Proxy {
	return defaultProxy
}
