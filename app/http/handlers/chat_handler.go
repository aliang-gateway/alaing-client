package handlers

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"nursor.org/nursorgate/app/http/common"
)

type ChatHandler struct{}

const (
	chatRequestMaxBytes   int64 = 256 * 1024
	chatHistoryMaxEntries       = 20
)

func NewChatHandler() *ChatHandler {
	return &ChatHandler{}
}

type ChatRequest struct {
	Message string            `json:"message"`
	History []ChatHistoryItem `json:"history"`
}

type ChatHistoryItem struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openAIChatPayload struct {
	Model    string              `json:"model"`
	Messages []openAIMessageItem `json:"messages"`
}

type openAIMessageItem struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openAIResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

func (h *ChatHandler) HandleCompletions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		common.Error(w, http.StatusMethodNotAllowed, "Method not allowed", nil)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, chatRequestMaxBytes)

	var req ChatRequest
	if err := common.DecodeRequest(r, &req); err != nil {
		common.ErrorBadRequest(w, "Invalid request format", nil)
		return
	}

	message := strings.TrimSpace(req.Message)
	if message == "" {
		common.ErrorBadRequest(w, "Message is required", nil)
		return
	}

	apiKey := strings.TrimSpace(os.Getenv("OPENAI_API_KEY"))
	if apiKey == "" {
		common.Success(w, map[string]interface{}{
			"reply": "AI 服务暂未配置（缺少 OPENAI_API_KEY），请先配置后重试。",
		})
		return
	}

	messages := make([]openAIMessageItem, 0, len(req.History)+1)
	for _, item := range req.History {
		role := strings.TrimSpace(strings.ToLower(item.Role))
		if role != "user" && role != "assistant" {
			continue
		}
		content := strings.TrimSpace(item.Content)
		if content == "" {
			continue
		}
		messages = append(messages, openAIMessageItem{Role: role, Content: content})
	}
	if len(messages) > chatHistoryMaxEntries {
		messages = messages[len(messages)-chatHistoryMaxEntries:]
	}

	if len(messages) == 0 || messages[len(messages)-1].Role != "user" || strings.TrimSpace(messages[len(messages)-1].Content) != message {
		messages = append(messages, openAIMessageItem{Role: "user", Content: message})
	}

	payload := openAIChatPayload{
		Model:    "gpt-4o-mini",
		Messages: messages,
	}

	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		log.Printf("[chat] marshal payload failed: %v", err)
		common.ErrorInternalServer(w, "Failed to build chat payload", nil)
		return
	}

	httpClient := &http.Client{Timeout: 45 * time.Second}
	upstreamReq, err := http.NewRequest(http.MethodPost, "https://api.openai.com/v1/chat/completions", bytes.NewReader(bodyBytes))
	if err != nil {
		log.Printf("[chat] build upstream request failed: %v", err)
		common.ErrorInternalServer(w, "Failed to build upstream request", nil)
		return
	}
	upstreamReq.Header.Set("Content-Type", "application/json")
	upstreamReq.Header.Set("Authorization", "Bearer "+apiKey)

	upstreamResp, err := httpClient.Do(upstreamReq)
	if err != nil {
		log.Printf("[chat] call AI service failed: %v", err)
		common.ErrorInternalServer(w, "Failed to call AI service", nil)
		return
	}
	defer upstreamResp.Body.Close()

	respBody, err := io.ReadAll(upstreamResp.Body)
	if err != nil {
		log.Printf("[chat] read AI response failed: %v", err)
		common.ErrorInternalServer(w, "Failed to read AI response", nil)
		return
	}

	if upstreamResp.StatusCode < 200 || upstreamResp.StatusCode >= 300 {
		log.Printf("[chat] AI service returned status=%d body=%s", upstreamResp.StatusCode, string(respBody))
		common.ErrorInternalServer(w, "AI service returned error", nil)
		return
	}

	var parsed openAIResponse
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		log.Printf("[chat] parse AI response failed: %v body=%s", err, string(respBody))
		common.ErrorInternalServer(w, "Invalid AI response payload", nil)
		return
	}

	if len(parsed.Choices) == 0 {
		common.ErrorInternalServer(w, "AI response is empty", nil)
		return
	}

	reply := strings.TrimSpace(parsed.Choices[0].Message.Content)
	if reply == "" {
		common.ErrorInternalServer(w, "AI response content is empty", nil)
		return
	}

	common.Success(w, map[string]interface{}{
		"reply": reply,
	})
}
