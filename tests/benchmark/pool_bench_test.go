package benchmark

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"sync"
	"testing"

	"github.com/kimuyb/bingsan/internal/pool"
)

// =============================================================================
// Pool Benchmarks for Performance Optimization (US1)
// =============================================================================
// These benchmarks measure the improvement from sync.Pool implementation.
// Run AFTER implementing the pool package to compare with baseline.
//
// Usage:
//   go test -bench=BenchmarkPool -benchmem ./tests/benchmark/... | tee optimized.txt
//   benchstat baseline_results.txt optimized.txt
// =============================================================================

// BenchmarkPoolTableMetadataSerialization measures JSON serialization with pooled buffers.
func BenchmarkPoolTableMetadataSerialization(b *testing.B) {
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

	bp := pool.NewBufferPool(nil) // No metrics overhead for benchmark

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		buf := bp.Get()
		encoder := json.NewEncoder(buf)
		if err := encoder.Encode(metadata); err != nil {
			b.Fatal(err)
		}
		_ = buf.Bytes() // Simulate reading the result
		bp.Put(buf)
	}
}

// BenchmarkPoolTableMetadataLargeSchema measures large schema serialization with pooled buffers.
func BenchmarkPoolTableMetadataLargeSchema(b *testing.B) {
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

	bp := pool.NewBufferPool(nil)

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

// BenchmarkPoolTokenGeneration measures token generation with pooled byte slices.
func BenchmarkPoolTokenGeneration(b *testing.B) {
	bp := pool.NewBytePool(pool.TokenSize, nil)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		slice := bp.Get()
		if _, err := rand.Read(slice); err != nil {
			b.Fatal(err)
		}
		token := hex.EncodeToString(slice)
		_ = len(token)
		bp.Put(slice)
	}
}

// BenchmarkPoolConcurrentAccess measures pool performance under concurrent load.
func BenchmarkPoolConcurrentAccess(b *testing.B) {
	bp := pool.NewBufferPool(nil)

	b.ReportAllocs()
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			buf := bp.Get()
			buf.WriteString("concurrent access test data")
			bp.Put(buf)
		}
	})
}

// BenchmarkPoolConcurrent10 measures concurrent pool access with 10 goroutines.
func BenchmarkPoolConcurrent10(b *testing.B) {
	bp := pool.NewBufferPool(nil)

	b.ReportAllocs()
	b.ResetTimer()

	b.SetParallelism(10)
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			buf := bp.Get()
			buf.WriteString("concurrent access test data")
			bp.Put(buf)
		}
	})
}

// BenchmarkPoolConcurrent100 measures concurrent pool access with 100 goroutines.
func BenchmarkPoolConcurrent100(b *testing.B) {
	bp := pool.NewBufferPool(nil)

	b.ReportAllocs()
	b.ResetTimer()

	b.SetParallelism(100)
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			buf := bp.Get()
			buf.WriteString("concurrent access test data")
			bp.Put(buf)
		}
	})
}

// BenchmarkPoolUnderSustainedLoad measures pool under sustained load.
func BenchmarkPoolUnderSustainedLoad(b *testing.B) {
	bp := pool.NewBufferPool(nil)
	metrics := pool.NewPoolMetrics()

	// Use metrics version for tracking
	bpWithMetrics := pool.NewBufferPool(metrics)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		buf := bpWithMetrics.Get()
		buf.WriteString("sustained load test")
		bpWithMetrics.Put(buf)
	}

	b.StopTimer()

	// Report hit rate (estimated)
	stats := metrics.Stats()
	if stats.Gets > 0 {
		// For sync.Pool, we can't measure true hits, but we can track discards
		effectiveRate := float64(stats.Returns) / float64(stats.Gets)
		b.ReportMetric(effectiveRate*100, "%_returned")
	}

	// Also benchmark without metrics to see overhead
	b.StartTimer()
	for i := 0; i < 1000; i++ {
		buf := bp.Get()
		buf.WriteString("sustained load test")
		bp.Put(buf)
	}
}

// =============================================================================
// Comparison Benchmarks (baseline without pool)
// =============================================================================

// BenchmarkNoPoolSerializationForComparison provides direct comparison baseline.
func BenchmarkNoPoolSerializationForComparison(b *testing.B) {
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
		buf := new(bytes.Buffer)
		buf.Grow(4096) // Same initial size as pool
		encoder := json.NewEncoder(buf)
		if err := encoder.Encode(metadata); err != nil {
			b.Fatal(err)
		}
		_ = buf.Bytes()
		// Buffer discarded - no pool
	}
}

// BenchmarkNoPoolTokenGenerationForComparison provides direct comparison baseline.
func BenchmarkNoPoolTokenGenerationForComparison(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		slice := make([]byte, 32)
		if _, err := rand.Read(slice); err != nil {
			b.Fatal(err)
		}
		token := hex.EncodeToString(slice)
		_ = len(token)
		// Slice discarded - no pool
	}
}

// =============================================================================
// Memory Stability Benchmarks (US1)
// =============================================================================

// BenchmarkPoolMemoryStability measures memory stability over many iterations.
func BenchmarkPoolMemoryStability(b *testing.B) {
	bp := pool.NewBufferPool(nil)
	bytePool := pool.NewBytePool(pool.TokenSize, nil)

	metadata := map[string]any{
		"format-version":    2,
		"table-uuid":        "550e8400-e29b-41d4-a716-446655440000",
		"location":          "s3://warehouse/default/test_table",
		"last-updated-ms":   1703376000000,
		"properties":        map[string]string{"owner": "test"},
		"schemas":           []any{map[string]any{"type": "struct", "schema-id": 0, "fields": generateFields(20)}},
		"current-schema-id": 0,
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		// Simulate mixed workload
		buf := bp.Get()
		encoder := json.NewEncoder(buf)
		_ = encoder.Encode(metadata) //nolint:errcheck // benchmark
		bp.Put(buf)

		slice := bytePool.Get()
		_, _ = rand.Read(slice) //nolint:errcheck // benchmark
		bytePool.Put(slice)
	}
}

// BenchmarkPoolGCPressure measures GC pressure with pooling.
func BenchmarkPoolGCPressure(b *testing.B) {
	bp := pool.NewBufferPool(nil)

	metadata := map[string]any{
		"format-version":    2,
		"table-uuid":        "550e8400-e29b-41d4-a716-446655440000",
		"location":          "s3://warehouse/default/test_table",
		"last-updated-ms":   1703376000000,
		"properties":        map[string]string{"owner": "test"},
		"schemas":           []any{map[string]any{"type": "struct", "schema-id": 0, "fields": generateFields(20)}},
		"current-schema-id": 0,
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		buf := bp.Get()
		encoder := json.NewEncoder(buf)
		_ = encoder.Encode(metadata) //nolint:errcheck // benchmark
		bp.Put(buf)
	}
}

// =============================================================================
// Concurrent Benchmark Helpers.
// =============================================================================

// BenchmarkPoolVsNoPoolConcurrent compares pooled vs non-pooled under concurrent load.
func BenchmarkPoolVsNoPoolConcurrent(b *testing.B) {
	b.Run("WithPool", func(b *testing.B) {
		bp := pool.NewBufferPool(nil)

		b.ReportAllocs()
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				buf := bp.Get()
				buf.WriteString("test data for concurrent benchmark")
				bp.Put(buf)
			}
		})
	})

	b.Run("WithoutPool", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				buf := new(bytes.Buffer)
				buf.Grow(4096)
				buf.WriteString("test data for concurrent benchmark")
				// Discarded
			}
		})
	})
}

// BenchmarkPoolMultiGoroutine tests scaling with different goroutine counts.
func BenchmarkPoolMultiGoroutine(b *testing.B) {
	goroutineCounts := []int{1, 10, 100, 1000}

	for _, count := range goroutineCounts {
		b.Run(string(rune('0'+count/100))+string(rune('0'+(count/10)%10))+string(rune('0'+count%10))+"_goroutines", func(b *testing.B) {
			bp := pool.NewBufferPool(nil)

			b.ReportAllocs()
			b.ResetTimer()

			var wg sync.WaitGroup
			iterations := b.N / count
			if iterations < 1 {
				iterations = 1
			}

			for g := 0; g < count; g++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					for i := 0; i < iterations; i++ {
						buf := bp.Get()
						buf.WriteString("test")
						bp.Put(buf)
					}
				}()
			}
			wg.Wait()
		})
	}
}
