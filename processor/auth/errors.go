package user

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"

	"aliang.one/nursorgate/common/logger"
	"aliang.one/nursorgate/processor/config"
)

var ErrRefreshTokenInvalid = errors.New("refresh token invalid")

var (
	authExpirationHandlerMu sync.RWMutex
	authExpirationHandler   func()
)

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
	notifyAuthExpirationHandler()
	logger.Warn("Authentication expired - proxy service should be stopped")
}

func SetAuthExpirationHandler(handler func()) {
	authExpirationHandlerMu.Lock()
	defer authExpirationHandlerMu.Unlock()
	authExpirationHandler = handler
}

func notifyAuthExpirationHandler() {
	authExpirationHandlerMu.RLock()
	handler := authExpirationHandler
	authExpirationHandlerMu.RUnlock()

	if handler != nil {
		handler()
	}
}

func stopProxyDueToAuthExpiration() {
	logger.Info("Stopping proxy service due to authentication expiration")
	logger.Warn("Proxy service should be stopped due to authentication expiration - checking HasLocalUserInfo() status")
}
