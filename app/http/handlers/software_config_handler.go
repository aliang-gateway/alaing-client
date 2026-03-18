package handlers

import (
	"errors"
	"net/http"
	"strings"

	"nursor.org/nursorgate/app/http/common"
	"nursor.org/nursorgate/app/http/models"
	"nursor.org/nursorgate/app/http/services"
)

type SoftwareConfigHandler struct {
	service *services.SoftwareConfigService
}

func NewSoftwareConfigHandler(service *services.SoftwareConfigService) *SoftwareConfigHandler {
	return &SoftwareConfigHandler{service: service}
}

func (h *SoftwareConfigHandler) HandleSave(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		common.Error(w, http.StatusMethodNotAllowed, "Method not allowed", nil)
		return
	}

	var req models.SaveSoftwareConfigRequest
	if err := common.DecodeRequest(r, &req); err != nil {
		common.ErrorBadRequest(w, "Invalid request body", map[string]interface{}{"error": err.Error()})
		return
	}

	cfg, err := h.service.Save(req)
	if err != nil {
		common.ErrorBadRequest(w, err.Error(), nil)
		return
	}

	common.Success(w, cfg)
}

func (h *SoftwareConfigHandler) HandleActivate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		common.Error(w, http.StatusMethodNotAllowed, "Method not allowed", nil)
		return
	}

	var req models.ActivateSoftwareConfigRequest
	if err := common.DecodeRequest(r, &req); err != nil {
		common.ErrorBadRequest(w, "Invalid request body", map[string]interface{}{"error": err.Error()})
		return
	}

	cfg, err := h.service.Activate(req)
	if err != nil {
		common.ErrorBadRequest(w, err.Error(), nil)
		return
	}

	common.Success(w, cfg)
}

func (h *SoftwareConfigHandler) HandlePushToCloud(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		common.Error(w, http.StatusMethodNotAllowed, "Method not allowed", nil)
		return
	}

	var req models.CloudPushRequest
	if err := common.DecodeRequest(r, &req); err != nil {
		common.ErrorBadRequest(w, "Invalid request body", map[string]interface{}{"error": err.Error()})
		return
	}

	resp, err := h.service.PushToCloud(req)
	if err != nil {
		if isBadRequestError(err) {
			common.ErrorBadRequest(w, err.Error(), nil)
			return
		}
		common.ErrorInternalServer(w, "Cloud push failed", map[string]interface{}{"error": err.Error()})
		return
	}

	common.Success(w, resp)
}

func (h *SoftwareConfigHandler) HandlePullFromCloud(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		common.Error(w, http.StatusMethodNotAllowed, "Method not allowed", nil)
		return
	}

	var req models.CloudPullRequest
	if err := common.DecodeRequest(r, &req); err != nil {
		common.ErrorBadRequest(w, "Invalid request body", map[string]interface{}{"error": err.Error()})
		return
	}

	resp, err := h.service.PullFromCloud(req)
	if err != nil {
		if isBadRequestError(err) {
			common.ErrorBadRequest(w, err.Error(), nil)
			return
		}
		common.ErrorInternalServer(w, "Cloud pull failed", map[string]interface{}{"error": err.Error()})
		return
	}

	common.Success(w, resp)
}

func isBadRequestError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, http.ErrMissingFile) {
		return true
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "is required") || strings.Contains(msg, "must be rfc3339") || strings.Contains(msg, "not valid")
}
