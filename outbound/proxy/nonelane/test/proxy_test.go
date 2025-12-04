package test

import (
	"context"
	"testing"
	"time"

	"nursor.org/nursorgate/outbound/proxy/cursor_h2"
)

// TestProxyCreation tests basic proxy instance creation
func TestProxyCreation(t *testing.T) {
	config := &cursor_h2.CursorH2Config{
		Addr:                 "localhost:443",
		DialTimeout:          10 * time.Second,
		ReadTimeout:          30 * time.Second,
		WriteTimeout:         30 * time.Second,
		MaxConcurrentStreams: 250,
		ConnectionPool: &cursor_h2.ConnectionPoolConfig{
			MaxConnPerHost:  4,
			MaxIdleTime:     5 * time.Minute,
			CleanupInterval: 1 * time.Minute,
		},
	}

	proxy, err := cursor_h2.New(config)
	if err != nil {
		t.Fatalf("Failed to create proxy: %v", err)
	}

	if proxy == nil {
		t.Fatal("Proxy is nil")
	}

	defer proxy.Close()

	// Verify proxy properties
	if proxy.Addr() != "localhost:443" {
		t.Errorf("Expected address localhost:443, got %s", proxy.Addr())
	}

	if proxy.Proto() != "cursor_h2" {
		t.Errorf("Expected protocol cursor_h2, got %s", proxy.Proto())
	}
}

// TestProxyConfigValidation tests configuration validation
func TestProxyConfigValidation(t *testing.T) {
	tests := []struct {
		name    string
		config  *cursor_h2.CursorH2Config
		wantErr bool
	}{
		{
			name:    "nil config",
			config:  nil,
			wantErr: true,
		},
		{
			name: "empty address",
			config: &cursor_h2.CursorH2Config{
				Addr:         "",
				DialTimeout:  10 * time.Second,
				ReadTimeout:  30 * time.Second,
				WriteTimeout: 30 * time.Second,
			},
			wantErr: true,
		},
		{
			name: "zero dial timeout",
			config: &cursor_h2.CursorH2Config{
				Addr:         "localhost:443",
				DialTimeout:  0,
				ReadTimeout:  30 * time.Second,
				WriteTimeout: 30 * time.Second,
			},
			wantErr: true,
		},
		{
			name: "valid config",
			config: &cursor_h2.CursorH2Config{
				Addr:         "localhost:443",
				DialTimeout:  10 * time.Second,
				ReadTimeout:  30 * time.Second,
				WriteTimeout: 30 * time.Second,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			proxy, err := cursor_h2.New(tt.config)

			if (err != nil) != tt.wantErr {
				t.Errorf("New() error = %v, wantErr %v", err, tt.wantErr)
			}

			if proxy != nil {
				proxy.Close()
			}
		})
	}
}

// TestProxyDialUDP tests that UDP is not supported
func TestProxyDialUDP(t *testing.T) {
	config := &cursor_h2.CursorH2Config{
		Addr:         "localhost:443",
		DialTimeout:  10 * time.Second,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	proxy, err := cursor_h2.New(config)
	if err != nil {
		t.Fatalf("Failed to create proxy: %v", err)
	}
	defer proxy.Close()

	ctx := context.Background()
	conn, err := proxy.DialUDP(ctx, "udp", "example.com:443")

	if err == nil {
		t.Fatal("Expected error for UDP dial, got nil")
	}

	if conn != nil {
		t.Fatal("Expected nil connection for UDP dial")
	}
}

// TestProxyClose tests proxy closing
func TestProxyClose(t *testing.T) {
	config := &cursor_h2.CursorH2Config{
		Addr:         "localhost:443",
		DialTimeout:  10 * time.Second,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	proxy, err := cursor_h2.New(config)
	if err != nil {
		t.Fatalf("Failed to create proxy: %v", err)
	}

	// Close should succeed
	err = proxy.Close()
	if err != nil {
		t.Errorf("Close() error = %v", err)
	}

	// Second close should also succeed (idempotent)
	err = proxy.Close()
	if err != nil {
		t.Errorf("Second Close() error = %v", err)
	}
}

// TestProxyStats tests GetStats functionality
func TestProxyStats(t *testing.T) {
	config := &cursor_h2.CursorH2Config{
		Addr:                 "localhost:443",
		DialTimeout:          10 * time.Second,
		ReadTimeout:          30 * time.Second,
		WriteTimeout:         30 * time.Second,
		MaxConcurrentStreams: 250,
	}

	proxy, err := cursor_h2.New(config)
	if err != nil {
		t.Fatalf("Failed to create proxy: %v", err)
	}
	defer proxy.Close()

	stats := proxy.GetStats()
	if stats == nil {
		t.Fatal("GetStats() returned nil")
	}

	// Verify expected stats keys for core mTLS functionality
	expectedKeys := []string{"addr", "proto", "closed", "connection_pool"}
	for _, key := range expectedKeys {
		if _, exists := stats[key]; !exists {
			t.Errorf("Missing expected stats key: %s", key)
		}
	}
}
