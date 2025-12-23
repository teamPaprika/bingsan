package benchmark

import (
	"bytes"
	"net/http/httptest"
	"testing"

	"github.com/kimuyb/bingsan/internal/api"
)

// BenchmarkListTables measures the performance of listing tables.
func BenchmarkListTables(b *testing.B) {
	cfg := benchConfig()
	server := api.NewServer(cfg, nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/v1/namespaces/bench_ns/tables", nil)
		req.Header.Set("Content-Type", "application/json")
		resp, err := server.App().Test(req, -1)
		if err != nil {
			b.Fatal(err)
		}
		resp.Body.Close()
	}
}

// BenchmarkListTablesParallel measures parallel table listing performance.
func BenchmarkListTablesParallel(b *testing.B) {
	cfg := benchConfig()
	server := api.NewServer(cfg, nil)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			req := httptest.NewRequest("GET", "/v1/namespaces/bench_ns/tables", nil)
			req.Header.Set("Content-Type", "application/json")
			resp, err := server.App().Test(req, -1)
			if err != nil {
				b.Fatal(err)
			}
			resp.Body.Close()
		}
	})
}

// BenchmarkGetTableMetadata measures getting table metadata.
func BenchmarkGetTableMetadata(b *testing.B) {
	cfg := benchConfig()
	server := api.NewServer(cfg, nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/v1/namespaces/bench_ns/tables/bench_table", nil)
		req.Header.Set("Content-Type", "application/json")
		resp, err := server.App().Test(req, -1)
		if err != nil {
			b.Fatal(err)
		}
		resp.Body.Close()
	}
}

// BenchmarkTableExists measures the HEAD table check.
func BenchmarkTableExists(b *testing.B) {
	cfg := benchConfig()
	server := api.NewServer(cfg, nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("HEAD", "/v1/namespaces/bench_ns/tables/bench_table", nil)
		resp, err := server.App().Test(req, -1)
		if err != nil {
			b.Fatal(err)
		}
		resp.Body.Close()
	}
}

// BenchmarkCreateTableRequest measures table creation request parsing/validation.
func BenchmarkCreateTableRequest(b *testing.B) {
	cfg := benchConfig()
	server := api.NewServer(cfg, nil)

	// Minimal table creation payload
	payload := []byte(`{
		"name": "bench_table",
		"schema": {
			"type": "struct",
			"schema-id": 0,
			"fields": [
				{"id": 1, "name": "id", "required": true, "type": "long"}
			]
		}
	}`)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("POST", "/v1/namespaces/bench_ns/tables", bytes.NewReader(payload))
		req.Header.Set("Content-Type", "application/json")
		resp, err := server.App().Test(req, -1)
		if err != nil {
			b.Fatal(err)
		}
		resp.Body.Close()
	}
}
