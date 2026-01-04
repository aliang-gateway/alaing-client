package user

import (
	"sync"
)

var (
	mu          sync.RWMutex
	accessToken []string
	innerToken  string
)

func SetInnerToken(token string) {
	mu.Lock()
	defer mu.Unlock()
	innerToken = token
}

func GetInnerToken() string {
	mu.RLock()
	defer mu.RUnlock()
	return innerToken
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
