package user

import (
	"strings"
	"sync"
)

var (
	mu          sync.Mutex
	accessToken []string
)

func GetCurrentAuthorizationHeader() string {
	current := GetCurrentUserInfo()
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
