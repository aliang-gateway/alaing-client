package models

// ConfigGetRequest is the request to get specific config
type ConfigGetRequest struct {
	Name string `json:"name"`
}

// ConfigInfo represents configuration information
type ConfigInfo struct {
	Name string      `json:"name"`
	Data interface{} `json:"data"`
}
