package http

// Response 通用的HTTP响应结构体
type Response struct {
	Code int         `json:"code"`
	Msg  string      `json:"msg"`
	Data interface{} `json:"data"`
}

// LoginRequest 登录请求
type LoginRequest struct {
	RefreshToken string `json:"refreshToken"`
	AccessToken  string `json:"accessToken"`
	SubId        string `json:"subId"`
}
