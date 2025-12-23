package benchmark

import (
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/kimuyb/bingsan/internal/api"
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
