package handlers

import (
	"encoding/json"
	"net/http"

	"nursor.org/nursorgate/app/http/common"
	"nursor.org/nursorgate/app/http/models"
	"nursor.org/nursorgate/app/http/services"
)

// LogHandler handles HTTP requests for log-related operations
type LogHandler struct {
	logService       *services.LogService
	logConfigService *services.LogConfigService
}

// NewLogHandler creates a new log handler instance with dependency injection
func NewLogHandler(
	logService *services.LogService,
	logConfigService *services.LogConfigService,
) *LogHandler {
	return &LogHandler{
		logService:       logService,
		logConfigService: logConfigService,
	}
}

// HandleGetLogs handles GET /api/logs
// Query parameters: limit, level, source
func (lh *LogHandler) HandleGetLogs(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	limit := common.GetQueryParamInt(r, "limit", 100)
	levelStr := common.GetQueryParamString(r, "level", "")
	source := common.GetQueryParamString(r, "source", "")

	params := models.LogsQueryParams{
		Limit:  limit,
		Level:  levelStr,
		Source: source,
	}

	// Get logs from service
	responses := lh.logService.GetLogs(params)

	common.Success(w, map[string]interface{}{
		"entries": responses,
		"count":   len(responses),
	})
}

// HandleClearLogs handles POST /api/logs/clear
func (lh *LogHandler) HandleClearLogs(w http.ResponseWriter, r *http.Request) {
	if err := lh.logService.ClearLogs(); err != nil {
		common.ErrorInternalServer(w, "Failed to clear logs", map[string]string{"error": err.Error()})
		return
	}

	common.Success(w, map[string]string{"status": "cleared"})
}

// HandleSetLogLevel handles POST /api/logs/level
func (lh *LogHandler) HandleSetLogLevel(w http.ResponseWriter, r *http.Request) {
	var req models.LogLevelRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		common.ErrorBadRequest(w, services.ErrInvalidRequestBody.Error(), nil)
		return
	}

	level, err := lh.logService.UpdateLogLevel(req.Level)
	if err != nil {
		common.ErrorBadRequest(w, err.Error(), nil)
		return
	}

	common.Success(w, map[string]string{"level": services.LogLevelTypeToString(level)})
}

// HandleGetLogConfig handles GET /api/logs/config
func (lh *LogHandler) HandleGetLogConfig(w http.ResponseWriter, r *http.Request) {
	config := lh.logConfigService.GetConfig()
	common.Success(w, config)
}

// HandleSetLogConfig handles POST /api/logs/config
func (lh *LogHandler) HandleSetLogConfig(w http.ResponseWriter, r *http.Request) {
	var req models.LogConfigRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		common.ErrorBadRequest(w, services.ErrInvalidRequestBody.Error(), nil)
		return
	}

	if err := lh.logConfigService.UpdateConfig(req); err != nil {
		common.ErrorBadRequest(w, err.Error(), nil)
		return
	}

	common.Success(w, map[string]string{"status": "config updated"})
}

// HandleLogConfig handles both GET and POST for /api/logs/config
func (lh *LogHandler) HandleLogConfig(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		lh.HandleGetLogConfig(w, r)
	case http.MethodPost:
		lh.HandleSetLogConfig(w, r)
	default:
		common.Error(w, http.StatusMethodNotAllowed, "Method not allowed", nil)
	}
}
