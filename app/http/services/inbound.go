package services

import (
	"nursor.org/nursorgate/processor/inbound"
)

// InboundService provides inbound proxy management services
type InboundService struct{}

// NewInboundService creates a new InboundService instance
func NewInboundService() *InboundService {
	return &InboundService{}
}

// GetInboundStatus returns current inbound cache status
func (s *InboundService) GetInboundStatus() map[string]interface{} {
	cachedInbounds, timestamp := inbound.GetCachedInbounds()

	return map[string]interface{}{
		"status":      "success",
		"count":       len(cachedInbounds),
		"last_update": timestamp,
		"has_cache":   inbound.HasCachedInbounds(),
	}
}
