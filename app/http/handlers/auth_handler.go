package handlers

import (
	"net/http"

	"nursor.org/nursorgate/app/http/common"
	"nursor.org/nursorgate/app/http/models"
	"nursor.org/nursorgate/app/http/services"
	"nursor.org/nursorgate/common/logger"
	userAuth "nursor.org/nursorgate/processor/auth"
	"nursor.org/nursorgate/processor/runtime"
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

	// 激活 token
	result := h.authService.ActivateToken(req.Token)

	// 检查激活结果并更新启动状态
	if result["status"] == "success" {
		// 激活成功，更新启动状态
		userInfo := userAuth.GetCurrentUserInfo()
		if userInfo != nil {
			startupState := runtime.GetStartupState()

			// 立即设置用户信息（同步）
			startupState.SetUserInfo(userInfo)

			// 登录成功即视为可启动
			startupState.SetFetchSuccess(true)
			startupState.SetStatus(runtime.READY)
			logger.Info("User activated successfully, status: READY")
		}
	}

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
