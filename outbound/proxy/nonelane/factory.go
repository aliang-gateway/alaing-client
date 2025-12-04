package nonelane

// NewNonelane creates a new nonelane proxy instance for mTLS connections
func NewNonelane(config *NoneLaneConfig) (*NoneLane, error) {
	return New(config)
}
