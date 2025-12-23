package unit

import (
	"bytes"
	"sync"
	"testing"

	"github.com/kimuyb/bingsan/internal/pool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// BufferPool Tests
// =============================================================================

func TestBufferPoolGetPut(t *testing.T) {
	metrics := pool.NewPoolMetrics()
	bp := pool.NewBufferPool(metrics)

	// Get returns valid buffer
	buf := bp.Get()
	require.NotNil(t, buf)
	assert.Equal(t, 0, buf.Len(), "buffer should be empty after Get")

	// Write some data
	buf.WriteString("test data")
	assert.Equal(t, 9, buf.Len())

	// Put accepts used buffer
	bp.Put(buf)

	// Get again - should be reset
	buf2 := bp.Get()
	require.NotNil(t, buf2)
	assert.Equal(t, 0, buf2.Len(), "buffer should be reset after Get")
	bp.Put(buf2)
}

func TestBufferPoolResetClearsContents(t *testing.T) {
	metrics := pool.NewPoolMetrics()
	bp := pool.NewBufferPool(metrics)

	// Get and write data
	buf := bp.Get()
	buf.WriteString("sensitive data")

	// Return to pool
	bp.Put(buf)

	// Get again - content should be cleared
	buf2 := bp.Get()
	assert.Equal(t, 0, buf2.Len(), "buffer should be empty")
	assert.NotContains(t, buf2.String(), "sensitive", "previous content should be cleared")
	bp.Put(buf2)
}

func TestBufferPoolOversizedDiscarded(t *testing.T) {
	metrics := pool.NewPoolMetrics()
	bp := pool.NewBufferPool(metrics)

	// Create an oversized buffer
	buf := bp.Get()
	oversizedData := make([]byte, pool.MaxBufferSize+1)
	buf.Write(oversizedData)
	assert.True(t, buf.Cap() > pool.MaxBufferSize, "buffer should be oversized")

	// Put should discard it (not crash)
	bp.Put(buf)

	// Check metrics show discard
	stats := metrics.Stats()
	assert.Equal(t, uint64(1), stats.Discards, "oversized buffer should be discarded")
}

func TestBufferPoolNilSafe(t *testing.T) {
	metrics := pool.NewPoolMetrics()
	bp := pool.NewBufferPool(metrics)

	// Put nil should not panic
	bp.Put(nil)
}

func TestBufferPoolConcurrent(t *testing.T) {
	metrics := pool.NewPoolMetrics()
	bp := pool.NewBufferPool(metrics)

	const goroutines = 100
	const iterations = 1000

	var wg sync.WaitGroup
	wg.Add(goroutines)

	for g := 0; g < goroutines; g++ {
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

	// Verify metrics
	stats := metrics.Stats()
	expectedGets := uint64(goroutines * iterations)
	assert.Equal(t, expectedGets, stats.Gets, "all gets should be recorded")
}

// =============================================================================
// BytePool Tests
// =============================================================================

func TestBytePoolGetPut(t *testing.T) {
	metrics := pool.NewPoolMetrics()
	bp := pool.NewBytePool(pool.TokenSize, metrics)

	// Get returns correct size slice
	slice := bp.Get()
	require.NotNil(t, slice)
	assert.Len(t, slice, pool.TokenSize, "slice should be TokenSize bytes")

	// Put accepts used slice
	bp.Put(slice)
}

func TestBytePoolSliceContentsOverwritten(t *testing.T) {
	metrics := pool.NewPoolMetrics()
	bp := pool.NewBytePool(pool.TokenSize, metrics)

	// Get and fill with pattern
	slice := bp.Get()
	for i := range slice {
		slice[i] = 0xFF
	}

	// Return and get again
	bp.Put(slice)
	slice2 := bp.Get()

	// Content should be overwritten on use (not guaranteed to be same slice)
	// Just verify we get a valid slice
	assert.Len(t, slice2, pool.TokenSize)
	bp.Put(slice2)
}

func TestBytePoolWrongSizeDiscarded(t *testing.T) {
	metrics := pool.NewPoolMetrics()
	bp := pool.NewBytePool(pool.TokenSize, metrics)

	// Try to put wrong size slice
	wrongSize := make([]byte, 16)
	bp.Put(wrongSize)

	// Check metrics show discard
	stats := metrics.Stats()
	assert.Equal(t, uint64(1), stats.Discards, "wrong size slice should be discarded")
}

func TestBytePoolNilSafe(t *testing.T) {
	metrics := pool.NewPoolMetrics()
	bp := pool.NewBytePool(pool.TokenSize, metrics)

	// Put nil should not panic
	bp.Put(nil)
}

func TestBytePoolConcurrent(t *testing.T) {
	metrics := pool.NewPoolMetrics()
	bp := pool.NewBytePool(pool.TokenSize, metrics)

	const goroutines = 100
	const iterations = 1000

	var wg sync.WaitGroup
	wg.Add(goroutines)

	for g := 0; g < goroutines; g++ {
		go func() {
			defer wg.Done()
			for i := 0; i < iterations; i++ {
				slice := bp.Get()
				// Simulate token generation
				for j := range slice {
					slice[j] = byte(j)
				}
				bp.Put(slice)
			}
		}()
	}

	wg.Wait()

	// Verify no panics or race conditions occurred
	stats := metrics.Stats()
	assert.True(t, stats.Gets > 0, "gets should be recorded")
}

// =============================================================================
// PoolMetrics Tests
// =============================================================================

func TestPoolMetricsRecording(t *testing.T) {
	metrics := pool.NewPoolMetrics()

	// Record various operations
	metrics.RecordGet("test")
	metrics.RecordGet("test")
	metrics.RecordReturn("test")
	metrics.RecordDiscard("test")
	metrics.RecordMiss("test")

	stats := metrics.Stats()
	assert.Equal(t, uint64(2), stats.Gets)
	assert.Equal(t, uint64(1), stats.Returns)
	assert.Equal(t, uint64(1), stats.Discards)
	assert.Equal(t, uint64(1), stats.Misses)
}

func TestPoolStatsHitRate(t *testing.T) {
	tests := []struct {
		name     string
		gets     uint64
		misses   uint64
		expected float64
	}{
		{"zero gets", 0, 0, 0},
		{"all hits", 100, 0, 1.0},
		{"all misses", 100, 100, 0},
		{"80% hit rate", 100, 20, 0.8},
		{"50% hit rate", 100, 50, 0.5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stats := pool.PoolStats{
				Gets:   tt.gets,
				Misses: tt.misses,
			}
			assert.InDelta(t, tt.expected, stats.HitRate(), 0.001)
		})
	}
}

func TestPoolMetricsConcurrent(t *testing.T) {
	metrics := pool.NewPoolMetrics()

	const goroutines = 100
	const iterations = 1000

	var wg sync.WaitGroup
	wg.Add(goroutines)

	for g := 0; g < goroutines; g++ {
		go func() {
			defer wg.Done()
			for i := 0; i < iterations; i++ {
				metrics.RecordGet("test")
				metrics.RecordReturn("test")
			}
		}()
	}

	wg.Wait()

	stats := metrics.Stats()
	expected := uint64(goroutines * iterations)
	assert.Equal(t, expected, stats.Gets)
	assert.Equal(t, expected, stats.Returns)
}

// =============================================================================
// Convenience Function Tests
// =============================================================================

func TestGetPutBufferConvenience(t *testing.T) {
	buf := pool.GetBuffer()
	require.NotNil(t, buf)
	assert.Equal(t, 0, buf.Len())

	buf.WriteString("test")
	pool.PutBuffer(buf)

	// Should not panic
	pool.PutBuffer(nil)
}

func TestGetPutBytesConvenience(t *testing.T) {
	slice := pool.GetBytes()
	require.NotNil(t, slice)
	assert.Len(t, slice, pool.TokenSize)

	pool.PutBytes(slice)

	// Should not panic
	pool.PutBytes(nil)

	// Wrong size should not be returned to pool
	wrongSize := make([]byte, 16)
	pool.PutBytes(wrongSize)
}

// =============================================================================
// Benchmark Integration Tests (without metrics overhead)
// =============================================================================

func BenchmarkBufferPoolGetPut(b *testing.B) {
	bp := pool.NewBufferPool(nil) // No metrics for benchmark

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		buf := bp.Get()
		buf.WriteString("test data for benchmark")
		bp.Put(buf)
	}
}

func BenchmarkBytePoolGetPut(b *testing.B) {
	bp := pool.NewBytePool(pool.TokenSize, nil) // No metrics for benchmark

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		slice := bp.Get()
		// Simulate token generation
		for j := range slice {
			slice[j] = byte(j)
		}
		bp.Put(slice)
	}
}

func BenchmarkBufferPoolGetPutParallel(b *testing.B) {
	bp := pool.NewBufferPool(nil)

	b.ReportAllocs()
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			buf := bp.Get()
			buf.WriteString("test data for benchmark")
			bp.Put(buf)
		}
	})
}

func BenchmarkBytePoolGetPutParallel(b *testing.B) {
	bp := pool.NewBytePool(pool.TokenSize, nil)

	b.ReportAllocs()
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			slice := bp.Get()
			for j := range slice {
				slice[j] = byte(j)
			}
			bp.Put(slice)
		}
	})
}

// BenchmarkNoPool shows allocation without pooling for comparison.
func BenchmarkNoPoolBuffer(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		buf := new(bytes.Buffer)
		buf.Grow(4096)
		buf.WriteString("test data for benchmark")
		// No pool - just discard
	}
}

// BenchmarkNoPoolBytes shows allocation without pooling for comparison.
func BenchmarkNoPoolBytes(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		slice := make([]byte, 32)
		for j := range slice {
			slice[j] = byte(j)
		}
		// No pool - just discard
	}
}
