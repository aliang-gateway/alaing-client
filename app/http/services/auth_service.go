package services

import (
	"fmt"
	"time"

	"nursor.org/nursorgate/app/http/models"
	"nursor.org/nursorgate/common/logger"
	auth "nursor.org/nursorgate/processor/auth"
)

// AuthService 认证服务
type AuthService struct{}

// NewAuthService 创建新的认证服务实例
func NewAuthService() *AuthService {
	return &AuthService{}
}

// ActivateToken 激活Token
func (s *AuthService) ActivateToken(token string) map[string]interface{} {
	if token == "" {
		return map[string]interface{}{
			"status": "failed",
			"error":  "token_required",
			"msg":    "Token cannot be empty",
		}
	}

	// 调用认证包的激活函数
	userInfo, err := auth.ActivateToken(token)
	if err != nil {
		logger.Error(fmt.Sprintf("Token activation failed: %v", err))
		return map[string]interface{}{
			"status": "failed",
			"error":  "activation_failed",
			"msg":    fmt.Sprintf("Failed to activate token: %v", err),
		}
	}

	// 返回用户信息
	return map[string]interface{}{
		"status": "success",
		"msg":    "Token activated successfully",
		"data": models.UserInfoResponse{
			Username:     userInfo.Username,
			PlanName:     userInfo.PlanName,
			PlanType:     userInfo.PlanType,
			TrafficUsed:  userInfo.TrafficUsed,
			TrafficTotal: userInfo.TrafficTotal,
			AIAskUsed:    userInfo.AIAskUsed,
			AIAskTotal:   userInfo.AIAskTotal,
			StartTime:    userInfo.StartTime,
			EndTime:      userInfo.EndTime,
			UpdatedAt:    userInfo.UpdatedAt.Format(time.RFC3339),
		},
	}
}

// GetUserInfo 获取当前用户信息
func (s *AuthService) GetUserInfo() map[string]interface{} {
	userInfo := auth.GetCurrentUserInfo()
	if userInfo == nil {
		return map[string]interface{}{
			"status": "no_user",
			"msg":    "No user info available",
		}
	}

	return map[string]interface{}{
		"status": "success",
		"data": models.UserInfoResponse{
			Username:     userInfo.Username,
			PlanName:     userInfo.PlanName,
			PlanType:     userInfo.PlanType,
			TrafficUsed:  userInfo.TrafficUsed,
			TrafficTotal: userInfo.TrafficTotal,
			AIAskUsed:    userInfo.AIAskUsed,
			AIAskTotal:   userInfo.AIAskTotal,
			StartTime:    userInfo.StartTime,
			EndTime:      userInfo.EndTime,
			UpdatedAt:    userInfo.UpdatedAt.Format(time.RFC3339),
		},
	}
}

// GetRefreshStatus 获取刷新状态
func (s *AuthService) GetRefreshStatus() map[string]interface{} {
	refresher := auth.GetTokenRefresher()
	if refresher == nil {
		return map[string]interface{}{
			"status":           "success",
			"is_running":       false,
			"refresh_interval": "1 minute",
		}
	}

	resp := models.RefreshStatusResponse{
		IsRunning:       refresher.IsRunning(),
		RefreshInterval: "1 minute",
	}

	if refresher.IsRunning() {
		// 添加最后更新时间
		userInfo := auth.GetCurrentUserInfo()
		if userInfo != nil {
			resp.LastUpdateTime = userInfo.UpdatedAt.Format(time.RFC3339)
		}

		// 添加最后错误信息
		if lastErr := refresher.GetLastError(); lastErr != nil {
			resp.LastError = lastErr.Error()
		}
	}

	return map[string]interface{}{
		"status": "success",
		"data":   resp,
	}
}

// LogoutUser 登出用户
func (s *AuthService) LogoutUser() map[string]interface{} {
	// 停止定时刷新
	auth.StopTokenRefresh()

	// 删除本地用户信息
	if err := auth.DeleteUserInfo(); err != nil {
		logger.Warn(fmt.Sprintf("Failed to delete user info: %v", err))
	}

	// 清空运行时状态
	auth.SetInnerToken("")

	logger.Info("User logged out successfully")

	return map[string]interface{}{
		"status": "success",
		"msg":    "User logged out successfully",
	}
}
