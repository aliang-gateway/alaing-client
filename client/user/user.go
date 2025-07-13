package user

import (
	"bytes"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"

	"nursor.org/nursorgate/common/logger"
)

var (
	mu           sync.RWMutex
	userId       int
	accessToken  string
	userToken    string
	oncePostLock sync.Mutex
	postInFlight bool
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
	changed := accessToken != newToken
	if changed {
		accessToken = newToken
	}
	mu.Unlock()

	if changed {
		triggerAuthPost()
	}
}

// SetUserToken 设置userToken（线程安全）
func SetUserToken(token string) {
	mu.Lock()
	defer mu.Unlock()
	userToken = token
}

// triggerAuthPost 发起POST请求（同时只允许一个进行）
func triggerAuthPost() {
	oncePostLock.Lock()
	if postInFlight {
		oncePostLock.Unlock()
		return // 已有请求在执行
	}
	postInFlight = true
	oncePostLock.Unlock()

	go func() {
		defer func() {
			oncePostLock.Lock()
			postInFlight = false
			oncePostLock.Unlock()
		}()

		mu.RLock()
		tokenPayload := strings.Replace(userToken, "Bearer ", "", 0)
		authHeader := fmt.Sprintf("Bearer %s", userToken)
		mu.RUnlock()

		body := []byte(`{"client_id":"` + tokenPayload + `"}`)
		req, err := http.NewRequest("POST", "https://api.nursor.org/api/user/auth/info/binding_v2", bytes.NewBuffer(body))

		if err != nil {
			logger.Error(fmt.Printf("❌ Failed to create request:", err))
			return
		}
		req.Header.Set("Authorization", authHeader)
		req.Header.Set("Content-Type", "application/json")

		resp, err := http.DefaultClient.Do(req)
		if err != nil && resp.StatusCode < 299 {
			logger.Error(fmt.Printf("❌ Failed to send request:", err))
			return
		}
		defer resp.Body.Close()

		log.Println("✅ Auth post completed, status:", resp.Status)
	}()
}
