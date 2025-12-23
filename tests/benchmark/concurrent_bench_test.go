package benchmark

import (
	"encoding/json"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/kimuyb/bingsan/internal/api"
	"github.com/kimuyb/bingsan/internal/pool"
)

// BenchmarkConcurrentConnections measures performance under concurrent load.
func BenchmarkConcurrentConnections(b *testing.B) {
	cfg := benchConfig()
	server := api.NewServer(cfg, nil)

	concurrencyLevels := []int{1, 10, 50, 100}

	for _, concurrency := range concurrencyLevels {
		b.Run(string(rune('0'+concurrency/10))+string(rune('0'+concurrency%10))+"_concurrent", func(b *testing.B) {
			b.SetParallelism(concurrency)
			b.ResetTimer()

			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					req := httptest.NewRequest("GET", "/health", nil)
					resp, err := server.App().Test(req, -1)
					if err != nil {
						b.Fatal(err)
					}
					resp.Body.Close()
				}
			})
		})
	}
}

// BenchmarkSustainedLoad measures performance under sustained concurrent load.
func BenchmarkSustainedLoad(b *testing.B) {
	cfg := benchConfig()
	server := api.NewServer(cfg, nil)

	// Run sustained load for the benchmark duration
	var totalRequests int64
	var totalErrors int64

	b.ResetTimer()

	var wg sync.WaitGroup
	workers := 10

	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < b.N/workers; i++ {
				req := httptest.NewRequest("GET", "/health", nil)
				resp, err := server.App().Test(req, 1000)
				if err != nil {
					atomic.AddInt64(&totalErrors, 1)
					continue
				}
				resp.Body.Close()
				atomic.AddInt64(&totalRequests, 1)
			}
		}()
	}

	wg.Wait()

	b.ReportMetric(float64(atomic.LoadInt64(&totalRequests)), "requests")
	b.ReportMetric(float64(atomic.LoadInt64(&totalErrors)), "errors")
}

// BenchmarkMixedWorkload measures performance with mixed endpoint access.
func BenchmarkMixedWorkload(b *testing.B) {
	cfg := benchConfig()
	server := api.NewServer(cfg, nil)

	endpoints := []string{
		"/health",
		"/ready",
		"/metrics",
		"/v1/config",
		"/v1/namespaces",
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			endpoint := endpoints[i%len(endpoints)]
			req := httptest.NewRequest("GET", endpoint, nil)
			req.Header.Set("Content-Type", "application/json")
			resp, err := server.App().Test(req, -1)
			if err != nil {
				b.Fatal(err)
			}
			resp.Body.Close()
			i++
		}
	})
}

// BenchmarkConnectionThroughput measures maximum request throughput.
func BenchmarkConnectionThroughput(b *testing.B) {
	cfg := benchConfig()
	server := api.NewServer(cfg, nil)

	start := time.Now()
	var requestCount int64

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			req := httptest.NewRequest("GET", "/health", nil)
			resp, err := server.App().Test(req, -1)
			if err != nil {
				continue
			}
			resp.Body.Close()
			atomic.AddInt64(&requestCount, 1)
		}
	})
	b.StopTimer()

	elapsed := time.Since(start)
	rps := float64(atomic.LoadInt64(&requestCount)) / elapsed.Seconds()
	b.ReportMetric(rps, "rps")
}

// BenchmarkBurstTraffic measures performance handling burst traffic.
func BenchmarkBurstTraffic(b *testing.B) {
	cfg := benchConfig()
	server := api.NewServer(cfg, nil)

	burstSize := 100

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Create burst of concurrent requests
		var wg sync.WaitGroup
		var errors int64

		for j := 0; j < burstSize; j++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				req := httptest.NewRequest("GET", "/health", nil)
				resp, err := server.App().Test(req, 1000)
				if err != nil {
					atomic.AddInt64(&errors, 1)
					return
				}
				resp.Body.Close()
			}()
		}

		wg.Wait()

		if errors > 0 {
			b.Logf("Burst %d: %d errors out of %d requests", i, errors, burstSize)
		}
	}
}

// =============================================================================
// US2 Concurrent Request Benchmarks
// =============================================================================
// These benchmarks measure concurrent request handling performance.
// Target: Support 100 concurrent requests without performance degradation.
// =============================================================================

// BenchmarkConcurrent100Serialization measures concurrent JSON serialization with 100 goroutines.
// Target: No performance degradation at 100 concurrent requests
func BenchmarkConcurrent100Serialization(b *testing.B) {
	bp := pool.NewBufferPool(nil)

	metadata := map[string]any{
		"format-version":        2,
		"table-uuid":            "550e8400-e29b-41d4-a716-446655440000",
		"location":              "s3://warehouse/default/test_table",
		"last-updated-ms":       1703376000000,
		"properties":            map[string]string{"owner": "test"},
		"schemas":               []any{map[string]any{"type": "struct", "schema-id": 0, "fields": generateFields(20)}},
		"current-schema-id":     0,
		"partition-specs":       []any{},
		"default-spec-id":       0,
		"sort-orders":           []any{},
		"default-sort-order-id": 0,
	}

	b.SetParallelism(100)
	b.ReportAllocs()
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			buf := bp.Get()
			encoder := json.NewEncoder(buf)
			if err := encoder.Encode(metadata); err != nil {
				b.Fatal(err)
			}
			_ = buf.Bytes()
			bp.Put(buf)
		}
	})
}

// BenchmarkConcurrent100VsBaseline compares pooled vs non-pooled under 100 concurrent requests.
func BenchmarkConcurrent100VsBaseline(b *testing.B) {
	metadata := map[string]any{
		"format-version":        2,
		"table-uuid":            "550e8400-e29b-41d4-a716-446655440000",
		"location":              "s3://warehouse/default/test_table",
		"last-updated-ms":       1703376000000,
		"properties":            map[string]string{"owner": "test"},
		"schemas":               []any{map[string]any{"type": "struct", "schema-id": 0, "fields": generateFields(20)}},
		"current-schema-id":     0,
	}

	b.Run("WithPool", func(b *testing.B) {
		bp := pool.NewBufferPool(nil)

		b.SetParallelism(100)
		b.ReportAllocs()
		b.ResetTimer()

		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				buf := bp.Get()
				encoder := json.NewEncoder(buf)
				_ = encoder.Encode(metadata)
				bp.Put(buf)
			}
		})
	})

	b.Run("WithoutPool", func(b *testing.B) {
		b.SetParallelism(100)
		b.ReportAllocs()
		b.ResetTimer()

		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				_, err := json.Marshal(metadata)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	})
}

// BenchmarkConcurrentScaling measures how pool performance scales with concurrency.
func BenchmarkConcurrentScaling(b *testing.B) {
	bp := pool.NewBufferPool(nil)

	metadata := map[string]any{
		"format-version":        2,
		"table-uuid":            "550e8400-e29b-41d4-a716-446655440000",
		"location":              "s3://warehouse/default/test_table",
		"last-updated-ms":       1703376000000,
		"properties":            map[string]string{"owner": "test"},
		"schemas":               []any{map[string]any{"type": "struct", "schema-id": 0, "fields": generateFields(20)}},
		"current-schema-id":     0,
	}

	concurrencyLevels := []int{1, 10, 50, 100, 200}

	for _, level := range concurrencyLevels {
		name := ""
		if level < 100 {
			name = "0"
		}
		if level < 10 {
			name = "00"
		}
		name += string(rune('0'+level/100)) + string(rune('0'+(level/10)%10)) + string(rune('0'+level%10)) + "_concurrent"

		b.Run(name, func(b *testing.B) {
			b.SetParallelism(level)
			b.ReportAllocs()
			b.ResetTimer()

			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					buf := bp.Get()
					encoder := json.NewEncoder(buf)
					_ = encoder.Encode(metadata)
					bp.Put(buf)
				}
			})
		})
	}
}

// BenchmarkConcurrentLargeSchema measures large schema serialization under concurrent load.
func BenchmarkConcurrentLargeSchema(b *testing.B) {
	bp := pool.NewBufferPool(nil)

	// Large schema with 150 columns
	metadata := map[string]any{
		"format-version":        2,
		"table-uuid":            "550e8400-e29b-41d4-a716-446655440000",
		"location":              "s3://warehouse/default/large_table",
		"last-updated-ms":       1703376000000,
		"properties":            map[string]string{"owner": "test"},
		"schemas":               []any{map[string]any{"type": "struct", "schema-id": 0, "fields": generateFields(150)}},
		"current-schema-id":     0,
		"snapshots":             generateSnapshots(20),
	}

	b.SetParallelism(100)
	b.ReportAllocs()
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			buf := bp.Get()
			encoder := json.NewEncoder(buf)
			if err := encoder.Encode(metadata); err != nil {
				b.Fatal(err)
			}
			_ = buf.Bytes()
			bp.Put(buf)
		}
	})
}
