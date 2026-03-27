package user

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"nursor.org/nursorgate/processor/config"
)

type UserProfile struct {
	ID            int64   `json:"id"`
	Email         string  `json:"email"`
	Username      string  `json:"username"`
	Role          string  `json:"role"`
	Balance       float64 `json:"balance"`
	Concurrency   int     `json:"concurrency"`
	Status        string  `json:"status"`
	AllowedGroups []int64 `json:"allowed_groups"`
	CreatedAt     string  `json:"created_at"`
	UpdatedAt     string  `json:"updated_at"`
}

type userProfileEnvelope struct {
	Data    UserProfile `json:"data"`
	Message string      `json:"message"`
}

type userCenterUsageSummary struct {
	ActiveCount   int               `json:"active_count"`
	TotalUsedUSD  float64           `json:"total_used_usd"`
	Subscriptions []json.RawMessage `json:"subscriptions"`
}

type userUsageSummaryEnvelope struct {
	Data    userCenterUsageSummary `json:"data"`
	Message string                 `json:"message"`
}

type UserUsageSummary struct {
	ActiveCount   int               `json:"active_count"`
	TotalUsedUSD  float64           `json:"total_used_usd"`
	Subscriptions []json.RawMessage `json:"subscriptions"`
}

type userUsageProgressEnvelope struct {
	Data    []json.RawMessage `json:"data"`
	Message string            `json:"message"`
}

type UserUsageProgress struct {
	Items []json.RawMessage `json:"items"`
}

type RedeemResult struct {
	Data json.RawMessage `json:"data"`
}

func resolveAccessToken() (string, error) {
	current := GetCurrentUserInfo()
	if current == nil {
		return "", fmt.Errorf("no user session")
	}
	token := strings.TrimSpace(current.AccessToken)
	if token == "" {
		return "", fmt.Errorf("missing access token")
	}
	return token, nil
}

func callAuthenticatedAPI(method, endpoint, accessToken string, body any) ([]byte, error) {
	var reqBody io.Reader
	if body != nil {
		jsonData, marshalErr := json.Marshal(body)
		if marshalErr != nil {
			return nil, fmt.Errorf("failed to marshal request: %w", marshalErr)
		}
		reqBody = bytes.NewBuffer(jsonData)
	}

	req, err := http.NewRequest(method, endpoint, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", accessToken))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: apiTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("api returned status %d: %s", resp.StatusCode, string(responseBody))
	}

	return responseBody, nil
}

func callUserCenterAPI(method, endpoint string, body any) ([]byte, error) {
	accessToken, err := resolveAccessToken()
	if err != nil {
		return nil, err
	}
	return callAuthenticatedAPI(method, endpoint, accessToken, body)
}

func GetUserProfileWithToken(accessToken string) (*UserProfile, error) {
	if strings.TrimSpace(accessToken) == "" {
		return nil, fmt.Errorf("missing access token")
	}

	urlBuilder, err := config.NewURLBuilder()
	if err != nil {
		return nil, err
	}
	profileURL, err := urlBuilder.GetUserProfileURL()
	if err != nil {
		return nil, err
	}

	body, err := callAuthenticatedAPI(http.MethodGet, profileURL, accessToken, nil)
	if err != nil {
		return nil, err
	}

	var envelope userProfileEnvelope
	if err := json.Unmarshal(body, &envelope); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &envelope.Data, nil
}

func GetUserProfile() (*UserProfile, error) {
	accessToken, err := resolveAccessToken()
	if err != nil {
		return nil, err
	}
	return GetUserProfileWithToken(accessToken)
}

func UpdateUserProfile(username string) (*UserProfile, error) {
	trimmed := strings.TrimSpace(username)
	if trimmed == "" {
		return nil, fmt.Errorf("username cannot be empty")
	}

	if _, err := resolveAccessToken(); err != nil {
		return nil, err
	}

	urlBuilder, err := config.NewURLBuilder()
	if err != nil {
		return nil, err
	}
	updateURL, err := urlBuilder.GetUserUpdateURL()
	if err != nil {
		return nil, err
	}

	body, err := callUserCenterAPI(http.MethodPut, updateURL, map[string]string{"username": trimmed})
	if err != nil {
		return nil, err
	}

	var envelope userProfileEnvelope
	if err := json.Unmarshal(body, &envelope); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	current := GetCurrentUserInfo()
	if current != nil {
		applyUserProfileToUserInfo(current, &envelope.Data)
		if strings.TrimSpace(current.Username) == "" {
			current.Username = trimmed
		}
		current.UpdatedAt = time.Now()
		if err := SaveUserInfo(current); err != nil {
			return &envelope.Data, nil
		}
	}

	return &envelope.Data, nil
}

func GetUserUsageSummary() (*UserUsageSummary, error) {
	if _, err := resolveAccessToken(); err != nil {
		return nil, err
	}

	urlBuilder, err := config.NewURLBuilder()
	if err != nil {
		return nil, err
	}
	usageURL, err := urlBuilder.GetSubscriptionsSummaryURL()
	if err != nil {
		return nil, err
	}

	body, err := callUserCenterAPI(http.MethodGet, usageURL, nil)
	if err != nil {
		return nil, err
	}

	var envelope userUsageSummaryEnvelope
	if err := json.Unmarshal(body, &envelope); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &UserUsageSummary{
		ActiveCount:   envelope.Data.ActiveCount,
		TotalUsedUSD:  envelope.Data.TotalUsedUSD,
		Subscriptions: envelope.Data.Subscriptions,
	}, nil
}

func GetUserUsageProgress() (*UserUsageProgress, error) {
	if _, err := resolveAccessToken(); err != nil {
		return nil, err
	}

	urlBuilder, err := config.NewURLBuilder()
	if err != nil {
		return nil, err
	}
	progressURL, err := urlBuilder.GetSubscriptionsProgressURL()
	if err != nil {
		return nil, err
	}

	body, err := callUserCenterAPI(http.MethodGet, progressURL, nil)
	if err != nil {
		return nil, err
	}

	var envelope userUsageProgressEnvelope
	if err := json.Unmarshal(body, &envelope); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &UserUsageProgress{Items: envelope.Data}, nil
}

func RedeemCode(code string) (*RedeemResult, error) {
	trimmed := strings.TrimSpace(code)
	if trimmed == "" {
		return nil, fmt.Errorf("redeem code cannot be empty")
	}

	if _, err := resolveAccessToken(); err != nil {
		return nil, err
	}

	urlBuilder, err := config.NewURLBuilder()
	if err != nil {
		return nil, err
	}
	redeemURL, err := urlBuilder.GetRedeemURL()
	if err != nil {
		return nil, err
	}

	body, err := callUserCenterAPI(http.MethodPost, redeemURL, map[string]string{"code": trimmed})
	if err != nil {
		return nil, err
	}

	var envelope struct {
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &RedeemResult{Data: envelope.Data}, nil
}
