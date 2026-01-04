package handlers

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"nursor.org/nursorgate/app/http/common"
	"nursor.org/nursorgate/app/http/services"
	"nursor.org/nursorgate/common/logger"
	"nursor.org/nursorgate/outbound"
)

// LatencyHandler handles HTTP requests for proxy latency testing
type LatencyHandler struct {
	latencyService *services.LatencyService
	testMutex      *sync.Mutex
	lastTestTime   int64
	testTimeout    time.Duration
}

// NewLatencyHandler creates a new latency handler instance
func NewLatencyHandler(latencyService *services.LatencyService) *LatencyHandler {
	return &LatencyHandler{
		latencyService: latencyService,
		testMutex:      &sync.Mutex{},
		lastTestTime:   0,
		testTimeout:    30 * time.Second, // Allow up to 30 seconds for all tests to complete
	}
}

// TestingResponse represents the response when a test is already in progress
type TestingResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

// HandleTestAllMembers handles POST /api/proxy/door/test-latency
// Tests latency for all door proxy members
func (lh *LatencyHandler) HandleTestAllMembers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		common.Error(w, http.StatusMethodNotAllowed, "Method not allowed", nil)
		return
	}

	// Check if a test is already in progress
	if !lh.testMutex.TryLock() {
		logger.Info("Latency test already in progress, returning status")
		common.Success(w, TestingResponse{
			Status:  "testing",
			Message: "Latency test in progress, please wait...",
		})
		return
	}
	defer lh.testMutex.Unlock()

	// Check if too soon since last test (prevent spam)
	now := time.Now().Unix()
	if now-lh.lastTestTime < 10 {
		logger.Info(fmt.Sprintf("Last test was %d seconds ago, skipping", now-lh.lastTestTime))
		common.Success(w, TestingResponse{
			Status:  "testing",
			Message: "Test already completed recently, please wait a bit longer...",
		})
		return
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), lh.testTimeout)
	defer cancel()

	logger.Info("Starting latency test for all door members")

	// Perform the actual test
	results, err := lh.latencyService.TestAllMembers(ctx)
	if err != nil {
		logger.Error(fmt.Sprintf("Latency test failed: %v", err))
		common.ErrorInternalServer(w, fmt.Sprintf("Failed to test latency: %v", err), nil)
		return
	}

	// Update last test time
	lh.lastTestTime = now

	// Calculate statistics for logging
	successCount := 0
	for _, result := range results {
		if result.Status == "success" {
			successCount++
		}
	}

	logger.Info(fmt.Sprintf("Latency test completed: %d total, %d success, %d failed",
		len(results), successCount, len(results)-successCount))

	// Return the same structure as /api/proxy/list
	registry := outbound.GetRegistry()
	proxyList := registry.ListWithInfo()

	common.Success(w, proxyList)
}

// HandleTestMember handles POST /api/proxy/door/test-latency/{showName}
// Tests latency for a specific door proxy member
func (lh *LatencyHandler) HandleTestMember(w http.ResponseWriter, r *http.Request, showName string) {
	if r.Method != http.MethodPost {
		common.Error(w, http.StatusMethodNotAllowed, "Method not allowed", nil)
		return
	}

	// Check if a test is already in progress
	if !lh.testMutex.TryLock() {
		common.Success(w, TestingResponse{
			Status:  "testing",
			Message: "Latency test in progress, please wait...",
		})
		return
	}
	defer lh.testMutex.Unlock()

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), lh.testTimeout)
	defer cancel()

	logger.Info(fmt.Sprintf("Starting latency test for member: %s", showName))

	// Perform the test
	latency, err := lh.latencyService.TestMember(ctx, showName)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to test member %s: %v", showName, err))
		common.ErrorBadRequest(w, fmt.Sprintf("Failed to test member: %v", err), nil)
		return
	}

	// Build response
	status := "success"
	if latency < 0 {
		status = "failed"
	}

	result := services.LatencyResult{
		ShowName:   showName,
		Latency:    latency,
		LastUpdate: time.Now().Unix(),
		Status:     status,
	}

	logger.Info(fmt.Sprintf("Latency test for %s completed: %d ms", showName, latency))

	common.Success(w, result)
}
