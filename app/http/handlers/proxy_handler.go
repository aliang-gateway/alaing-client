package handlers

import (
	"net/http"

	"nursor.org/nursorgate/app/http/common"
	proxyRegistry "nursor.org/nursorgate/outbound"
)

// HandleGetCurrentProxy 获取当前使用的代理
func HandleGetCurrentProxy(w http.ResponseWriter, r *http.Request) {
	registry := proxyRegistry.GetRegistry()
	currentName := registry.GetDefaultName()
	proxy, err := registry.GetDefault()

	if err != nil {
		common.SendError(w, "No proxy set", http.StatusNotFound, nil)
		return
	}

	common.SendResponse(w, map[string]interface{}{
		"name": currentName,
		"type": proxy.Proto().String(),
		"addr": proxy.Addr(),
	})
}

// HandleSetCurrentProxy 设置当前使用的代理
func HandleSetCurrentProxy(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name string `json:"name"`
	}

	if err := common.DecodeRequest(r, &req); err != nil {
		common.SendError(w, "Invalid request body", http.StatusBadRequest, nil)
		return
	}

	if req.Name == "" {
		common.SendError(w, "name is required", http.StatusBadRequest, nil)
		return
	}

	registry := proxyRegistry.GetRegistry()
	if err := registry.SetDefault(req.Name); err != nil {
		common.SendError(w, err.Error(), http.StatusBadRequest, nil)
		return
	}

	proxy, _ := registry.GetDefault()
	common.SendResponse(w, map[string]interface{}{
		"name": req.Name,
		"type": proxy.Proto().String(),
		"addr": proxy.Addr(),
	})
}

// RegisterProxyRoutes 注册Proxy(当前代理)相关路由
func RegisterProxyRoutes() {
	http.HandleFunc("/proxy/current/get", HandleGetCurrentProxy)
	http.HandleFunc("/proxy/current/set", HandleSetCurrentProxy)
}
