package registry

// This package provides backward compatibility
// The actual Registry implementation has been moved to processor/registry

import "nursor.org/nursorgate/outbound"

// GetRegistry 获取全局代理注册中心（向后兼容）
func GetRegistry() *outbound.Registry {
	return outbound.GetRegistry()
}
