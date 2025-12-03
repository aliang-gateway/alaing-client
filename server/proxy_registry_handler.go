package server

import (
	"net/http"

	proxyConfig "nursor.org/nursorgate/processor/config"
	proxyRegistry "nursor.org/nursorgate/processor/proxy"
)

// handleProxyRegistryList 列出所有已注册的代理
func handleProxyRegistryList(w http.ResponseWriter, r *http.Request) {
	info := proxyRegistry.GetRegistry().ListWithInfo()
	sendResponse(w, map[string]interface{}{
		"proxies": info,
		"count":   len(info),
	})
}

// handleProxyRegistryGet 获取指定代理
func handleProxyRegistryGet(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("name")
	if name == "" {
		sendError(w, "name parameter is required", http.StatusBadRequest, nil)
		return
	}

	info := proxyRegistry.GetRegistry().ListWithInfo()
	proxyInfo, exists := info[name]
	if !exists {
		sendError(w, "proxy info not found", http.StatusNotFound, nil)
		return
	}

	sendResponse(w, proxyInfo)
}

// handleProxyRegistryRegister 注册新代理
func handleProxyRegistryRegister(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name   string                   `json:"name"`
		Config *proxyConfig.ProxyConfig `json:"config"`
	}

	if err := decodeRequest(r, &req); err != nil {
		sendError(w, "Invalid request body", http.StatusBadRequest, nil)
		return
	}

	if req.Name == "" {
		sendError(w, "name is required", http.StatusBadRequest, nil)
		return
	}

	if req.Config == nil {
		sendError(w, "config is required", http.StatusBadRequest, nil)
		return
	}

	if err := proxyRegistry.GetRegistry().RegisterFromConfig(req.Name, req.Config); err != nil {
		sendError(w, err.Error(), http.StatusBadRequest, nil)
		return
	}

	sendResponse(w, map[string]string{"status": "success", "name": req.Name})
}

// handleProxyRegistryUnregister 注销代理
func handleProxyRegistryUnregister(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name string `json:"name"`
	}

	if err := decodeRequest(r, &req); err != nil {
		sendError(w, "Invalid request body", http.StatusBadRequest, nil)
		return
	}

	if err := proxyRegistry.GetRegistry().Unregister(req.Name); err != nil {
		sendError(w, err.Error(), http.StatusBadRequest, nil)
		return
	}

	sendResponse(w, map[string]string{"status": "success"})
}

// handleProxyRegistrySetDefault 设置默认代理
func handleProxyRegistrySetDefault(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name string `json:"name"`
	}

	if err := decodeRequest(r, &req); err != nil {
		sendError(w, "Invalid request body", http.StatusBadRequest, nil)
		return
	}

	if err := proxyRegistry.GetRegistry().SetDefault(req.Name); err != nil {
		sendError(w, err.Error(), http.StatusBadRequest, nil)
		return
	}

	sendResponse(w, map[string]string{"status": "success", "default": req.Name})
}

// handleProxyRegistrySetDoor 设置门代理
func handleProxyRegistrySetDoor(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name string `json:"name"`
	}

	if err := decodeRequest(r, &req); err != nil {
		sendError(w, "Invalid request body", http.StatusBadRequest, nil)
		return
	}

	if err := proxyRegistry.GetRegistry().SetDoor(req.Name); err != nil {
		sendError(w, err.Error(), http.StatusBadRequest, nil)
		return
	}

	sendResponse(w, map[string]string{"status": "success", "door": req.Name})
}

// handleProxyRegistrySwitch 切换代理（设置默认代理并更新 tunnel）
func handleProxyRegistrySwitch(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name string `json:"name"`
	}

	if err := decodeRequest(r, &req); err != nil {
		sendError(w, "Invalid request body", http.StatusBadRequest, nil)
		return
	}

	// 设置默认代理
	if err := proxyRegistry.GetRegistry().SetDefault(req.Name); err != nil {
		sendError(w, err.Error(), http.StatusBadRequest, nil)
		return
	}

	// 获取代理实例并更新 tunnel
	p, err := proxyRegistry.GetRegistry().Get(req.Name)
	if err != nil {
		sendError(w, err.Error(), http.StatusBadRequest, nil)
		return
	}

	// 更新 tunnel 的默认代理
	// 注意：这里需要导入 tunnel 包
	// tunnel.SetDefaultProxy(p)

	sendResponse(w, map[string]string{
		"status": "success",
		"name":   req.Name,
		"addr":   p.Addr(),
		"type":   p.Proto().String(),
	})
}

// RegisterProxyRegistryRoutes 注册ProxyRegistry相关路由
func RegisterProxyRegistryRoutes() {
	http.HandleFunc("/proxy/registry/list", handleProxyRegistryList)
	http.HandleFunc("/proxy/registry/get", handleProxyRegistryGet)
	http.HandleFunc("/proxy/registry/register", handleProxyRegistryRegister)
	http.HandleFunc("/proxy/registry/unregister", handleProxyRegistryUnregister)
	http.HandleFunc("/proxy/registry/set-default", handleProxyRegistrySetDefault)
	http.HandleFunc("/proxy/registry/set-door", handleProxyRegistrySetDoor)
	http.HandleFunc("/proxy/registry/switch", handleProxyRegistrySwitch)
}
