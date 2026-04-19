package handlers

import (
	"net/http"

	"aliang.one/nursorgate/app/http/common"
	"aliang.one/nursorgate/processor/statistic"
)

type StatusHandler struct {
	trafficCollector *statistic.StatsCollector
	httpCollector    *statistic.HTTPStatsCollector
	aiTracker        *statistic.AIActivityTracker
}

func NewStatusHandler(
	trafficCollector *statistic.StatsCollector,
	httpCollector *statistic.HTTPStatsCollector,
	aiTracker *statistic.AIActivityTracker,
) *StatusHandler {
	return &StatusHandler{
		trafficCollector: trafficCollector,
		httpCollector:    httpCollector,
		aiTracker:        aiTracker,
	}
}

func (h *StatusHandler) HandleGetSummary(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		common.Error(w, http.StatusMethodNotAllowed, "Method not allowed", nil)
		return
	}

	var traffic *statistic.CurrentStats
	if h.trafficCollector != nil {
		traffic = h.trafficCollector.GetCurrent()
	}
	if traffic == nil {
		traffic = &statistic.CurrentStats{}
	}

	httpStats := map[string]interface{}{}
	if h.httpCollector != nil {
		httpStats = h.httpCollector.GetStats()
	}

	aiSummary := statistic.GetDefaultAIActivityTracker().Summary()
	if h.aiTracker != nil {
		aiSummary = h.aiTracker.Summary()
	}

	common.Success(w, map[string]interface{}{
		"traffic": traffic,
		"http":    httpStats,
		"ai":      aiSummary,
	})
}
