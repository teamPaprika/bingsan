package benchmark

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/kimuyb/bingsan/internal/api"
	"github.com/kimuyb/bingsan/internal/config"
)

// BenchmarkHealthEndpoint measures the performance of the /health endpoint.
func BenchmarkHealthEndpoint(b *testing.B) {
	cfg := benchConfig()
	server := api.NewServer(cfg, nil)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			req := httptest.NewRequest("GET", "/health", http.NoBody)
			resp, err := server.App().Test(req, -1)
			if err != nil {
				b.Fatal(err)
			}
			resp.Body.Close()
		}
	})
}

// BenchmarkReadyEndpoint measures the performance of the /ready endpoint.
func BenchmarkReadyEndpoint(b *testing.B) {
	cfg := benchConfig()
	server := api.NewServer(cfg, nil)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			req := httptest.NewRequest("GET", "/ready", http.NoBody)
			resp, err := server.App().Test(req, -1)
			if err != nil {
				b.Fatal(err)
			}
			resp.Body.Close()
		}
	})
}

// BenchmarkMetricsEndpoint measures the performance of the /metrics endpoint.
func BenchmarkMetricsEndpoint(b *testing.B) {
	cfg := benchConfig()
	server := api.NewServer(cfg, nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/metrics", http.NoBody)
		resp, err := server.App().Test(req, -1)
		if err != nil {
			b.Fatal(err)
		}
		resp.Body.Close()
	}
}

// BenchmarkHealthEndpointLatency measures latency percentiles.
func BenchmarkHealthEndpointLatency(b *testing.B) {
	cfg := benchConfig()
	server := api.NewServer(cfg, nil)

	latencies := make([]time.Duration, 0, b.N)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		start := time.Now()
		req := httptest.NewRequest("GET", "/health", http.NoBody)
		resp, err := server.App().Test(req, -1)
		latency := time.Since(start)
		if err != nil {
			b.Fatal(err)
		}
		resp.Body.Close()
		latencies = append(latencies, latency)
	}

	b.StopTimer()

	// Report custom metrics
	if len(latencies) > 0 {
		var total time.Duration
		for _, l := range latencies {
			total += l
		}
		avg := total / time.Duration(len(latencies))
		b.ReportMetric(float64(avg.Nanoseconds()), "ns/avg")
	}
}

func benchConfig() *config.Config {
	return &config.Config{
		Server: config.ServerConfig{
			Host:         "127.0.0.1",
			Port:         0,
			Debug:        false, // Disable debug for benchmarks
			Version:      "bench",
			ReadTimeout:  5 * time.Second,
			WriteTimeout: 5 * time.Second,
			IdleTimeout:  10 * time.Second,
		},
		Storage: config.StorageConfig{
			Type:      "local",
			Warehouse: "/tmp/iceberg-bench/warehouse",
		},
		Auth: config.AuthConfig{
			Enabled: false,
		},
		Catalog: config.CatalogConfig{
			Prefix: "",
		},
	}
}
