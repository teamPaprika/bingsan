package benchmark

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/kimuyb/bingsan/internal/api"
	"github.com/kimuyb/bingsan/internal/pool"
)

// BenchmarkListTables measures the performance of listing tables.
func BenchmarkListTables(b *testing.B) {
	cfg := benchConfig()
	server := api.NewServer(cfg, nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/v1/namespaces/bench_ns/tables", http.NoBody)
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
			req := httptest.NewRequest("GET", "/v1/namespaces/bench_ns/tables", http.NoBody)
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
		req := httptest.NewRequest("GET", "/v1/namespaces/bench_ns/tables/bench_table", http.NoBody)
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
		req := httptest.NewRequest("HEAD", "/v1/namespaces/bench_ns/tables/bench_table", http.NoBody)
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

// =============================================================================
// Table Latency Benchmarks (US2)
// =============================================================================
// These benchmarks measure latency targets for table metadata operations:
// - Serialization < 1ms
// - Average response < 50ms
// - Large schema operations < 200ms
// =============================================================================

// BenchmarkTableLoadLatency measures table load operation latency.
// Target: p50 < 50ms, p95 < 100ms, p99 < 200ms.
func BenchmarkTableLoadLatency(b *testing.B) {
	bp := pool.NewBufferPool(nil)

	// Simulate table metadata typical of production workload
	metadata := map[string]any{
		"format-version":        2,
		"table-uuid":            "550e8400-e29b-41d4-a716-446655440000",
		"location":              "s3://warehouse/default/test_table",
		"last-updated-ms":       1703376000000,
		"last-column-id":        20,
		"properties":            map[string]string{"owner": "test", "created-by": "bingsan", "write.format.default": "parquet"},
		"schemas":               []any{map[string]any{"type": "struct", "schema-id": 0, "fields": generateFields(20)}},
		"current-schema-id":     0,
		"partition-specs":       []any{map[string]any{"spec-id": 0, "fields": []any{}}},
		"default-spec-id":       0,
		"sort-orders":           []any{map[string]any{"order-id": 0, "fields": []any{}}},
		"default-sort-order-id": 0,
		"snapshots":             generateSnapshots(5),
		"snapshot-log":          []any{},
		"refs":                  map[string]any{"main": map[string]any{"snapshot-id": 1, "type": "branch"}},
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		buf := bp.Get()
		encoder := json.NewEncoder(buf)
		if err := encoder.Encode(metadata); err != nil {
			b.Fatal(err)
		}
		_ = buf.Bytes()
		bp.Put(buf)
	}
}

// BenchmarkTableLargeSchemaLatency measures operations on tables with 100+ columns.
// Target: < 200ms for large schemas.
func BenchmarkTableLargeSchemaLatency(b *testing.B) {
	bp := pool.NewBufferPool(nil)

	// Large schema with 100+ columns
	metadata := map[string]any{
		"format-version":        2,
		"table-uuid":            "550e8400-e29b-41d4-a716-446655440000",
		"location":              "s3://warehouse/default/large_table",
		"last-updated-ms":       1703376000000,
		"last-column-id":        150,
		"properties":            map[string]string{"owner": "test", "created-by": "bingsan"},
		"schemas":               []any{map[string]any{"type": "struct", "schema-id": 0, "fields": generateFields(150)}},
		"current-schema-id":     0,
		"partition-specs":       []any{map[string]any{"spec-id": 0, "fields": generatePartitionFields(10)}},
		"default-spec-id":       0,
		"sort-orders":           []any{map[string]any{"order-id": 0, "fields": generateSortFields(5)}},
		"default-sort-order-id": 0,
		"snapshots":             generateSnapshots(20),
		"snapshot-log":          generateSnapshotLog(20),
		"refs":                  map[string]any{"main": map[string]any{"snapshot-id": 20, "type": "branch"}},
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		buf := bp.Get()
		encoder := json.NewEncoder(buf)
		if err := encoder.Encode(metadata); err != nil {
			b.Fatal(err)
		}
		_ = buf.Bytes()
		bp.Put(buf)
	}
}

// BenchmarkTableSerializationLatency measures pure serialization latency.
// Target: < 1ms.
func BenchmarkTableSerializationLatency(b *testing.B) {
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

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		buf := bp.Get()
		encoder := json.NewEncoder(buf)
		if err := encoder.Encode(metadata); err != nil {
			b.Fatal(err)
		}
		_ = buf.Bytes()
		bp.Put(buf)
	}
}

// BenchmarkTableCommitLatency measures table commit operation latency.
// Target: < 50ms average.
func BenchmarkTableCommitLatency(b *testing.B) {
	bp := pool.NewBufferPool(nil)

	// Simulate commit request with snapshot changes
	commitRequest := map[string]any{
		"requirements": []any{
			map[string]any{"type": "assert-table-uuid", "uuid": "550e8400-e29b-41d4-a716-446655440000"},
			map[string]any{"type": "assert-current-schema-id", "current-schema-id": 0},
		},
		"updates": []any{
			map[string]any{
				"action":   "add-snapshot",
				"snapshot": generateSnapshot(10),
			},
			map[string]any{
				"action":      "set-snapshot-ref",
				"ref-name":    "main",
				"type":        "branch",
				"snapshot-id": 10,
			},
		},
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		buf := bp.Get()
		encoder := json.NewEncoder(buf)
		if err := encoder.Encode(commitRequest); err != nil {
			b.Fatal(err)
		}
		_ = buf.Bytes()
		bp.Put(buf)
	}
}

// BenchmarkTableDeserialization measures JSON deserialization performance.
func BenchmarkTableDeserialization(b *testing.B) {
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

	// Pre-serialize the data
	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)
	if err := encoder.Encode(metadata); err != nil {
		b.Fatal(err)
	}
	data := buf.Bytes()

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		var result map[string]any
		decoder := json.NewDecoder(bytes.NewReader(data))
		if err := decoder.Decode(&result); err != nil {
			b.Fatal(err)
		}
	}
}

// =============================================================================
// Helpers for US2 benchmarks
// =============================================================================

func generatePartitionFields(count int) []any {
	fields := make([]any, count)
	for i := 0; i < count; i++ {
		fields[i] = map[string]any{
			"source-id": i + 1,
			"field-id":  1000 + i,
			"name":      "partition_" + string(rune('a'+i%26)),
			"transform": "identity",
		}
	}
	return fields
}

func generateSortFields(count int) []any {
	fields := make([]any, count)
	for i := 0; i < count; i++ {
		fields[i] = map[string]any{
			"source-id":  i + 1,
			"transform":  "identity",
			"direction":  "asc",
			"null-order": "nulls-first",
		}
	}
	return fields
}

func generateSnapshot(id int) map[string]any {
	return map[string]any{
		"snapshot-id":        id,
		"parent-snapshot-id": id - 1,
		"sequence-number":    id,
		"timestamp-ms":       1703376000000 + int64(id)*1000,
		"summary": map[string]string{
			"operation":        "append",
			"added-data-files": "10",
			"total-data-files": "100",
			"added-records":    "1000",
			"total-records":    "10000",
		},
		"manifest-list": "s3://warehouse/default/test_table/metadata/snap-" + string(rune('0'+id)) + ".avro",
		"schema-id":     0,
	}
}

func generateSnapshotLog(count int) []any {
	log := make([]any, count)
	for i := 0; i < count; i++ {
		log[i] = map[string]any{
			"snapshot-id":  i + 1,
			"timestamp-ms": 1703376000000 + int64(i)*1000,
		}
	}
	return log
}
