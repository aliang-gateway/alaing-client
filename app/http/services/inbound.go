package services

import (
	"fmt"

	"nursor.org/nursorgate/processor/inbound"
)

// InboundService provides inbound proxy management services
type InboundService struct{}

// NewInboundService creates a new InboundService instance
func NewInboundService() *InboundService {
	return &InboundService{}
}

// UpdateProxiesFromInbounds updates Door proxy members from inbound configurations
func (s *InboundService) UpdateProxiesFromInbounds(accessToken string) map[string]interface{} {
	// Update Door proxies with network-first strategy
	err := inbound.UpdateDoorProxies(accessToken)

	if err != nil {
		return map[string]interface{}{
			"status": "failed",
			"error":  "update_failed",
			"msg":    fmt.Sprintf("Failed to update inbounds: %v", err),
		}
	}

	// Get updated cache information
	cachedInbounds, timestamp := inbound.GetCachedInbounds()

	return map[string]interface{}{
		"status":          "success",
		"msg":             "Inbounds updated successfully",
		"count":           len(cachedInbounds),
		"last_update":     timestamp,
		"inbounds_count":  len(cachedInbounds),
	}
}

// GetInboundStatus returns current inbound cache status
func (s *InboundService) GetInboundStatus() map[string]interface{} {
	cachedInbounds, timestamp := inbound.GetCachedInbounds()

	return map[string]interface{}{
		"status":       "success",
		"count":        len(cachedInbounds),
		"last_update":  timestamp,
		"has_cache":    inbound.HasCachedInbounds(),
	}
}
