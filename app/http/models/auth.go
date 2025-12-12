package models

// ActivateTokenRequest Token激活请求
type ActivateTokenRequest struct {
	Token string `json:"token"`
}

// UserInfoResponse 用户信息响应
type UserInfoResponse struct {
	Username     string `json:"username"`
	PlanName     string `json:"plan_name"`
	PlanType     string `json:"plan_type"`
	TrafficUsed  int64  `json:"traffic_used"`
	TrafficTotal int64  `json:"traffic_total"`
	AIAskUsed    int    `json:"ai_ask_used"`
	AIAskTotal   int    `json:"ai_ask_total"`
	StartTime    string `json:"start_time"`
	EndTime      string `json:"end_time"`
	UpdatedAt    string `json:"updated_at"`
}

// RefreshStatusResponse 刷新状态响应
type RefreshStatusResponse struct {
	IsRunning      bool   `json:"is_running"`
	LastUpdateTime string `json:"last_update_time,omitempty"`
	LastError      string `json:"last_error,omitempty"`
	RefreshInterval string `json:"refresh_interval"`
}
