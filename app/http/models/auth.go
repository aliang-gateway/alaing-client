package models

type LoginRequest struct {
	Email          string `json:"email"`
	Password       string `json:"password"`
	TurnstileToken string `json:"turnstile_token,omitempty"`
}

type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token,omitempty"`
}

type LogoutRequest struct {
	RefreshToken string `json:"refresh_token,omitempty"`
}

// UserInfoResponse 用户信息响应
type UserInfoResponse struct {
	Username     string `json:"username"`
	Email        string `json:"email,omitempty"`
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
