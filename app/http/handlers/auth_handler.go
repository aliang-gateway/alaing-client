package handlers

import (
	"fmt"
	"net/http"

	"nursor.org/nursorgate/app/http/common"
	"nursor.org/nursorgate/app/http/models"
	"nursor.org/nursorgate/app/http/services"
	"nursor.org/nursorgate/common/logger"
	userAuth "nursor.org/nursorgate/processor/auth"
	"nursor.org/nursorgate/processor/proxyserver"
)

// AuthHandler Token和用户认证处理器
type AuthHandler struct {
	authService *services.AuthService
}

// NewAuthHandler 创建新的认证处理器实例
func NewAuthHandler() *AuthHandler {
	return &AuthHandler{
		authService: services.NewAuthService(),
	}
}

// HandleActivateToken 处理Token激活请求
// POST /api/auth/activate
func (h *AuthHandler) HandleActivateToken(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		common.Error(w, http.StatusMethodNotAllowed, "Method not allowed", nil)
		return
	}

	var req models.ActivateTokenRequest
	if err := common.DecodeRequest(r, &req); err != nil {
		common.ErrorBadRequest(w, "Invalid request format", nil)
		return
	}

	result := h.authService.ActivateToken(req.Token)
	go func() {
		// 异步更新代理配置（不阻塞响应）
		userInfo := userAuth.GetCurrentUserInfo()
		if userInfo != nil && userInfo.AccessToken != "" {
			if err := proxyserver.UpdateDoorProxies(userInfo.AccessToken); err != nil {
				logger.Warn(fmt.Sprintf("Failed to fetch proxyserver config after token activation: %v", err))
			} else {
				logger.Info("Proxyserver config updated successfully after token activation")
			}
		}
	}()
	common.Success(w, result)
}

// HandleGetUserInfo 处理获取用户信息请求
// GET /api/auth/userinfo
func (h *AuthHandler) HandleGetUserInfo(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		common.Error(w, http.StatusMethodNotAllowed, "Method not allowed", nil)
		return
	}

	result := h.authService.GetUserInfo()
	common.Success(w, result)
}

// HandleGetRefreshStatus 处理获取刷新状态请求
// GET /api/auth/refresh-status
func (h *AuthHandler) HandleGetRefreshStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		common.Error(w, http.StatusMethodNotAllowed, "Method not allowed", nil)
		return
	}

	result := h.authService.GetRefreshStatus()
	common.Success(w, result)
}

// HandleLogout 处理登出请求
// POST /api/auth/logout
func (h *AuthHandler) HandleLogout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		common.Error(w, http.StatusMethodNotAllowed, "Method not allowed", nil)
		return
	}

	result := h.authService.LogoutUser()
	common.Success(w, result)
}
