package benchmark

import (
	"net/http/httptest"
	"runtime"
	"testing"

	"github.com/kimuyb/bingsan/internal/api"
)

// BenchmarkMemoryAllocation measures memory allocations per request.
func BenchmarkMemoryAllocation(b *testing.B) {
	cfg := benchConfig()
	server := api.NewServer(cfg, nil)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/health", nil)
		resp, err := server.App().Test(req, -1)
		if err != nil {
			b.Fatal(err)
		}
		resp.Body.Close()
	}
}

// BenchmarkMemoryAllocationConfig measures memory allocations for config endpoint.
func BenchmarkMemoryAllocationConfig(b *testing.B) {
	cfg := benchConfig()
	server := api.NewServer(cfg, nil)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/v1/config", nil)
		req.Header.Set("Content-Type", "application/json")
		resp, err := server.App().Test(req, -1)
		if err != nil {
			b.Fatal(err)
		}
		resp.Body.Close()
	}
}

// BenchmarkMemoryAllocationNamespaces measures memory allocations for namespace listing.
func BenchmarkMemoryAllocationNamespaces(b *testing.B) {
	cfg := benchConfig()
	server := api.NewServer(cfg, nil)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/v1/namespaces", nil)
		req.Header.Set("Content-Type", "application/json")
		resp, err := server.App().Test(req, -1)
		if err != nil {
			b.Fatal(err)
		}
		resp.Body.Close()
	}
}

// BenchmarkMemoryUnderLoad measures memory usage under sustained load.
func BenchmarkMemoryUnderLoad(b *testing.B) {
	cfg := benchConfig()
	server := api.NewServer(cfg, nil)

	// Force GC and get baseline
	runtime.GC()
	var m1 runtime.MemStats
	runtime.ReadMemStats(&m1)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/health", nil)
		resp, err := server.App().Test(req, -1)
		if err != nil {
			b.Fatal(err)
		}
		resp.Body.Close()
	}

	b.StopTimer()

	// Get final memory stats
	var m2 runtime.MemStats
	runtime.ReadMemStats(&m2)

	// Report custom metrics
	allocBytes := m2.TotalAlloc - m1.TotalAlloc
	allocPerOp := float64(allocBytes) / float64(b.N)
	b.ReportMetric(allocPerOp, "B/op_total")

	// Report heap growth
	heapGrowth := int64(m2.HeapAlloc) - int64(m1.HeapAlloc)
	b.ReportMetric(float64(heapGrowth), "heap_growth")
}

// BenchmarkMemoryMetrics measures memory allocations for metrics endpoint.
func BenchmarkMemoryMetrics(b *testing.B) {
	cfg := benchConfig()
	server := api.NewServer(cfg, nil)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/metrics", nil)
		resp, err := server.App().Test(req, -1)
		if err != nil {
			b.Fatal(err)
		}
		resp.Body.Close()
	}
}

// BenchmarkGCPressure measures GC pressure under load.
func BenchmarkGCPressure(b *testing.B) {
	cfg := benchConfig()
	server := api.NewServer(cfg, nil)

	// Get initial GC stats
	var m1 runtime.MemStats
	runtime.ReadMemStats(&m1)
	initialGC := m1.NumGC

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/health", nil)
		resp, err := server.App().Test(req, -1)
		if err != nil {
			b.Fatal(err)
		}
		resp.Body.Close()
	}

	b.StopTimer()

	// Get final GC stats
	var m2 runtime.MemStats
	runtime.ReadMemStats(&m2)

	gcRuns := m2.NumGC - initialGC
	gcPerOp := float64(gcRuns) / float64(b.N)
	b.ReportMetric(gcPerOp, "gc/op")
	b.ReportMetric(float64(m2.PauseTotalNs-m1.PauseTotalNs)/float64(b.N), "gc_pause_ns/op")
}

// BenchmarkLargeResponseMemory measures memory for large response handling.
func BenchmarkLargeResponseMemory(b *testing.B) {
	cfg := benchConfig()
	server := api.NewServer(cfg, nil)

	// Metrics endpoint typically returns larger responses
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/metrics", nil)
		resp, err := server.App().Test(req, -1)
		if err != nil {
			b.Fatal(err)
		}

		// Read the entire response body to measure full allocation
		buf := make([]byte, 4096)
		for {
			_, err := resp.Body.Read(buf)
			if err != nil {
				break
			}
		}
		resp.Body.Close()
	}
}
