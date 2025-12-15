package latency

import (
	"context"
	"fmt"
	"sync"
	"time"

	"nursor.org/nursorgate/app/http/services"
	"nursor.org/nursorgate/common/logger"
	"nursor.org/nursorgate/outbound"
)

// LatencyTestManager manages periodic latency testing for door proxy members
type LatencyTestManager struct {
	latencyService *services.LatencyService
	testMutex      sync.Mutex
	isRunning      bool
	stopChan       chan struct{}
	testTicker     *time.Ticker
	lastTestTime   int64
	testInterval   time.Duration
	minInterval    time.Duration // Minimum interval between tests
}

// NewLatencyTestManager creates a new latency test manager instance
func NewLatencyTestManager(latencyService *services.LatencyService) *LatencyTestManager {
	return &LatencyTestManager{
		latencyService: latencyService,
		stopChan:       make(chan struct{}),
		testInterval:   60 * time.Second,  // Test every 60 seconds
		minInterval:    60 * time.Second,  // Minimum 60 seconds between tests
		isRunning:      false,
		lastTestTime:   0,
	}
}

// Start initializes and starts the background latency testing
// Performs an initial test immediately, then runs tests at regular intervals
func (m *LatencyTestManager) Start() error {
	m.testMutex.Lock()
	defer m.testMutex.Unlock()

	if m.isRunning {
		return fmt.Errorf("latency test manager is already running")
	}

	logger.Info("[LatencyManager] Starting latency test manager")

	// Perform initial test
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	results, err := m.latencyService.TestAllMembers(ctx)
	cancel()

	if err != nil {
		logger.Warn(fmt.Sprintf("[LatencyManager] Initial latency test failed: %v", err))
		// Continue even if initial test fails
	} else {
		successCount := 0
		for _, result := range results {
			if result.Status == "success" {
				successCount++
			}
		}
		logger.Info(fmt.Sprintf("[LatencyManager] Initial test completed: %d/%d members tested successfully",
			successCount, len(results)))
	}

	m.lastTestTime = time.Now().Unix()

	// Start background ticker
	m.testTicker = time.NewTicker(m.testInterval)
	m.isRunning = true

	go m.runBackgroundTests()

	logger.Info("[LatencyManager] Latency test manager started successfully")
	return nil
}

// Stop gracefully stops the background latency testing
func (m *LatencyTestManager) Stop() error {
	m.testMutex.Lock()
	defer m.testMutex.Unlock()

	if !m.isRunning {
		return fmt.Errorf("latency test manager is not running")
	}

	logger.Info("[LatencyManager] Stopping latency test manager")

	m.isRunning = false

	// Stop the ticker
	if m.testTicker != nil {
		m.testTicker.Stop()
	}

	// Signal the background goroutine to stop
	close(m.stopChan)

	logger.Info("[LatencyManager] Latency test manager stopped")
	return nil
}

// TestNow performs an immediate latency test if not already running
// Respects the minimum interval between tests
func (m *LatencyTestManager) TestNow() error {
	m.testMutex.Lock()
	defer m.testMutex.Unlock()

	now := time.Now().Unix()
	minIntervalSeconds := int64(m.minInterval.Seconds())

	// Check if enough time has passed since last test
	if now-m.lastTestTime < minIntervalSeconds {
		return fmt.Errorf("test already performed recently, please wait at least %d seconds",
			minIntervalSeconds-(now-m.lastTestTime))
	}

	logger.Info("[LatencyManager] Manual latency test triggered")

	// Perform the test
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	results, err := m.latencyService.TestAllMembers(ctx)
	cancel()

	if err != nil {
		logger.Error(fmt.Sprintf("[LatencyManager] Manual latency test failed: %v", err))
		return fmt.Errorf("latency test failed: %w", err)
	}

	m.lastTestTime = now

	successCount := 0
	for _, result := range results {
		if result.Status == "success" {
			successCount++
		}
	}

	logger.Info(fmt.Sprintf("[LatencyManager] Manual test completed: %d/%d members tested successfully",
		successCount, len(results)))

	return nil
}

// runBackgroundTests is the background goroutine that performs periodic latency tests
func (m *LatencyTestManager) runBackgroundTests() {
	for {
		select {
		case <-m.testTicker.C:
			// Check if it's time to run a test
			now := time.Now().Unix()
			if now-m.lastTestTime >= int64(m.minInterval) {
				m.performBackgroundTest()
			}

		case <-m.stopChan:
			logger.Debug("[LatencyManager] Background test goroutine stopping")
			return
		}
	}
}

// performBackgroundTest performs a latency test in the background
func (m *LatencyTestManager) performBackgroundTest() {
	m.testMutex.Lock()
	defer m.testMutex.Unlock()

	logger.Debug("[LatencyManager] Starting background latency test")

	// Check once more if manager is still running
	if !m.isRunning {
		logger.Debug("[LatencyManager] Manager stopped, skipping test")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	results, err := m.latencyService.TestAllMembers(ctx)
	cancel()

	if err != nil {
		logger.Warn(fmt.Sprintf("[LatencyManager] Background latency test failed: %v", err))
		return
	}

	m.lastTestTime = time.Now().Unix()

	successCount := 0
	for _, result := range results {
		if result.Status == "success" {
			successCount++
		}
	}

	logger.Debug(fmt.Sprintf("[LatencyManager] Background test completed: %d/%d members tested successfully",
		successCount, len(results)))

	// Log registry state for debugging
	registry := outbound.GetRegistry()
	members, err := registry.ListDoorMembers()
	if err == nil {
		for _, member := range members {
			logger.Debug(fmt.Sprintf("[LatencyManager] %s: %d ms (last_update: %d)",
				member.ShowName, member.Latency, member.LastUpdate))
		}
	}
}

// IsRunning returns whether the manager is currently running
func (m *LatencyTestManager) IsRunning() bool {
	m.testMutex.Lock()
	defer m.testMutex.Unlock()
	return m.isRunning
}

// SetTestInterval sets the interval between automatic tests
func (m *LatencyTestManager) SetTestInterval(interval time.Duration) {
	m.testMutex.Lock()
	defer m.testMutex.Unlock()

	if interval < 10*time.Second {
		logger.Warn(fmt.Sprintf("[LatencyManager] Test interval too short (%v), setting to 10 seconds", interval))
		interval = 10 * time.Second
	}

	m.testInterval = interval
	if m.testTicker != nil {
		m.testTicker.Reset(interval)
	}

	logger.Info(fmt.Sprintf("[LatencyManager] Test interval set to %v", interval))
}

// SetMinInterval sets the minimum interval between tests
func (m *LatencyTestManager) SetMinInterval(seconds int64) {
	m.testMutex.Lock()
	defer m.testMutex.Unlock()

	if seconds < 10 {
		logger.Warn(fmt.Sprintf("[LatencyManager] Min interval too short (%d seconds), setting to 10", seconds))
		seconds = 10
	}

	m.minInterval = time.Duration(seconds) * time.Second
	logger.Info(fmt.Sprintf("[LatencyManager] Min interval set to %d seconds", seconds))
}

// GetLastTestTime returns the timestamp of the last test
func (m *LatencyTestManager) GetLastTestTime() int64 {
	m.testMutex.Lock()
	defer m.testMutex.Unlock()
	return m.lastTestTime
}
