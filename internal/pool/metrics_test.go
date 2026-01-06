package pool

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewPoolMetrics(t *testing.T) {
	metrics := NewPoolMetrics()
	require.NotNil(t, metrics)

	stats := metrics.Stats()
	assert.Equal(t, uint64(0), stats.Gets)
	assert.Equal(t, uint64(0), stats.Returns)
	assert.Equal(t, uint64(0), stats.Discards)
	assert.Equal(t, uint64(0), stats.Misses)
}

func TestMetrics_RecordGet(t *testing.T) {
	metrics := NewPoolMetrics()

	metrics.RecordGet("test")
	metrics.RecordGet("test")
	metrics.RecordGet("other")

	stats := metrics.Stats()
	assert.Equal(t, uint64(3), stats.Gets)
}

func TestMetrics_RecordReturn(t *testing.T) {
	metrics := NewPoolMetrics()

	metrics.RecordReturn("test")
	metrics.RecordReturn("test")

	stats := metrics.Stats()
	assert.Equal(t, uint64(2), stats.Returns)
}

func TestMetrics_RecordDiscard(t *testing.T) {
	metrics := NewPoolMetrics()

	metrics.RecordDiscard("test")
	metrics.RecordDiscard("test")
	metrics.RecordDiscard("test")

	stats := metrics.Stats()
	assert.Equal(t, uint64(3), stats.Discards)
}

func TestMetrics_RecordMiss(t *testing.T) {
	metrics := NewPoolMetrics()

	metrics.RecordMiss("test")

	stats := metrics.Stats()
	assert.Equal(t, uint64(1), stats.Misses)
}

func TestMetrics_Stats(t *testing.T) {
	metrics := NewPoolMetrics()

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

func TestMetrics_Concurrent(t *testing.T) {
	metrics := NewPoolMetrics()

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

func TestStats_HitRate(t *testing.T) {
	tests := []struct {
		name     string
		stats    Stats
		expected float64
	}{
		{
			name:     "zero gets returns zero",
			stats:    Stats{Gets: 0, Misses: 0},
			expected: 0,
		},
		{
			name:     "all hits",
			stats:    Stats{Gets: 100, Misses: 0},
			expected: 1.0,
		},
		{
			name:     "all misses",
			stats:    Stats{Gets: 100, Misses: 100},
			expected: 0,
		},
		{
			name:     "80 percent hit rate",
			stats:    Stats{Gets: 100, Misses: 20},
			expected: 0.8,
		},
		{
			name:     "50 percent hit rate",
			stats:    Stats{Gets: 100, Misses: 50},
			expected: 0.5,
		},
		{
			name:     "high volume",
			stats:    Stats{Gets: 1000000, Misses: 100000},
			expected: 0.9,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.InDelta(t, tt.expected, tt.stats.HitRate(), 0.001)
		})
	}
}

func TestStats_Fields(t *testing.T) {
	stats := Stats{
		Gets:     10,
		Returns:  8,
		Discards: 1,
		Misses:   2,
	}

	assert.Equal(t, uint64(10), stats.Gets)
	assert.Equal(t, uint64(8), stats.Returns)
	assert.Equal(t, uint64(1), stats.Discards)
	assert.Equal(t, uint64(2), stats.Misses)
}

// Benchmarks

func BenchmarkMetricsRecordGet(b *testing.B) {
	metrics := NewPoolMetrics()

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		metrics.RecordGet("test")
	}
}

func BenchmarkMetricsRecordGetParallel(b *testing.B) {
	metrics := NewPoolMetrics()

	b.ReportAllocs()
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			metrics.RecordGet("test")
		}
	})
}

func BenchmarkMetricsStats(b *testing.B) {
	metrics := NewPoolMetrics()
	for i := 0; i < 1000; i++ {
		metrics.RecordGet("test")
		metrics.RecordReturn("test")
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = metrics.Stats()
	}
}

func BenchmarkStatsHitRate(b *testing.B) {
	stats := Stats{Gets: 1000000, Misses: 100000}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = stats.HitRate()
	}
}
