package handlers

import (
	"net/http"

	"nursor.org/nursorgate/app/http/common"
	"nursor.org/nursorgate/processor/nacos"
)

// NacosHandler handles Nacos-related API requests
type NacosHandler struct {
	manager *nacos.ConfigManager
}

// NewNacosHandler creates a new Nacos handler instance
func NewNacosHandler(manager *nacos.ConfigManager) *NacosHandler {
	return &NacosHandler{
		manager: manager,
	}
}

// T079: HandleGetNacosHealth handles GET /api/v1/nacos/health
// Returns the health status of Nacos connection
func (nh *NacosHandler) HandleGetNacosHealth(w http.ResponseWriter, r *http.Request) {
	health := nacos.GetNacosHealth(nh.manager)
	common.Success(w, health)
}

// T080: HandleGetNacosConnection handles GET /api/v1/nacos/connection
// Returns Nacos connection configuration and status
func (nh *NacosHandler) HandleGetNacosConnection(w http.ResponseWriter, r *http.Request) {
	conn := nacos.GetNacosConnection(nh.manager)
	common.Success(w, conn)
}

// T081: HandleGetListenerStatus handles GET /api/v1/nacos/listener/status
// Returns the current listener status
func (nh *NacosHandler) HandleGetListenerStatus(w http.ResponseWriter, r *http.Request) {
	if nh.manager == nil {
		common.ErrorInternalServer(w, "Nacos manager not initialized", nil)
		return
	}

	status := nh.manager.GetListenerStatus()
	common.Success(w, status)
}

// T082: HandleStartListener handles POST /api/v1/nacos/listener/start
// Manually starts the Nacos listener
func (nh *NacosHandler) HandleStartListener(w http.ResponseWriter, r *http.Request) {
	if nh.manager == nil {
		common.ErrorInternalServer(w, "Nacos manager not initialized", nil)
		return
	}

	if err := nh.manager.StartListeningManual(); err != nil {
		common.ErrorInternalServer(w, "Failed to start listener: "+err.Error(), nil)
		return
	}

	common.Success(w, map[string]interface{}{
		"status":  "success",
		"message": "Listener started successfully",
	})
}

// T083: HandleStopListener handles POST /api/v1/nacos/listener/stop
// Manually stops the Nacos listener
func (nh *NacosHandler) HandleStopListener(w http.ResponseWriter, r *http.Request) {
	if nh.manager == nil {
		common.ErrorInternalServer(w, "Nacos manager not initialized", nil)
		return
	}

	if err := nh.manager.StopListeningManual(); err != nil {
		common.ErrorInternalServer(w, "Failed to stop listener: "+err.Error(), nil)
		return
	}

	common.Success(w, map[string]interface{}{
		"status":  "success",
		"message": "Listener stopped successfully",
	})
}

// T084: HandleGetSyncStatus handles GET /api/v1/nacos/sync/status
// Returns the synchronization status between local and Nacos config
func (nh *NacosHandler) HandleGetSyncStatus(w http.ResponseWriter, r *http.Request) {
	status := nacos.GetSyncStatus(nh.manager)
	common.Success(w, status)
}

// T085: HandleCompareConfigs handles GET /api/v1/nacos/config/compare
// Returns detailed configuration comparison
func (nh *NacosHandler) HandleCompareConfigs(w http.ResponseWriter, r *http.Request) {
	if nh.manager == nil {
		common.ErrorInternalServer(w, "Nacos manager not initialized", nil)
		return
	}

	status := nacos.GetSyncStatus(nh.manager)
	common.Success(w, map[string]interface{}{
		"is_synced":      status.IsSynced,
		"local_version":  status.LocalVersion,
		"remote_version": status.RemoteVersion,
		"differences":    status.Differences,
		"last_sync_time": status.LastSyncTime,
		"time_diff_ms":   status.TimeDiff,
		"is_auto_update": status.IsAutoUpdate,
	})
}

// T086: HandleManualSync handles POST /api/v1/nacos/sync/manual
// Forcefully syncs local config with Nacos
func (nh *NacosHandler) HandleManualSync(w http.ResponseWriter, r *http.Request) {
	if nh.manager == nil {
		common.ErrorInternalServer(w, "Nacos manager not initialized", nil)
		return
	}

	if err := nacos.ManualSync(nh.manager); err != nil {
		common.ErrorInternalServer(w, "Manual sync failed: "+err.Error(), nil)
		return
	}

	common.Success(w, map[string]interface{}{
		"status":  "success",
		"message": "Manual sync completed successfully",
	})
}
