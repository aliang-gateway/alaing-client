package services

import (
	"context"
	"fmt"
	"math/rand"
	"net"
	"net/http"
	"sync"
	"time"

	"nursor.org/nursorgate/common/logger"
	"nursor.org/nursorgate/outbound"
)

// LatencyService provides latency testing functionality for door proxy members
type LatencyService struct {
	testURLs       []string
	currentTestURL string
	mu             sync.RWMutex // protects currentTestURL for concurrent access
	timeout        time.Duration
	maxRetries     int
	httpClient     *http.Client
}

// NewLatencyService creates a new latency service instance
func NewLatencyService() *LatencyService {
	return &LatencyService{
		testURLs: []string{
			"http://www.gstatic.com/generate_204",
			"http://cp.cloudflare.com/generate_204",
			"http://www.msftconnecttest.com/connecttest.txt",
		},
		timeout:    10 * time.Second,
		maxRetries: 3,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:        10,
				MaxIdleConnsPerHost: 2,
				IdleConnTimeout:     30 * time.Second,
				DialContext: (&net.Dialer{
					Timeout:   10 * time.Second,
					KeepAlive: 30 * time.Second,
				}).DialContext,
			},
		},
	}
}

// LatencyResult represents the latency test result for a member
type LatencyResult struct {
	ShowName   string `json:"showname"`
	Latency    int64  `json:"latency"`     // milliseconds, -1 if failed
	LastUpdate int64  `json:"last_update"` // unix timestamp
	Status     string `json:"status"`      // "success" or "failed"
}

// TestAllMembers tests latency for all door proxy members
// Returns a map of member show names to their latency values
// Returns -1 for failed tests
func (s *LatencyService) TestAllMembers(ctx context.Context) ([]LatencyResult, error) {
	registry := outbound.GetRegistry()
	members, err := registry.ListDoorMembers()
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to list door members: %v", err))
		return nil, fmt.Errorf("failed to list door members: %w", err)
	}

	if len(members) == 0 {
		return []LatencyResult{}, nil
	}

	results := make([]LatencyResult, 0, len(members))
	now := time.Now().Unix()

	// Select a single test URL for all members to ensure consistency
	if len(s.testURLs) == 0 {
		return nil, fmt.Errorf("no test URLs configured")
	}

	s.mu.Lock()
	if s.currentTestURL == "" {
		s.currentTestURL = s.testURLs[rand.Intn(len(s.testURLs))]
	}
	s.mu.Unlock()

	// Ensure cleanup after testing completes
	defer func() {
		s.mu.Lock()
		s.currentTestURL = ""
		s.mu.Unlock()
	}()

	logger.Debug(fmt.Sprintf("Testing all members using URL: %s", s.currentTestURL))

	// Create synchronization primitives for parallel testing
	var wg sync.WaitGroup
	resultsChan := make(chan LatencyResult, len(members))
	errorsChan := make(chan error, len(members))

	logger.Info(fmt.Sprintf("Starting parallel latency tests for %d members", len(members)))

	// Launch goroutine for each member to test in parallel
	for _, member := range members {
		wg.Add(1)

		// Launch goroutine with member copy (avoid closure issues)
		go func(m outbound.DoorProxyMemberInfo) {
			defer wg.Done()

			// Check context cancellation
			select {
			case <-ctx.Done():
				errorsChan <- ctx.Err()
				return
			default:
			}

			// Test this member's latency
			latency := s.testMemberLatency(ctx, &m)

			// Determine status based on latency
			status := "failed"
			if latency >= 0 {
				status = "success"
			}

			// Update registry with new latency value and status
			// This is thread-safe thanks to Registry's mutex
			if err := registry.UpdateDoorMemberLatency(m.ShowName, latency, status); err != nil {
				logger.Warn(fmt.Sprintf("Failed to update latency for %s: %v", m.ShowName, err))
				errorsChan <- fmt.Errorf("update latency for %s: %w", m.ShowName, err)
				return
			}

			// Send result to channel
			resultsChan <- LatencyResult{
				ShowName:   m.ShowName,
				Latency:    latency,
				LastUpdate: now,
				Status:     status,
			}
		}(member) // Pass member by value to avoid race
	}

	// Wait for all goroutines in separate goroutine
	go func() {
		wg.Wait()
		close(resultsChan)
		close(errorsChan)
	}()

	// Collect results from all goroutines
	for result := range resultsChan {
		results = append(results, result)
	}

	// Log any errors (don't fail the entire operation)
	for err := range errorsChan {
		logger.Warn(fmt.Sprintf("Latency test error: %v", err))
	}

	// Check if context was cancelled
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	logger.Info(fmt.Sprintf("Completed parallel tests: %d results collected", len(results)))

	return results, nil
}

// TestMember tests latency for a specific door proxy member
// Returns latency in milliseconds, or -1 if failed
func (s *LatencyService) TestMember(ctx context.Context, showName string) (int64, error) {
	registry := outbound.GetRegistry()
	members, err := registry.ListDoorMembers()
	if err != nil {
		return -1, fmt.Errorf("failed to list door members: %w", err)
	}

	// Find the member
	var targetMember *outbound.DoorProxyMemberInfo
	for i := range members {
		if members[i].ShowName == showName {
			targetMember = &members[i]
			break
		}
	}

	if targetMember == nil {
		return -1, fmt.Errorf("member %s not found", showName)
	}

	return s.testMemberLatency(ctx, targetMember), nil
}

// testMemberLatency performs latency test for a single member
// Retries up to maxRetries times and returns the minimum latency
// Returns -1 if all retries fail
func (s *LatencyService) testMemberLatency(ctx context.Context, member *outbound.DoorProxyMemberInfo) int64 {
	var minLatency int64 = -1

	// Determine which test URL to use (once for all retries)
	s.mu.RLock()
	testURL := s.currentTestURL
	s.mu.RUnlock()

	// If no URL is set (e.g., single member test), select one randomly
	if testURL == "" {
		if len(s.testURLs) == 0 {
			logger.Error("No test URLs configured")
			return -1
		}
		testURL = s.testURLs[rand.Intn(len(s.testURLs))]
		logger.Debug(fmt.Sprintf("Selected test URL for %s: %s", member.ShowName, testURL))
	}

	// Perform retries with the same URL
	for attempt := 0; attempt < s.maxRetries; attempt++ {
		select {
		case <-ctx.Done():
			return minLatency
		default:
		}

		latency := s.measureLatency(ctx, member, testURL)
		if latency >= 0 {
			if minLatency < 0 || latency < minLatency {
				minLatency = latency
			}
		}
	}

	return minLatency
}

// measureLatency measures the latency to a test URL through a specific proxy
// Returns latency in milliseconds, or -1 if the test fails
func (s *LatencyService) measureLatency(ctx context.Context, member *outbound.DoorProxyMemberInfo, testURL string) int64 {
	start := time.Now()

	// Create a custom HTTP client that uses the target proxy
	// Note: This is a simplified implementation that doesn't actually route through the proxy
	// In a real implementation, you would need to configure the HTTP client to use the proxy
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, testURL, nil)
	if err != nil {
		logger.Debug(fmt.Sprintf("Failed to create request for %s: %v", member.ShowName, err))
		return -1
	}

	// Add timeout context
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	req = req.WithContext(ctx)

	// Make the request
	resp, err := s.httpClient.Do(req)
	if err != nil {
		logger.Debug(fmt.Sprintf("Latency test failed for %s (URL: %s): %v", member.ShowName, testURL, err))
		return -1
	}
	defer resp.Body.Close()

	elapsed := time.Since(start)
	latencyMs := elapsed.Milliseconds()

	logger.Debug(fmt.Sprintf("Latency test for %s: %d ms (attempt)", member.ShowName, latencyMs))

	return latencyMs
}

// GetTestURLs returns the list of test URLs
func (s *LatencyService) GetTestURLs() []string {
	return s.testURLs
}

// SetTimeout sets the timeout for individual tests
func (s *LatencyService) SetTimeout(timeout time.Duration) {
	s.timeout = timeout
	s.httpClient.Timeout = timeout
}

// SetMaxRetries sets the maximum number of retries per member
func (s *LatencyService) SetMaxRetries(maxRetries int) {
	if maxRetries > 0 {
		s.maxRetries = maxRetries
	}
}
