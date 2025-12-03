package common

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
)

// DecodeRequest 解析HTTP请求体
func DecodeRequest(r *http.Request, v interface{}) error {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return err
	}
	return json.Unmarshal(body, v)
}
