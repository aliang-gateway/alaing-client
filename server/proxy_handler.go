package server

import (
	"net/http"

	proxyRegistry "nursor.org/nursorgate/processor/proxy"
)

// handleGetCurrentProxy 获取当前使用的代理
func handleGetCurrentProxy(w http.ResponseWriter, r *http.Request) {
	registry := proxyRegistry.GetRegistry()
	currentName := registry.GetDefaultName()
	proxy, err := registry.GetDefault()

	if err != nil {
		sendError(w, "No proxy set", http.StatusNotFound, nil)
		return
	}

	sendResponse(w, map[string]interface{}{
		"name": currentName,
		"type": proxy.Proto().String(),
		"addr": proxy.Addr(),
	})
}

// handleSetCurrentProxy 设置当前使用的代理
func handleSetCurrentProxy(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name string `json:"name"`
	}

	if err := decodeRequest(r, &req); err != nil {
		sendError(w, "Invalid request body", http.StatusBadRequest, nil)
		return
	}

	if req.Name == "" {
		sendError(w, "name is required", http.StatusBadRequest, nil)
		return
	}

	registry := proxyRegistry.GetRegistry()
	if err := registry.SetDefault(req.Name); err != nil {
		sendError(w, err.Error(), http.StatusBadRequest, nil)
		return
	}

	proxy, _ := registry.GetDefault()
	sendResponse(w, map[string]interface{}{
		"name": req.Name,
		"type": proxy.Proto().String(),
		"addr": proxy.Addr(),
	})
}

// RegisterProxyRoutes 注册Proxy(当前代理)相关路由
func RegisterProxyRoutes() {
	http.HandleFunc("/proxy/current/get", handleGetCurrentProxy)
	http.HandleFunc("/proxy/current/set", handleSetCurrentProxy)
}
