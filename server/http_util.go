package server

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
)

// decodeRequest 解析HTTP请求体
func decodeRequest(r *http.Request, v interface{}) error {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return err
	}
	return json.Unmarshal(body, v)
}

// sendResponse 发送成功响应
func sendResponse(w http.ResponseWriter, data interface{}) {
	resp := Response{
		Code: 0,
		Msg:  "success",
		Data: data,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// sendError 发送错误响应
func sendError(w http.ResponseWriter, msg string, statusCode int, data interface{}) {
	resp := Response{
		Code: statusCode,
		Msg:  msg,
		Data: data,
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(resp)
}

// writePortToFile 写入端口到文件
func writePortToFile(port string) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	filePath := filepath.Join(homeDir, ".cursor", "nursor")
	err = os.MkdirAll(filepath.Dir(filePath), 0755)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(filePath, []byte(port[1:]), 0644) // 去掉冒号，只写 56431
}
