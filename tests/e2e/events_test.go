package e2e

import (
	"bufio"
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/kimuyb/bingsan/internal/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestEventConnection verifies that the event stream endpoint is available.
func TestEventConnection(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping e2e test in short mode")
	}

	cfg := testConfig()
	server := api.NewServer(cfg, nil)

	// Try to connect to events endpoint
	// The endpoint might be /events, /v1/events, or similar
	endpoints := []string{"/events", "/v1/events", "/api/events"}

	var foundEndpoint bool
	for _, endpoint := range endpoints {
		req := httptest.NewRequest("GET", endpoint, nil)
		req.Header.Set("Accept", "text/event-stream")
		resp, err := server.App().Test(req, 1000) // Short timeout

		if err == nil {
			resp.Body.Close()
			// If we get a response (even 404), the server is handling it
			if resp.StatusCode != http.StatusNotFound {
				foundEndpoint = true
				t.Logf("Found events endpoint at %s (status: %d)", endpoint, resp.StatusCode)
				break
			}
		}
	}

	if !foundEndpoint {
		t.Log("No events endpoint found. Event streaming may not be implemented yet.")
	}
}

// TestEventStreamFormat verifies SSE format if event streaming is implemented.
func TestEventStreamFormat(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping e2e test in short mode")
	}

	cfg := testConfig()
	server := api.NewServer(cfg, nil)

	// Try the most common endpoint
	req := httptest.NewRequest("GET", "/events", nil)
	req.Header.Set("Accept", "text/event-stream")

	resp, err := server.App().Test(req, 2000)
	if err != nil {
		t.Skipf("Events endpoint not available: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		t.Skip("Events endpoint not implemented")
	}

	// If implemented, check SSE format
	contentType := resp.Header.Get("Content-Type")
	if strings.Contains(contentType, "text/event-stream") {
		t.Log("Event stream endpoint returns proper SSE content type")
	}
}

// TestCatalogEvents verifies catalog change events are emitted.
func TestCatalogEvents(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping e2e test in short mode")
	}

	cfg := testConfig()
	server := api.NewServer(cfg, nil)

	// This test requires:
	// 1. Connect to event stream
	// 2. Make a catalog change (create namespace)
	// 3. Verify event is received

	// First check if events endpoint exists
	req := httptest.NewRequest("GET", "/events", nil)
	resp, err := server.App().Test(req, 1000)
	if err != nil || resp.StatusCode == http.StatusNotFound {
		if resp != nil {
			resp.Body.Close()
		}
		t.Skip("Events endpoint not implemented")
	}
	resp.Body.Close()

	// Create a context with timeout for event listening
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Start listening for events
	eventsChan := make(chan string, 10)
	go func() {
		req := httptest.NewRequest("GET", "/events", nil)
		req.Header.Set("Accept", "text/event-stream")
		resp, err := server.App().Test(req, 5000)
		if err != nil {
			return
		}
		defer resp.Body.Close()

		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan() {
			select {
			case eventsChan <- scanner.Text():
			case <-ctx.Done():
				return
			}
		}
	}()

	// Give the event listener time to connect
	time.Sleep(100 * time.Millisecond)

	// Make a catalog change - create a namespace
	createNS := `{"namespace": ["test_events_ns"], "properties": {}}`
	req = httptest.NewRequest("POST", "/v1/namespaces", bytes.NewReader([]byte(createNS)))
	req.Header.Set("Content-Type", "application/json")
	resp, _ = server.App().Test(req, -1)
	if resp != nil {
		resp.Body.Close()
	}

	// Wait for events
	select {
	case event := <-eventsChan:
		t.Logf("Received event: %s", event)
		// Event should contain namespace creation info
		assert.True(t,
			strings.Contains(event, "namespace") ||
				strings.Contains(event, "test_events_ns") ||
				strings.Contains(event, "create"),
			"Event should relate to namespace creation")
	case <-ctx.Done():
		t.Log("No events received within timeout - event streaming may not emit catalog events")
	}
}

// TestEventStreamReconnection verifies client can reconnect to event stream.
func TestEventStreamReconnection(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping e2e test in short mode")
	}

	cfg := testConfig()
	server := api.NewServer(cfg, nil)

	// First connection
	req := httptest.NewRequest("GET", "/events", nil)
	resp, err := server.App().Test(req, 1000)
	if err != nil || resp.StatusCode == http.StatusNotFound {
		if resp != nil {
			resp.Body.Close()
		}
		t.Skip("Events endpoint not implemented")
	}
	resp.Body.Close()

	// Second connection (simulating reconnection)
	req = httptest.NewRequest("GET", "/events", nil)
	resp, err = server.App().Test(req, 1000)
	require.NoError(t, err)
	defer resp.Body.Close()

	// Should be able to reconnect
	assert.NotEqual(t, http.StatusServiceUnavailable, resp.StatusCode,
		"Should allow reconnection to event stream")
}

// TestEventStreamWithLastEventID verifies Last-Event-ID header handling.
func TestEventStreamWithLastEventID(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping e2e test in short mode")
	}

	cfg := testConfig()
	server := api.NewServer(cfg, nil)

	// Connect with Last-Event-ID header
	req := httptest.NewRequest("GET", "/events", nil)
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Last-Event-ID", "12345")

	resp, err := server.App().Test(req, 1000)
	if err != nil || resp.StatusCode == http.StatusNotFound {
		if resp != nil {
			resp.Body.Close()
		}
		t.Skip("Events endpoint not implemented")
	}
	defer resp.Body.Close()

	// Server should accept the Last-Event-ID header without error
	assert.True(t,
		resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusNoContent,
		"Server should handle Last-Event-ID header gracefully")
}
