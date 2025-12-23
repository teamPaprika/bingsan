package benchmark

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"runtime"
	"testing"
)

// =============================================================================
// Baseline Benchmarks for Performance Optimization
// =============================================================================
// These benchmarks establish allocation baselines BEFORE sync.Pool implementation.
// Run and save results before making changes to compare with optimized versions.
//
// Usage:
//   go test -bench=BenchmarkBaseline -benchmem ./tests/benchmark/... | tee baseline.txt
//
// After implementing sync.Pool:
//   go test -bench=BenchmarkPool -benchmem ./tests/benchmark/... | tee optimized.txt
//   benchstat baseline.txt optimized.txt
// =============================================================================

// BenchmarkBaselineTableMetadataSerialization measures JSON serialization allocations
// for table metadata (the primary optimization target).
func BenchmarkBaselineTableMetadataSerialization(b *testing.B) {
	// Sample table metadata similar to what handlers serialize
	metadata := map[string]any{
		"format-version":        2,
		"table-uuid":            "550e8400-e29b-41d4-a716-446655440000",
		"location":              "s3://warehouse/default/test_table",
		"last-updated-ms":       1703376000000,
		"properties":            map[string]string{"owner": "test", "created-by": "bingsan"},
		"schemas":               []any{map[string]any{"type": "struct", "schema-id": 0, "fields": generateFields(20)}},
		"current-schema-id":     0,
		"partition-specs":       []any{map[string]any{"spec-id": 0, "fields": []any{}}},
		"default-spec-id":       0,
		"sort-orders":           []any{map[string]any{"order-id": 0, "fields": []any{}}},
		"default-sort-order-id": 0,
		"snapshots":             []any{},
		"snapshot-log":          []any{},
		"refs":                  map[string]any{},
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		data, err := json.Marshal(metadata)
		if err != nil {
			b.Fatal(err)
		}
		// Simulate writing to response (prevents compiler optimization)
		_ = len(data)
	}
}

// BenchmarkBaselineTableMetadataLargeSchema measures allocations for large schemas (100+ columns).
func BenchmarkBaselineTableMetadataLargeSchema(b *testing.B) {
	metadata := map[string]any{
		"format-version":        2,
		"table-uuid":            "550e8400-e29b-41d4-a716-446655440000",
		"location":              "s3://warehouse/default/large_table",
		"last-updated-ms":       1703376000000,
		"properties":            map[string]string{"owner": "test"},
		"schemas":               []any{map[string]any{"type": "struct", "schema-id": 0, "fields": generateFields(100)}},
		"current-schema-id":     0,
		"partition-specs":       []any{},
		"default-spec-id":       0,
		"sort-orders":           []any{},
		"default-sort-order-id": 0,
		"snapshots":             generateSnapshots(10),
		"snapshot-log":          []any{},
		"refs":                  map[string]any{"main": map[string]any{"snapshot-id": 1}},
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		data, err := json.Marshal(metadata)
		if err != nil {
			b.Fatal(err)
		}
		_ = len(data)
	}
}

// BenchmarkBaselineTokenGeneration measures byte slice allocation for token generation.
func BenchmarkBaselineTokenGeneration(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		// This is what oauth.go does - allocates 32 bytes each time
		bytes := make([]byte, 32)
		if _, err := rand.Read(bytes); err != nil {
			b.Fatal(err)
		}
		token := hex.EncodeToString(bytes)
		_ = len(token)
	}
}

// BenchmarkBaselineGCPressureSerialization measures GC pressure from repeated serialization.
func BenchmarkBaselineGCPressureSerialization(b *testing.B) {
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
		"snapshots":             []any{},
		"snapshot-log":          []any{},
		"refs":                  map[string]any{},
	}

	// Force GC and get baseline
	runtime.GC()
	var m1 runtime.MemStats
	runtime.ReadMemStats(&m1)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		data, err := json.Marshal(metadata)
		if err != nil {
			b.Fatal(err)
		}
		_ = len(data)
	}

	b.StopTimer()

	// Get final GC stats
	var m2 runtime.MemStats
	runtime.ReadMemStats(&m2)

	// Report GC metrics
	gcRuns := m2.NumGC - m1.NumGC
	b.ReportMetric(float64(gcRuns), "gc_runs_total")
	if b.N > 0 {
		b.ReportMetric(float64(gcRuns)/float64(b.N)*1000, "gc/1000ops")
	}
	pauseNs := m2.PauseTotalNs - m1.PauseTotalNs
	b.ReportMetric(float64(pauseNs)/1e6, "gc_pause_ms_total")
}

// BenchmarkBaselineMemoryStabilitySerialization measures heap growth from repeated serialization.
func BenchmarkBaselineMemoryStabilitySerialization(b *testing.B) {
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
		"snapshots":             []any{},
		"snapshot-log":          []any{},
		"refs":                  map[string]any{},
	}

	// Force GC and get baseline
	runtime.GC()
	var m1 runtime.MemStats
	runtime.ReadMemStats(&m1)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		data, err := json.Marshal(metadata)
		if err != nil {
			b.Fatal(err)
		}
		_ = len(data)
	}

	b.StopTimer()

	// Get final memory stats
	runtime.GC()
	var m2 runtime.MemStats
	runtime.ReadMemStats(&m2)

	// Report heap growth
	heapGrowth := int64(m2.HeapAlloc) - int64(m1.HeapAlloc)
	b.ReportMetric(float64(heapGrowth), "heap_growth_bytes")
	b.ReportMetric(float64(m2.HeapInuse), "heap_inuse_bytes")
}

// =============================================================================
// Helper functions
// =============================================================================

func generateFields(count int) []any {
	fields := make([]any, count)
	for i := 0; i < count; i++ {
		fields[i] = map[string]any{
			"id":       i + 1,
			"name":     "field_" + string(rune('a'+i%26)),
			"required": i%2 == 0,
			"type":     "string",
		}
	}
	return fields
}

func generateSnapshots(count int) []any {
	snapshots := make([]any, count)
	for i := 0; i < count; i++ {
		snapshots[i] = map[string]any{
			"snapshot-id":     int64(i + 1),
			"timestamp-ms":    1703376000000 + int64(i*1000),
			"summary":         map[string]any{"operation": "append"},
			"manifest-list":   "s3://warehouse/default/test/metadata/snap-" + string(rune('0'+i)),
			"sequence-number": int64(i + 1),
			"schema-id":       0,
		}
	}
	return snapshots
}
