package server

import (
	"net/http"

	proxyConfig "nursor.org/nursorgate/processor/config"
)

// handleConfigGet 获取存储的代理配置
func handleConfigGet(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("name")
	if name == "" {
		sendError(w, "name parameter is required", http.StatusBadRequest, nil)
		return
	}

	cfg, err := proxyConfig.GetConfigStore().Get(name)
	if err != nil {
		sendError(w, err.Error(), http.StatusNotFound, nil)
		return
	}

	sendResponse(w, cfg)
}

// handleConfigList 列出所有存储的代理配置
func handleConfigList(w http.ResponseWriter, r *http.Request) {
	store := proxyConfig.GetConfigStore()
	configs := store.GetAll()

	sendResponse(w, map[string]interface{}{
		"configs": configs,
		"count":   len(configs),
	})
}

// RegisterConfigRoutes 注册Config相关路由
func RegisterConfigRoutes() {
	http.HandleFunc("/config/get", handleConfigGet)
	http.HandleFunc("/config/list", handleConfigList)
}
