package common

import (
	"encoding/json"
	"net/http"
)

// Response 通用的HTTP响应结构体
type Response struct {
	Code int         `json:"code"`
	Msg  string      `json:"msg"`
	Data interface{} `json:"data"`
}

// SendResponse 发送成功响应
func SendResponse(w http.ResponseWriter, data interface{}) {
	resp := Response{
		Code: 0,
		Msg:  "success",
		Data: data,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// SendError 发送错误响应
func SendError(w http.ResponseWriter, msg string, statusCode int, data interface{}) {
	resp := Response{
		Code: statusCode,
		Msg:  msg,
		Data: data,
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(resp)
}
