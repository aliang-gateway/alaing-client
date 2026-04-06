package user

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"aliang.one/nursorgate/common/logger"
	"aliang.one/nursorgate/processor/config"
)

var ErrRefreshTokenInvalid = errors.New("refresh token invalid")

type authAPIErrorEnvelope struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Reason  string `json:"reason"`
}

func classifyRefreshSessionFailure(statusCode int, body []byte) error {
	if statusCode != http.StatusUnauthorized {
		return nil
	}

	var envelope authAPIErrorEnvelope
	if err := json.Unmarshal(body, &envelope); err != nil {
		return nil
	}

	message := strings.ToLower(strings.TrimSpace(envelope.Message))
	reason := strings.ToUpper(strings.TrimSpace(envelope.Reason))
	if envelope.Code == http.StatusUnauthorized && (reason == "REFRESH_TOKEN_INVALID" || strings.Contains(message, "invalid refresh token")) {
		return ErrRefreshTokenInvalid
	}

	return nil
}

func clearLocalSessionAfterInvalidRefreshToken() {
	StopTokenRefresh()

	if err := DeleteUserInfo(); err != nil {
		logger.Warn(fmt.Sprintf("Failed to clear invalid local auth session: %v", err))
		SetCurrentUserInfo(nil)
	}

	config.SetHasLocalUserInfo(false)
	logger.Info("Local auth session cleared after invalid refresh token")
}
