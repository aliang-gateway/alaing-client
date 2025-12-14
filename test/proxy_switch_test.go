package test

//
//import (
//	"context"
//	"fmt"
//	"io"
//	"net"
//	"sync"
//	"sync/atomic"
//	"testing"
//	"time"
//
//	"nursor.org/nursorgate/common/logger"
//	"nursor.org/nursorgate/outbound"
//	"nursor.org/nursorgate/processor/config"
//)
//
//// Test 1: New connection during proxy switch
//// This test simulates the most common failure case:
//// 1. Start a baseline connection
//// 2. Trigger UpdateDoorProxies() to change the proxy
//// 3. Create new connections within 100ms (danger window)
//// 4. Verify if these connections succeed (if they fail, registry replacement is the cause)
//func TestNewConnectionDuringProxySwitch(t *testing.T) {
//	t.Log("[TEST 1] New connection during proxy switch")
//
//	// Start mock server for baseline
//	listener, err := net.Listen("tcp", "127.0.0.1:0")
//	if err != nil {
//		t.Fatalf("Failed to start mock server: %v", err)
//	}
//	defer listener.Close()
//
//	addr := listener.Addr().String()
//	t.Logf("Mock server listening on %s", addr)
//
//	// Accept baseline connection in background
//	go func() {
//		for {
//			conn, err := listener.Accept()
//			if err != nil {
//				return // listener closed
//			}
//			defer conn.Close()
//			io.Copy(io.Discard, conn)
//		}
//	}()
//
//	// Create baseline connection
//	baseline, err := net.Dial("tcp", addr)
//	if err != nil {
//		t.Fatalf("Failed to create baseline connection: %v", err)
//	}
//	defer baseline.Close()
//
//	t.Log("✓ Baseline connection established")
//
//	// Record version before switch
//	//versionBefore := outbound.GetCurrentRegistryVersion()
//	//t.Logf("Registry version before switch: %d", versionBefore)
//
//	// Trigger proxy update (simulated)
//	// In real scenario, this would call UpdateDoorProxies()
//	// For testing, we just increment version to simulate an update
//	time.Sleep(50 * time.Millisecond)
//
//	// Try creating new connections in danger window (100ms)
//	successCount := 0
//	failureCount := 0
//	failureVersions := make([]int64, 0)
//
//	for i := 0; i < 10; i++ {
//		// Try to connect
//		conn, err := net.Dial("tcp", addr)
//		//versionDuringConnection := outbound.GetCurrentRegistryVersion()
//
//		if err != nil {
//			failureCount++
//			//failureVersions = append(failureVersions, versionDuringConnection)
//			//t.Logf("  ✗ Connection %d failed (version=%d): %v", i, versionDuringConnection, err)
//		} else {
//			successCount++
//			conn.Close()
//			//t.Logf("  ✓ Connection %d succeeded (version=%d)", i, versionDuringConnection)
//		}
//
//		time.Sleep(10 * time.Millisecond)
//	}
//
//	//versionAfter := outbound.GetCurrentRegistryVersion()
//	//t.Logf("Registry version after danger window: %d", versionAfter)
//
//	// Analysis
//	t.Logf("\n[ANALYSIS] New connections during proxy switch:")
//	//t.Logf("  - Version before: %d, after: %d (changed=%v)", versionBefore, versionAfter, versionBefore != versionAfter)
//	t.Logf("  - Successes: %d/10", successCount)
//	t.Logf("  - Failures: %d/10", failureCount)
//
//	if failureCount > 0 {
//		t.Logf("  - Failure versions: %v", failureVersions)
//	}
//
//	// In normal operation, this should have near 100% success rate
//	// If failures > 20%, it indicates registry replacement is causing issues
//	if failureCount > 2 {
//		t.Logf("⚠️  HIGH FAILURE RATE DETECTED - registry replacement may be the root cause")
//	}
//}
//
//// Test 2: Long connection interrupted by proxy update
//// This test simulates a long-lived connection (large file transfer)
//// and monitors if it gets interrupted by proxy updates
//func TestLongConnectionInterruptedByProxyUpdate(t *testing.T) {
//	t.Log("[TEST 2] Long connection interrupted by proxy update")
//
//	// Start mock server
//	listener, err := net.Listen("tcp", "127.0.0.1:0")
//	if err != nil {
//		t.Fatalf("Failed to start mock server: %v", err)
//	}
//	defer listener.Close()
//
//	addr := listener.Addr().String()
//
//	// Server that sends data continuously for 5 seconds
//	go func() {
//		for {
//			conn, err := listener.Accept()
//			if err != nil {
//				return
//			}
//
//			go func(c net.Conn) {
//				defer c.Close()
//				ticker := time.NewTicker(100 * time.Millisecond)
//				defer ticker.Stop()
//
//				for i := 0; i < 50; i++ { // 50 * 100ms = 5 seconds
//					<-ticker.C
//					c.Write([]byte("PING\n"))
//				}
//			}(conn)
//		}
//	}()
//
//	// Start long connection
//	conn, err := net.Dial("tcp", addr)
//	if err != nil {
//		t.Fatalf("Failed to create connection: %v", err)
//	}
//	defer conn.Close()
//
//	//versionStart := outbound.GetCurrentRegistryVersion()
//	//t.Logf("Long connection started, registry version=%d", versionStart)
//
//	// Monitor connection for 5 seconds
//	completedReads := 0
//	failedReads := 0
//	var failureTime time.Duration
//	failureOccurred := false
//
//	ticker := time.NewTicker(100 * time.Millisecond)
//	defer ticker.Stop()
//
//	startTime := time.Now()
//	for {
//		<-ticker.C
//		elapsed := time.Since(startTime)
//
//		if elapsed > 5*time.Second {
//			break
//		}
//
//		// Try to read from connection
//		conn.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
//		buf := make([]byte, 1024)
//		_, err := conn.Read(buf)
//
//		//currentVersion := outbound.GetCurrentRegistryVersion()
//
//		if err == io.EOF {
//			t.Logf("  Connection closed normally at %v", elapsed)
//			break
//		} else if err != nil {
//			if !failureOccurred {
//				failedReads++
//				failureOccurred = true
//				failureTime = elapsed
//				//versionAtFailure := currentVersion
//				//t.Logf("  ✗ Connection error at %v (version=%d→%d): %v",
//				//	failureTime, versionStart, versionAtFailure, err)
//			}
//		} else {
//			if !failureOccurred {
//				completedReads++
//			}
//		}
//	}
//
//	//versionEnd := outbound.GetCurrentRegistryVersion()
//
//	t.Logf("\n[ANALYSIS] Long connection during proxy update:")
//	t.Logf("  - Completed reads: %d", completedReads)
//	t.Logf("  - Failed reads: %d", failedReads)
//	//t.Logf("  - Registry version: %d→%d", versionStart, versionEnd)
//
//	if failureOccurred {
//		t.Logf("  ⚠️  Connection was interrupted at %v", failureTime)
//		t.Logf("  - Indicates proxy replacement may have caused interruption")
//	}
//}
//
//// Test 3: 50 concurrent requests with periodic proxy switching
//// This test simulates heavy concurrent load with proxy updates
//// to measure error rates and identify failure patterns
//func TestConcurrentRequestsWithPeriodicProxySwitching(t *testing.T) {
//	t.Log("[TEST 3] 50 concurrent requests with periodic proxy switching")
//
//	// Start mock server
//	listener, err := net.Listen("tcp", "127.0.0.1:0")
//	if err != nil {
//		t.Fatalf("Failed to start mock server: %v", err)
//	}
//	defer listener.Close()
//
//	addr := listener.Addr().String()
//
//	// Simple echo server
//	go func() {
//		for {
//			conn, err := listener.Accept()
//			if err != nil {
//				return
//			}
//			go func(c net.Conn) {
//				defer c.Close()
//				io.Copy(c, c)
//			}(conn)
//		}
//	}()
//
//	// Metrics
//	successCount := int32(0)
//	failureCount := int32(0)
//	failuresByVersion := make(map[string]int32)
//	var failuresMutex sync.Mutex
//
//	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
//	defer cancel()
//
//	wg := sync.WaitGroup{}
//
//	// Launch 50 concurrent workers
//	for i := 0; i < 50; i++ {
//		wg.Add(1)
//		go func(id int) {
//			defer wg.Done()
//
//			for {
//				select {
//				case <-ctx.Done():
//					return
//				default:
//				}
//
//				versionBefore := outbound.GetCurrentRegistryVersion()
//
//				// Try to connect and send data
//				conn, err := net.Dial("tcp", addr)
//				versionAfter := outbound.GetCurrentRegistryVersion()
//
//				if err != nil {
//					atomic.AddInt32(&failureCount, 1)
//
//					// Record version mismatch if applicable
//					if versionBefore != versionAfter {
//						failuresMutex.Lock()
//						key := fmt.Sprintf("version=%d→%d", versionBefore, versionAfter)
//						failuresByVersion[key]++
//						failuresMutex.Unlock()
//
//						logger.Debug(fmt.Sprintf("[TEST 3] Worker %d: connection failed with version change %d→%d", id, versionBefore, versionAfter))
//					}
//				} else {
//					atomic.AddInt32(&successCount, 1)
//					conn.Close()
//				}
//
//				time.Sleep(50 * time.Millisecond)
//			}
//		}(i)
//	}
//
//	// Wait for workers
//	wg.Wait()
//
//	totalAttempts := atomic.LoadInt32(&successCount) + atomic.LoadInt32(&failureCount)
//
//	t.Logf("\n[ANALYSIS] Concurrent requests with periodic proxy switching:")
//	t.Logf("  - Total attempts: %d", totalAttempts)
//	t.Logf("  - Successes: %d (%.1f%%)", successCount, 100*float64(successCount)/float64(totalAttempts))
//	t.Logf("  - Failures: %d (%.1f%%)", failureCount, 100*float64(failureCount)/float64(totalAttempts))
//
//	if len(failuresByVersion) > 0 {
//		t.Logf("  - Failures by version mismatch:")
//		for versionKey, count := range failuresByVersion {
//			t.Logf("    • %s: %d", versionKey, count)
//		}
//		t.Logf("  ⚠️  VERSION MISMATCH DETECTED - proxy group replacement is likely root cause")
//	}
//}
//
//// Test 4: Registry version correlation analysis
//// This test directly monitors registry version changes and logs them
//// for later analysis of version mismatch patterns
//func TestRegistryVersionCorrelationAnalysis(t *testing.T) {
//	t.Log("[TEST 4] Registry version correlation analysis")
//
//	versionHistory := make([]struct {
//		elapsed time.Duration
//		version int64
//	}, 0)
//
//	// Record version every 100ms for 3 seconds
//	ticker := time.NewTicker(100 * time.Millisecond)
//	defer ticker.Stop()
//
//	startTime := time.Now()
//
//	for i := 0; i < 30; i++ {
//		<-ticker.C
//		//currentVersion := outbound.GetCurrentRegistryVersion()
//		elapsed := time.Since(startTime)
//
//		//versionHistory = append(versionHistory, struct {
//		//	elapsed time.Duration
//		//	version int64
//		//}{elapsed, currentVersion})
//
//		if i > 0 && versionHistory[i].version != versionHistory[i-1].version {
//			t.Logf("  [VERSION CHANGE] At %v: %d→%d",
//				elapsed, versionHistory[i-1].version, versionHistory[i].version)
//		}
//	}
//
//	// Analyze version stability
//	versionChanges := 0
//	for i := 1; i < len(versionHistory); i++ {
//		if versionHistory[i].version != versionHistory[i-1].version {
//			versionChanges++
//		}
//	}
//
//	t.Logf("\n[ANALYSIS] Registry version correlation:")
//	t.Logf("  - Total observations: %d", len(versionHistory))
//	t.Logf("  - Version changes: %d", versionChanges)
//	t.Logf("  - Stability: %.1f%% (no changes)", 100*float64(len(versionHistory)-versionChanges)/float64(len(versionHistory)))
//
//	if versionChanges > 0 {
//		t.Logf("  ℹ️  Registry was updated %d times during test window", versionChanges)
//	}
//}
//
//// Test 5: Concurrent GetDoor() + UpdateDoorProxies() race testing
//// This test aggressively exercises the concurrent access patterns
//// to detect if there are race conditions in the registry
//func TestConcurrentRegistryAccess(t *testing.T) {
//	t.Log("[TEST 5] Concurrent GetDoor() + Registry access race testing")
//
//	registry := outbound.GetRegistry()
//	successCount := int32(0)
//	raceConditionCount := int32(0)
//	var raceMutex sync.Mutex
//
//	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
//	defer cancel()
//
//	wg := sync.WaitGroup{}
//
//	// Thread A: Continuously call GetRegistry and read version
//	for i := 0; i < 5; i++ {
//		wg.Add(1)
//		go func(id int) {
//			defer wg.Done()
//
//			for {
//				select {
//				case <-ctx.Done():
//					return
//				default:
//				}
//
//				versionBefore := outbound.GetCurrentRegistryVersion()
//
//				// Try to get door proxy
//				_, err := registry.GetDoor()
//
//				versionAfter := outbound.GetCurrentRegistryVersion()
//
//				if err == nil {
//					atomic.AddInt32(&successCount, 1)
//				}
//
//				// Check for race condition
//				// This is a simplified check - in real scenario, look for consistent version during operation
//				if versionBefore != versionAfter {
//					raceMutex.Lock()
//					raceConditionCount++
//					raceMutex.Unlock()
//
//					logger.Debug(fmt.Sprintf("[TEST 5] Potential race condition detected: version %d→%d during GetDoor()",
//						versionBefore, versionAfter))
//				}
//
//				time.Sleep(10 * time.Millisecond)
//			}
//		}(i)
//	}
//
//	// Thread B: Continuously get registry members (read-heavy operation)
//	for i := 0; i < 5; i++ {
//		wg.Add(1)
//		go func(id int) {
//			defer wg.Done()
//
//			for {
//				select {
//				case <-ctx.Done():
//					return
//				default:
//				}
//
//				members, err := config.GetDoorProxyMembers()
//
//				if err == nil {
//					atomic.AddInt32(&successCount, 1)
//				}
//
//				if len(members) > 0 {
//					// Simulate processing - increases likelihood of hitting race window
//					time.Sleep(5 * time.Millisecond)
//				}
//			}
//		}(i)
//	}
//
//	// Wait for test to complete
//	wg.Wait()
//
//	successOps := atomic.LoadInt32(&successCount)
//
//	t.Logf("\n[ANALYSIS] Concurrent registry access:")
//	t.Logf("  - Successful operations: %d", successOps)
//	t.Logf("  - Potential race conditions detected: %d", raceConditionCount)
//
//	if raceConditionCount > 0 {
//		t.Logf("  ⚠️  RACE CONDITIONS DETECTED - indicate concurrent access safety issues")
//	}
//}
//
//// LogAnalysisSummary helps analyze collected test logs for version mismatch patterns
//// This can be called after all tests run to summarize findings
//func LogAnalysisSummary(t *testing.T) {
//	t.Log(`
//[SUMMARY] Diagnostic Test Results Analysis
//
//Look for these patterns in the logs:
//
//1. VERSION MISMATCH PATTERN:
//   [TRACE] Relay START [version=5, ...]
//   [TRACE] Registry doorGroup REPLACED [version=6, ...]
//   [TRACE] relayStream ERROR [version=5→6, ...]
//
//   → Indicates DoorProxyGroup replacement caused connection failure
//
//2. TIME CORRELATION:
//   Registry update at T=1000ms
//   Error at T=1005ms
//
//   → 5ms delay indicates race condition in active relay
//
//3. CONCURRENT ACCESS ISSUES:
//   Multiple VERSION CHANGE logs during single operation
//
//   → Indicates unsafe concurrent access to registry
//
//EXPECTED FINDINGS (if proxy switching is the issue):
//✓ Test 1: High failure rate (>20%) on new connections in danger window
//✓ Test 2: Connection interruption at version change point
//✓ Test 3: Failures correlate with version=X→Y changes
//✓ Test 4: Version changes occur during test run
//✓ Test 5: Race conditions detected during concurrent access
//
//NEXT STEPS if findings confirm root cause:
//1. Implement Phase 2 - Graceful Connection Drain
//2. Add AcquireRef/ReleaseRef to DoorProxyGroup
//3. Ensure old instances stay alive until all connections complete
//4. Re-run tests to verify fix effectiveness
//	`)
//}
