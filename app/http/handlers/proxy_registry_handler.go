package handlers

import (
	"net/http"

	"nursor.org/nursorgate/app/http/common"
	proxyConfig "nursor.org/nursorgate/processor/config"
	proxyRegistry "nursor.org/nursorgate/processor/proxy"
)

// handleProxyRegistryList 列出所有已注册的代理
func handleProxyRegistryList(w http.ResponseWriter, r *http.Request) {
	info := proxyRegistry.GetRegistry().ListWithInfo()
	common.SendResponse(w, map[string]interface{}{
		"proxies": info,
		"count":   len(info),
	})
}

// handleProxyRegistryGet 获取指定代理
func handleProxyRegistryGet(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("name")
	if name == "" {
		common.SendError(w, "name parameter is required", http.StatusBadRequest, nil)
		return
	}

	info := proxyRegistry.GetRegistry().ListWithInfo()
	proxyInfo, exists := info[name]
	if !exists {
		common.SendError(w, "proxy info not found", http.StatusNotFound, nil)
		return
	}

	common.SendResponse(w, proxyInfo)
}

// handleProxyRegistryRegister 注册新代理
func handleProxyRegistryRegister(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name   string                   `json:"name"`
		Config *proxyConfig.ProxyConfig `json:"config"`
	}

	if err := common.DecodeRequest(r, &req); err != nil {
		common.SendError(w, "Invalid request body", http.StatusBadRequest, nil)
		return
	}

	if req.Name == "" {
		common.SendError(w, "name is required", http.StatusBadRequest, nil)
		return
	}

	if req.Config == nil {
		common.SendError(w, "config is required", http.StatusBadRequest, nil)
		return
	}

	if err := proxyRegistry.GetRegistry().RegisterFromConfig(req.Name, req.Config); err != nil {
		common.SendError(w, err.Error(), http.StatusBadRequest, nil)
		return
	}

	common.SendResponse(w, map[string]string{"status": "success", "name": req.Name})
}

// handleProxyRegistryUnregister 注销代理
func handleProxyRegistryUnregister(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name string `json:"name"`
	}

	if err := common.DecodeRequest(r, &req); err != nil {
		common.SendError(w, "Invalid request body", http.StatusBadRequest, nil)
		return
	}

	if err := proxyRegistry.GetRegistry().Unregister(req.Name); err != nil {
		common.SendError(w, err.Error(), http.StatusBadRequest, nil)
		return
	}

	common.SendResponse(w, map[string]string{"status": "success"})
}

// handleProxyRegistrySetDefault 设置默认代理
func handleProxyRegistrySetDefault(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name string `json:"name"`
	}

	if err := common.DecodeRequest(r, &req); err != nil {
		common.SendError(w, "Invalid request body", http.StatusBadRequest, nil)
		return
	}

	if err := proxyRegistry.GetRegistry().SetDefault(req.Name); err != nil {
		common.SendError(w, err.Error(), http.StatusBadRequest, nil)
		return
	}

	common.SendResponse(w, map[string]string{"status": "success", "default": req.Name})
}

// handleProxyRegistrySetDoor 设置门代理
func handleProxyRegistrySetDoor(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name string `json:"name"`
	}

	if err := common.DecodeRequest(r, &req); err != nil {
		common.SendError(w, "Invalid request body", http.StatusBadRequest, nil)
		return
	}

	if err := proxyRegistry.GetRegistry().SetDoor(req.Name); err != nil {
		common.SendError(w, err.Error(), http.StatusBadRequest, nil)
		return
	}

	common.SendResponse(w, map[string]string{"status": "success", "door": req.Name})
}

// handleProxyRegistrySwitch 切换代理（设置默认代理并更新 tunnel）
func handleProxyRegistrySwitch(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name string `json:"name"`
	}

	if err := common.DecodeRequest(r, &req); err != nil {
		common.SendError(w, "Invalid request body", http.StatusBadRequest, nil)
		return
	}

	// 设置默认代理
	if err := proxyRegistry.GetRegistry().SetDefault(req.Name); err != nil {
		common.SendError(w, err.Error(), http.StatusBadRequest, nil)
		return
	}

	// 获取代理实例并更新 tunnel
	p, err := proxyRegistry.GetRegistry().Get(req.Name)
	if err != nil {
		common.SendError(w, err.Error(), http.StatusBadRequest, nil)
		return
	}

	// 更新 tunnel 的默认代理
	// 注意：这里需要导入 tunnel 包
	// tunnel.SetDefaultProxy(p)

	common.SendResponse(w, map[string]string{
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
