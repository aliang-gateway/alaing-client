package proxyserver

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"nursor.org/nursorgate/common/logger"
	"nursor.org/nursorgate/processor/config"
)

const (
	// API call timeout
	apiTimeout = 10 * time.Second
)

// apiInboundResponse 内部API响应结构（保持与原始API兼容）
type apiInboundResponse struct {
	Code int            `json:"code"`
	Msg  string         `json:"msg"`
	Data []apiProxyInfo `json:"data"`
}

// apiProxyInfo 内部API代理信息结构
type apiProxyInfo struct {
	InboundType string      `json:"type"`
	Tag         string      `json:"show_name"` // API字段名
	Config      interface{} `json:"config"`
}

// FetchInbounds 从API获取用户代理配置，直接返回DoorProxyMember
func FetchInbounds(accessToken string) ([]config.DoorProxyMember, error) {
	if accessToken == "" {
		return nil, fmt.Errorf("access token cannot be empty")
	}

	// Get URL builder
	urlBuilder, err := config.NewURLBuilder()
	if err != nil {
		return nil, err
	}

	// Get and validate inbounds URL
	inboundsURL, err := urlBuilder.GetInboundsURL()
	if err != nil {
		return nil, err
	}

	// Create GET request
	req, err := http.NewRequest("GET", inboundsURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set Authorization header with Bearer token
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", accessToken))
	req.Header.Set("Content-Type", "application/json")

	// Send request
	client := &http.Client{
		Timeout: apiTimeout,
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Check HTTP status code
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("api returned status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response using internal structure
	var response apiInboundResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Check API response code
	if response.Code != 0 {
		return nil, fmt.Errorf("api error: %s (code: %d)", response.Msg, response.Code)
	}

	if len(response.Data) == 0 {
		logger.Warn("API returned empty proxyserver list")
		return []config.DoorProxyMember{}, nil
	}

	// 直接转换为DoorProxyMember，无中间层
	var members []config.DoorProxyMember
	for _, info := range response.Data {
		member := config.DoorProxyMember{
			ShowName: info.Tag,                          // 统一使用ShowName字段
			Type:     strings.ToLower(info.InboundType), // 统一为小写
			Latency:  0,                                 // 初始值，后续会更新
			Config:   info.Config,                       // 直接使用，无需转换
		}
		members = append(members, member)
	}

	logger.Info(fmt.Sprintf("Successfully fetched %d door proxy members from API", len(members)))
	return members, nil
}
