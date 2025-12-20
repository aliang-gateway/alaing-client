package nacos

import (
	"fmt"
	"time"

	"github.com/nacos-group/nacos-sdk-go/vo"
	"nursor.org/nursorgate/common/logger"
)

// T071: NacosHealth contains health check information
type NacosHealth struct {
	IsHealthy  bool        `json:"is_healthy"`
	Reachable  bool        `json:"reachable"`
	Connected  bool        `json:"connected"`
	ResponseMS int64       `json:"response_ms"`
	LastCheck  time.Time   `json:"last_check"`
	Error      string      `json:"error,omitempty"`
	Details    interface{} `json:"details,omitempty"`
}

// T071: GetNacosHealth checks the health status of Nacos connection
func GetNacosHealth(manager *ConfigManager) *NacosHealth {
	if manager == nil {
		return &NacosHealth{
			IsHealthy: false,
			Error:     "ConfigManager not initialized",
			LastCheck: time.Now(),
		}
	}

	health := &NacosHealth{
		LastCheck: time.Now(),
	}

	// Check if client is initialized
	if manager.client == nil {
		health.IsHealthy = false
		health.Error = "Nacos client not initialized"
		return health
	}

	// Try to get current config as health check
	startTime := time.Now()
	_, err := manager.client.GetConfig(vo.ConfigParam{
		DataId: RoutingRulesDataID,
		Group:  DefaultGroup,
	})
	health.ResponseMS = time.Since(startTime).Milliseconds()

	if err != nil {
		health.IsHealthy = false
		health.Reachable = false
		health.Error = err.Error()
		return health
	}

	// If we got here, Nacos is reachable and working
	health.IsHealthy = true
	health.Reachable = true
	health.Connected = manager.IsListening()

	// Additional details
	health.Details = map[string]interface{}{
		"listener_active": manager.IsListening(),
		"auto_update":     manager.GetAutoUpdateStatus(),
	}

	return health
}

// T072: NacosConnection contains connection configuration
type NacosConnection struct {
	Server        string                 `json:"server"`
	Connected     bool                   `json:"connected"`
	ListeningAuth bool                   `json:"listening_auth"`
	Namespace     string                 `json:"namespace"`
	Group         string                 `json:"group"`
	LastSync      *time.Time             `json:"last_sync,omitempty"`
	Config        map[string]interface{} `json:"config"`
}

// T072: GetNacosConnection returns Nacos connection information
func GetNacosConnection(manager *ConfigManager) *NacosConnection {
	conn := &NacosConnection{
		Server:    "unknown",
		Connected: false,
		Namespace: "",
		Group:     DefaultGroup,
		Config:    make(map[string]interface{}),
	}

	if manager == nil || manager.client == nil {
		conn.Config["error"] = "ConfigManager or client not initialized"
		return conn
	}

	conn.Connected = manager.IsListening()

	// Get current config information
	currentConfig := manager.GetCurrentConfig()
	if currentConfig != nil {
		conn.LastSync = &currentConfig.UpdatedAt
		conn.Config["version"] = currentConfig.Version
		conn.Config["non_lane_enabled"] = currentConfig.Settings.NoneLaneEnabled
		conn.Config["door_enabled"] = currentConfig.Settings.DoorEnabled
		conn.Config["geoip_enabled"] = currentConfig.Settings.GeoIPEnabled
		conn.Config["auto_update"] = currentConfig.Settings.AutoUpdate
	}

	return conn
}

// T073: GetListenerStatus returns the listener status
func (cm *ConfigManager) GetListenerStatus() map[string]interface{} {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	return map[string]interface{}{
		"is_listening": cm.listening,
		"auto_update":  cm.autoUpdate,
		"client_set":   cm.client != nil,
	}
}

// T074: StartListeningManual manually starts the listener
// This is a separate method from StartListening for API exposure
func (cm *ConfigManager) StartListeningManual() error {
	if err := cm.StartListening(); err != nil {
		logger.Error(fmt.Sprintf("Failed to start listener: %v", err))
		return err
	}
	return nil
}

// T075: StopListeningManual manually stops the listener
// This is a separate method from StopListening for API exposure
func (cm *ConfigManager) StopListeningManual() error {
	if err := cm.StopListening(); err != nil {
		logger.Error(fmt.Sprintf("Failed to stop listener: %v", err))
		return err
	}
	return nil
}
