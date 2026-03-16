package geoip

import (
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetService(t *testing.T) {
	service1 := GetService()
	service2 := GetService()
	assert.Equal(t, service1, service2, "GetService should return singleton instance")
}

func TestService_LoadDatabase(t *testing.T) {
	service := &Service{}

	// Use a guaranteed-invalid file path so test is deterministic.
	err := service.LoadDatabase("/definitely/nonexistent/path/GeoLite2-Country.mmdb")
	assert.Error(t, err, "Should fail with non-existent database path")
	assert.False(t, service.IsEnabled(), "Service should not be enabled after failed load")

	// Note: Actual database file tests would require a test database file
	// which is not included in the repository due to licensing
}

func TestService_LookupCountry_NotInitialized(t *testing.T) {
	service := &Service{enabled: false}

	_, err := service.LookupCountry(net.ParseIP("8.8.8.8"))
	assert.Error(t, err, "Should return error when service not initialized")
	assert.Contains(t, err.Error(), "not initialized")
}

func TestService_IsChina(t *testing.T) {
	service := &Service{enabled: false}

	// When not initialized, should return false (fail-safe)
	result := service.IsChina(net.ParseIP("8.8.8.8"))
	assert.False(t, result, "Should return false when service not initialized")
}

func TestService_IsEnabled(t *testing.T) {
	service := &Service{enabled: false}
	assert.False(t, service.IsEnabled())

	service.enabled = true
	assert.True(t, service.IsEnabled())
}

func TestService_EnableDisable(t *testing.T) {
	service := &Service{enabled: true}

	service.Disable()
	assert.False(t, service.IsEnabled())

	// Enable without database should fail
	err := service.Enable()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no database loaded")
}

func TestService_GetDatabasePath(t *testing.T) {
	service := &Service{dbPath: "/test/path/db.mmdb"}
	assert.Equal(t, "/test/path/db.mmdb", service.GetDatabasePath())
}

func TestService_Close(t *testing.T) {
	service := &Service{enabled: true}

	// Close without reader should not error
	err := service.Close()
	assert.NoError(t, err)
	assert.False(t, service.IsEnabled(), "Service should be disabled after close")
}

// Integration test - requires actual GeoLite2 database file
// This test is skipped by default and can be enabled manually for integration testing
func TestService_Integration(t *testing.T) {
	t.Skip("Skipping integration test - requires GeoLite2 database file")

	service := GetService()

	// Update this path to your actual test database location
	testDBPath := "~/.nonelane/GeoLite2-Country.mmdb"

	err := service.LoadDatabase(testDBPath)
	require.NoError(t, err)
	defer service.Close()

	// Test Chinese IP (Baidu DNS)
	country, err := service.LookupCountry(net.ParseIP("180.76.76.76"))
	assert.NoError(t, err)
	assert.Equal(t, "CN", country.ISOCode)
	assert.True(t, service.IsChina(net.ParseIP("180.76.76.76")))

	// Test US IP (Google DNS)
	country, err = service.LookupCountry(net.ParseIP("8.8.8.8"))
	assert.NoError(t, err)
	assert.Equal(t, "US", country.ISOCode)
	assert.False(t, service.IsChina(net.ParseIP("8.8.8.8")))

	// Test private IP (should fail)
	_, err = service.LookupCountry(net.ParseIP("192.168.1.1"))
	assert.Error(t, err)
	assert.False(t, service.IsChina(net.ParseIP("192.168.1.1")))
}
