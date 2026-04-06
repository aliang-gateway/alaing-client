package user

import (
	"strings"
	"sync"

	"aliang.one/nursorgate/common/logger"
)

var (
	mu          sync.Mutex
	accessToken []string
)

func GetCurrentAuthorizationHeader() string {
	current := resolveUserInfoForAuthorizationHeader()
	if current == nil {
		return ""
	}

	accessToken := strings.TrimSpace(current.AccessToken)
	if accessToken == "" {
		return ""
	}

	tokenType := strings.TrimSpace(current.TokenType)
	if tokenType == "" {
		tokenType = "Bearer"
	}

	return tokenType + " " + accessToken
}

func resolveUserInfoForAuthorizationHeader() *UserInfo {
	current := GetCurrentUserInfo()
	if current != nil && strings.TrimSpace(current.AccessToken) != "" {
		return current
	}

	loaded, err := LoadUserInfo()
	if err != nil {
		return current
	}
	if loaded == nil || strings.TrimSpace(loaded.AccessToken) == "" {
		return current
	}

	logger.Debug("Authorization header resolved from persisted user info")
	return loaded
}

// SetAccessToken 设置accessToken，如果变更则触发POST（线程安全 + 单请求）
func SetAccessToken(newToken string) {
	mu.Lock()
	isNewComming := true
	for _, token := range accessToken {
		if token == newToken {
			isNewComming = false
			break
		}
	}
	mu.Unlock()

	if isNewComming {
		// triggerAuthPost(newToken)
	}
}
