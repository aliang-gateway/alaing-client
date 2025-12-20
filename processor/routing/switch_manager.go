package routing

import (
	"sync"

	"nursor.org/nursorgate/common/logger"
	"nursor.org/nursorgate/common/model"
)

// SwitchManager manages the global routing switches
type SwitchManager struct {
	mu              sync.RWMutex
	noneLaneEnabled bool
	doorEnabled     bool
	geoIPEnabled    bool
	lastUpdatedAt   int64
	enabledAt       map[string]int64 // Track when each switch was last changed
}

var (
	defaultSwitchManager *SwitchManager
	switchOnce           sync.Once
)

// GetSwitchManager returns the singleton SwitchManager instance
func GetSwitchManager() *SwitchManager {
	switchOnce.Do(func() {
		defaultSwitchManager = &SwitchManager{
			noneLaneEnabled: true,
			doorEnabled:     true,
			geoIPEnabled:    false,
			enabledAt:       make(map[string]int64),
		}
	})
	return defaultSwitchManager
}

// UpdateSwitches updates the global switches based on configuration
func (sm *SwitchManager) UpdateSwitches(config *model.RoutingRulesConfig) {
	if config == nil {
		logger.Warn("Cannot update switches: config is nil")
		return
	}

	sm.mu.Lock()
	defer sm.mu.Unlock()

	settings := config.Settings

	if sm.noneLaneEnabled != settings.NoneLaneEnabled {
		if settings.NoneLaneEnabled {
			logger.Info("Global switch: NoneLane ENABLED")
		} else {
			logger.Info("Global switch: NoneLane DISABLED")
		}
		sm.noneLaneEnabled = settings.NoneLaneEnabled
	}

	if sm.doorEnabled != settings.DoorEnabled {
		if settings.DoorEnabled {
			logger.Info("Global switch: Door ENABLED")
		} else {
			logger.Info("Global switch: Door DISABLED")
		}
		sm.doorEnabled = settings.DoorEnabled
	}

	if sm.geoIPEnabled != settings.GeoIPEnabled {
		if settings.GeoIPEnabled {
			logger.Info("Global switch: GeoIP ENABLED")
		} else {
			logger.Info("Global switch: GeoIP DISABLED")
		}
		sm.geoIPEnabled = settings.GeoIPEnabled
	}
}

// IsNoneLaneEnabled returns the NoneLane switch status
func (sm *SwitchManager) IsNoneLaneEnabled() bool {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.noneLaneEnabled
}

// IsDoorEnabled returns the Door switch status
func (sm *SwitchManager) IsDoorEnabled() bool {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.doorEnabled
}

// IsGeoIPEnabled returns the GeoIP switch status
func (sm *SwitchManager) IsGeoIPEnabled() bool {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.geoIPEnabled
}

// SetNoneLaneEnabled sets the NoneLane switch
func (sm *SwitchManager) SetNoneLaneEnabled(enabled bool) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if sm.noneLaneEnabled != enabled {
		sm.noneLaneEnabled = enabled
		if enabled {
			logger.Info("Global switch: NoneLane ENABLED (manual)")
		} else {
			logger.Info("Global switch: NoneLane DISABLED (manual)")
		}
	}
}

// SetDoorEnabled sets the Door switch
func (sm *SwitchManager) SetDoorEnabled(enabled bool) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if sm.doorEnabled != enabled {
		sm.doorEnabled = enabled
		if enabled {
			logger.Info("Global switch: Door ENABLED (manual)")
		} else {
			logger.Info("Global switch: Door DISABLED (manual)")
		}
	}
}

// SetGeoIPEnabled sets the GeoIP switch
func (sm *SwitchManager) SetGeoIPEnabled(enabled bool) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if sm.geoIPEnabled != enabled {
		sm.geoIPEnabled = enabled
		if enabled {
			logger.Info("Global switch: GeoIP ENABLED (manual)")
		} else {
			logger.Info("Global switch: GeoIP DISABLED (manual)")
		}
	}
}

// GetStatus returns current switch status as a model struct
func (sm *SwitchManager) GetStatus() model.RulesSettings {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	return model.RulesSettings{
		NoneLaneEnabled: sm.noneLaneEnabled,
		DoorEnabled:     sm.doorEnabled,
		GeoIPEnabled:    sm.geoIPEnabled,
	}
}

// DisableAll disables all switches
func (sm *SwitchManager) DisableAll() {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	sm.noneLaneEnabled = false
	sm.doorEnabled = false
	sm.geoIPEnabled = false
	logger.Info("Global switches: ALL DISABLED")
}

// EnableAll enables all switches
func (sm *SwitchManager) EnableAll() {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	sm.noneLaneEnabled = true
	sm.doorEnabled = true
	sm.geoIPEnabled = true
	logger.Info("Global switches: ALL ENABLED")
}

// ResetToDefaults resets switches to default state
func (sm *SwitchManager) ResetToDefaults() {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	sm.noneLaneEnabled = true
	sm.doorEnabled = true
	sm.geoIPEnabled = false
	logger.Info("Global switches: RESET to defaults (NoneLane=ON, Door=ON, GeoIP=OFF)")
}
