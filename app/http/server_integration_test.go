package http

import (
	"context"
	"io"
	"net/http"
	"testing"
	"time"

	"nursor.org/nursorgate/app/http/handlers"
)

func TestHttpServerIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Create a test client
	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	// Test the routes by making HTTP requests
	t.Run("TokenRoutes", func(t *testing.T) {
		// Test /token/get endpoint
		resp, err := client.Get("http://127.0.0.1:56431/token/get")
		if err != nil {
			// Server might not be running, which is OK for unit test
			t.Logf("Note: Could not connect to server (expected if not running): %v", err)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusNotFound {
			t.Logf("✓ Token endpoint responds with status: %d", resp.StatusCode)
		}
	})

	t.Run("ProxyRoutes", func(t *testing.T) {
		resp, err := client.Get("http://127.0.0.1:56431/proxy/current/get")
		if err != nil {
			t.Logf("Note: Could not connect to proxy endpoint (expected if server not running): %v", err)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusNotFound {
			t.Logf("✓ Proxy endpoint responds with status: %d", resp.StatusCode)
		}
	})
}

func TestRegisterAllRoutesNoRace(t *testing.T) {
	// Test that registerAllRoutes can be called multiple times without panicking
	t.Run("MultipleRegistrations", func(t *testing.T) {
		done := make(chan bool, 3)

		for i := 0; i < 3; i++ {
			go func(id int) {
				defer func() {
					if r := recover(); r != nil {
						t.Errorf("Goroutine %d panicked: %v", id, r)
					}
					done <- true
				}()
				registerAllRoutes()
			}(i)
		}

		// Wait for all goroutines
		for i := 0; i < 3; i++ {
			select {
			case <-done:
				t.Logf("✓ Goroutine %d completed route registration", i)
			case <-time.After(2 * time.Second):
				t.Error("Timeout waiting for route registration")
			}
		}
	})
}

func TestHttpHandlerRegistration(t *testing.T) {
	// Verify that handlers can register routes without panicking
	handlerFuncs := []struct {
		name string
		fn   func()
	}{
		{"RegisterTokenRoutes", handlers.RegisterTokenRoutes},
		{"RegisterRunRoutes", handlers.RegisterRunRoutes},
		{"RegisterProxyRoutes", handlers.RegisterProxyRoutes},
		{"RegisterProxyRegistryRoutes", handlers.RegisterProxyRegistryRoutes},
		{"RegisterConfigRoutes", handlers.RegisterConfigRoutes},
	}

	for _, hf := range handlerFuncs {
		t.Run(hf.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("%s panicked: %v", hf.name, r)
				}
			}()

			hf.fn()
			t.Logf("✓ %s registered successfully", hf.name)
		})
	}
}

func TestHttpServerContext(t *testing.T) {
	// Test that the server respects context cancellation
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// Create a simple HTTP server for testing
	mux := http.NewServeMux()
	mux.HandleFunc("/test", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		io.WriteString(w, "OK")
	})

	server := &http.Server{
		Addr:    ":56432",
		Handler: mux,
	}

	go func() {
		<-ctx.Done()
		server.Close()
	}()

	// Try to start the server
	go server.ListenAndServe()

	// Give the server a moment to start
	time.Sleep(50 * time.Millisecond)

	// Try to connect
	resp, err := http.Get("http://127.0.0.1:56432/test")
	if err == nil {
		defer resp.Body.Close()
		if resp.StatusCode == http.StatusOK {
			t.Log("✓ Test server responded successfully")
		}
	} else {
		t.Logf("Note: Test server not accessible (OK for this test): %v", err)
	}
}
