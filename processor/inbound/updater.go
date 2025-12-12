package inbound

import (
	"fmt"

	"nursor.org/nursorgate/common/logger"
	"nursor.org/nursorgate/outbound"
	"nursor.org/nursorgate/processor/config"
)

// UpdateDoorProxies fetches and updates Door proxy members (network-first strategy)
// Strategy: Try network -> on failure use memory cache -> on cache miss load disk cache -> error
func UpdateDoorProxies(accessToken string) error {
	var inbounds []InboundInfo
	var err error

	// Step 1: Try to fetch from network
	inbounds, err = FetchInbounds(accessToken)

	if err == nil {
		// Network success: use network data, save to local
		logger.Info("Successfully fetched inbounds from network")

		// Save to local encrypted cache
		if saveErr := SaveInboundsCache(inbounds); err != nil {
			logger.Warn(fmt.Sprintf("Failed to save inbounds cache: %v", saveErr))
			// Save error doesn't affect usage, continue
		}

		// Update in-memory cache
		SetCachedInbounds(inbounds)
	} else {
		// Network failed: try local cache
		logger.Warn(fmt.Sprintf("Failed to fetch inbounds from network: %v, falling back to cache", err))

		// Step 2: Try memory cache first
		cachedInbounds, timestamp := GetCachedInbounds()
		if len(cachedInbounds) > 0 {
			logger.Info(fmt.Sprintf("Using cached inbounds (last updated: %d)", timestamp))
			inbounds = cachedInbounds
		} else {
			// Step 3: Try to load from disk cache file
			loaded, loadErr := LoadInboundsCache()
			if loadErr == nil && len(loaded) > 0 {
				logger.Info("Loaded inbounds from local cache file")
				inbounds = loaded
				SetCachedInbounds(loaded) // Also update memory cache
			} else {
				// Step 4: No inbounds available anywhere
				logger.Warn("No inbounds available (network error and no cache)")
				return fmt.Errorf("failed to fetch inbounds and no cache available")
			}
		}
	}

	// Convert and register to Door proxy group
	return registerInboundsToDoor(inbounds)
}

// registerInboundsToDoor converts InboundInfo to proxy format and registers with Door proxy group
func registerInboundsToDoor(inbounds []InboundInfo) error {
	if len(inbounds) == 0 {
		return fmt.Errorf("no inbounds to register")
	}

	var members []config.DoorProxyMember

	// Convert each inbound to proxy configuration
	for _, info := range inbounds {
		member, err := ConvertToProxyConfig(info)
		if err != nil {
			logger.Warn(fmt.Sprintf("Failed to convert inbound %s: %v, skipping", info.Tag, err))
			continue // Skip failed inbound, continue with others
		}
		members = append(members, *member)
	}

	if len(members) == 0 {
		return fmt.Errorf("no valid inbounds to register")
	}

	// Register Door proxy configuration
	registry := outbound.GetRegistry()
	if registry == nil {
		return fmt.Errorf("proxy registry is not available")
	}

	doorConfig := &config.DoorProxyConfig{
		Type:    "door",
		Members: members,
	}

	logger.Info(fmt.Sprintf("Registering %d inbound members to Door proxy", len(members)))
	return registry.RegisterDoorFromConfig(doorConfig)
}
