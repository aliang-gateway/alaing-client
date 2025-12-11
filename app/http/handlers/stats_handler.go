package handlers

import (
	"net/http"

	"nursor.org/nursorgate/app/http/common"
	"nursor.org/nursorgate/processor/statistic"
)

// StatsHandler handles HTTP requests for statistics operations
type StatsHandler struct {
	statsManager *statistic.Manager
}

// NewStatsHandler creates a new stats handler instance with dependency injection
func NewStatsHandler(statsManager *statistic.Manager) *StatsHandler {
	return &StatsHandler{
		statsManager: statsManager,
	}
}

// HandleGetStats handles GET /api/stats - returns snapshot with route-grouped statistics
func (sh *StatsHandler) HandleGetStats(w http.ResponseWriter, r *http.Request) {
	snapshot := sh.statsManager.Snapshot()
	common.Success(w, snapshot)
}
