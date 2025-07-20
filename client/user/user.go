package user

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"nursor.org/nursorgate/common/logger"
)

var (
	mu          sync.RWMutex
	userId      int
	accessToken []string
	userToken   string
	postLock    sync.Mutex
	postRunning bool
)

// SetUserId 设置用户ID（线程安全）
func SetUserId(uid int) {
	mu.Lock()
	defer mu.Unlock()
	userId = uid
}

// GetUserId 获取用户ID（线程安全）
func GetUserId() int {
	mu.RLock()
	defer mu.RUnlock()
	return userId
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
		triggerAuthPost(newToken)
	}
}

// SetUserToken 设置userToken（线程安全）
func SetUserToken(token string) {
	mu.Lock()
	defer mu.Unlock()
	userToken = token
}

// 只允许一个请求在运行，多余的触发会被忽略
func triggerAuthPost(newAccessToken string) {
	postLock.Lock()
	if postRunning {
		postLock.Unlock()
		return
	}
	postRunning = true
	postLock.Unlock()

	go func(token string) {
		defer func() {
			postLock.Lock()
			postRunning = false
			postLock.Unlock()
		}()

		// 提取 token
		tokenPayload := strings.TrimPrefix(token, "Bearer ")

		mu.RLock()
		authHeader := fmt.Sprintf("Bearer %s", userToken)
		mu.RUnlock()

		body := []byte(fmt.Sprintf(`{"client_id":"%s"}`, tokenPayload))
		req, err := http.NewRequest("POST", "https://api.nursor.org/api/user/auth/info/binding_v2", bytes.NewBuffer(body))
		if err != nil {
			logger.Error("❌ Failed to create request:", err)
			return
		}

		req.Header.Set("Authorization", authHeader)
		req.Header.Set("Content-Type", "application/json")

		client := &http.Client{
			Timeout: 10 * time.Second,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: true, // 注意：线上别这么干
				},
			},
		}

		resp, err := client.Do(req)
		if err != nil {
			logger.Error("❌ Request failed:", err)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode >= 300 {
			logger.Error("❌ Unexpected status:", resp.Status)
			return
		}

		logger.Info("✅ Auth post completed, status:", resp.Status)

		accessToken = append(accessToken, token)
	}(newAccessToken)
}
