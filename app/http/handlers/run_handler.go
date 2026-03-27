package handlers

import (
	"net/http"

	"nursor.org/nursorgate/app/http/common"
	"nursor.org/nursorgate/app/http/services"
)

// RunHandler handles HTTP requests for run mode operations
type RunHandler struct {
	runService *services.RunService
}

// NewRunHandler creates a new run handler instance with dependency injection
func NewRunHandler(runService *services.RunService) *RunHandler {
	return &RunHandler{
		runService: runService,
	}
}

// HandleRunStart handles POST /api/run/start
// No authentication required - starts the service for the current mode
func (rh *RunHandler) HandleRunStart(w http.ResponseWriter, r *http.Request) {
	result := rh.runService.StartService()
	common.Success(w, result)
}

// HandleRunStop handles POST /api/run/stop
func (rh *RunHandler) HandleRunStop(w http.ResponseWriter, r *http.Request) {
	result := rh.runService.StopService()
	common.Success(w, result)
}

// HandleRunStatus handles GET /api/run/status
func (rh *RunHandler) HandleRunStatus(w http.ResponseWriter, r *http.Request) {
	result := rh.runService.GetStatus()
	common.Success(w, result)
}

// HandleRunWintunInstall handles POST /api/run/wintun/install
func (rh *RunHandler) HandleRunWintunInstall(w http.ResponseWriter, r *http.Request) {
	result := services.StartWintunDependencyInstall()
	common.Success(w, result)
}

// HandleRunSwift handles POST /api/run/swift
func (rh *RunHandler) HandleRunSwift(w http.ResponseWriter, r *http.Request) {
	var req struct {
		TargetMode string `json:"mode"`
	}

	if err := common.DecodeRequest(r, &req); err != nil {
		common.ErrorBadRequest(w, "Invalid request body", nil)
		return
	}

	result := rh.runService.SwitchMode(req.TargetMode)
	common.Success(w, result)
}
