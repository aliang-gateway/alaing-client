package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"nursor.org/nursorgate/app/http/common"
)

// LogHandler handles HTTP requests for log-related operations
type LogHandler struct {
	logService       *LogService
	logConfigService *LogConfigService
}

// NewLogHandler creates a new log handler instance
func NewLogHandler() *LogHandler {
	return &LogHandler{
		logService:       NewLogService(),
		logConfigService: NewLogConfigService(),
	}
}

// HandleGetLogs handles GET /api/logs
// Query parameters: limit, level, source
func (lh *LogHandler) HandleGetLogs(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	levelStr := r.URL.Query().Get("level")
	source := r.URL.Query().Get("source")

	params := LogsQueryParams{
		Limit:  limit,
		Level:  levelStr,
		Source: source,
	}

	// Get logs from service
	responses := lh.logService.GetLogs(params)

	resp := BuildSuccessResponse(map[string]interface{}{
		"entries": responses,
		"count":   len(responses),
	})

	common.SendResponse(w, resp)
}

// HandleClearLogs handles POST /api/logs/clear
func (lh *LogHandler) HandleClearLogs(w http.ResponseWriter, r *http.Request) {
	if err := lh.logService.ClearLogs(); err != nil {
		common.SendError(w, "Failed to clear logs", http.StatusInternalServerError, nil)
		return
	}

	resp := BuildSuccessResponse(map[string]string{"status": "cleared"})
	common.SendResponse(w, resp)
}

// HandleSetLogLevel handles POST /api/logs/level
func (lh *LogHandler) HandleSetLogLevel(w http.ResponseWriter, r *http.Request) {
	var req LogLevelRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		common.SendError(w, ErrInvalidRequestBody.Error(), http.StatusBadRequest, nil)
		return
	}

	level, err := lh.logService.UpdateLogLevel(req.Level)
	if err != nil {
		common.SendError(w, err.Error(), http.StatusBadRequest, nil)
		return
	}

	resp := BuildSuccessResponse(map[string]string{"level": LogLevelTypeToString(level)})
	common.SendResponse(w, resp)
}

// HandleGetLogConfig handles GET /api/logs/config
func (lh *LogHandler) HandleGetLogConfig(w http.ResponseWriter, r *http.Request) {
	config := lh.logConfigService.GetConfig()
	resp := BuildSuccessResponse(config)
	common.SendResponse(w, resp)
}

// HandleSetLogConfig handles POST /api/logs/config
func (lh *LogHandler) HandleSetLogConfig(w http.ResponseWriter, r *http.Request) {
	var req LogConfigRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		common.SendError(w, ErrInvalidRequestBody.Error(), http.StatusBadRequest, nil)
		return
	}

	if err := lh.logConfigService.UpdateConfig(req); err != nil {
		statusCode := http.StatusBadRequest
		common.SendError(w, err.Error(), statusCode, nil)
		return
	}

	resp := BuildSuccessResponse(map[string]string{"status": "config updated"})
	common.SendResponse(w, resp)
}

// HandleLogConfig handles both GET and POST for /api/logs/config
func (lh *LogHandler) HandleLogConfig(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		lh.HandleGetLogConfig(w, r)
	case http.MethodPost:
		lh.HandleSetLogConfig(w, r)
	default:
		common.SendError(w, "Method not allowed", http.StatusMethodNotAllowed, nil)
	}
}
