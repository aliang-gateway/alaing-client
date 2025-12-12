package inbound

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"nursor.org/nursorgate/common/logger"
)

const (
	// API endpoint for fetching inbounds
	inboundsAPIURL = "http://127.0.0.1:8000/api/production/prod/sui/user/sui/inbounds"
	// API call timeout
	apiTimeout = 10 * time.Second
)

// FetchInbounds fetches user's inbound configurations from API
func FetchInbounds(accessToken string) ([]InboundInfo, error) {
	if accessToken == "" {
		return nil, fmt.Errorf("access token cannot be empty")
	}

	// Create GET request
	req, err := http.NewRequest("GET", inboundsAPIURL, nil)
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

	// Parse response
	var response InboundResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Check API response code
	if response.Code != 0 {
		return nil, fmt.Errorf("api error: %s (code: %d)", response.Msg, response.Code)
	}

	if len(response.Data) == 0 {
		logger.Warn("API returned empty inbound list")
		return []InboundInfo{}, nil
	}

	logger.Info(fmt.Sprintf("Successfully fetched %d inbounds from API", len(response.Data)))
	return response.Data, nil
}
