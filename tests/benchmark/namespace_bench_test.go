package benchmark

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/teamPaprika/bingsan/internal/api"
)

// BenchmarkListNamespaces measures the performance of listing namespaces.
func BenchmarkListNamespaces(b *testing.B) {
	cfg := benchConfig()
	server := api.NewServer(cfg, nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/v1/namespaces", http.NoBody)
		req.Header.Set("Content-Type", "application/json")
		resp, err := server.App().Test(req, -1)
		if err != nil {
			b.Fatal(err)
		}
		resp.Body.Close()
	}
}

// BenchmarkListNamespacesParallel measures parallel namespace listing performance.
func BenchmarkListNamespacesParallel(b *testing.B) {
	cfg := benchConfig()
	server := api.NewServer(cfg, nil)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			req := httptest.NewRequest("GET", "/v1/namespaces", http.NoBody)
			req.Header.Set("Content-Type", "application/json")
			resp, err := server.App().Test(req, -1)
			if err != nil {
				b.Fatal(err)
			}
			resp.Body.Close()
		}
	})
}

// BenchmarkGetNamespaceMetadata measures getting namespace metadata.
func BenchmarkGetNamespaceMetadata(b *testing.B) {
	cfg := benchConfig()
	server := api.NewServer(cfg, nil)

	// Note: This will likely return 404 without a real namespace,
	// but it still measures the endpoint's baseline performance

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/v1/namespaces/bench_test", http.NoBody)
		req.Header.Set("Content-Type", "application/json")
		resp, err := server.App().Test(req, -1)
		if err != nil {
			b.Fatal(err)
		}
		resp.Body.Close()
	}
}

// BenchmarkNamespaceExists measures the HEAD namespace check.
func BenchmarkNamespaceExists(b *testing.B) {
	cfg := benchConfig()
	server := api.NewServer(cfg, nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("HEAD", "/v1/namespaces/bench_test", http.NoBody)
		resp, err := server.App().Test(req, -1)
		if err != nil {
			b.Fatal(err)
		}
		resp.Body.Close()
	}
}
