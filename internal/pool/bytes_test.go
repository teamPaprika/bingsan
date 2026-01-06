package pool

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewBytePool(t *testing.T) {
	t.Run("with standard size", func(t *testing.T) {
		metrics := NewPoolMetrics()
		bp := NewBytePool(TokenSize, metrics)
		require.NotNil(t, bp)
		assert.Equal(t, TokenSize, bp.size)
		assert.Same(t, metrics, bp.metrics)
	})

	t.Run("with custom size", func(t *testing.T) {
		bp := NewBytePool(64, nil)
		require.NotNil(t, bp)
		assert.Equal(t, 64, bp.size)
	})

	t.Run("without metrics", func(t *testing.T) {
		bp := NewBytePool(TokenSize, nil)
		require.NotNil(t, bp)
		assert.Nil(t, bp.metrics)
	})
}

func TestBytePool_Get(t *testing.T) {
	t.Run("returns correct size slice for TokenSize", func(t *testing.T) {
		metrics := NewPoolMetrics()
		bp := NewBytePool(TokenSize, metrics)

		slice := bp.Get()
		require.NotNil(t, slice)
		assert.Len(t, slice, TokenSize)
	})

	t.Run("records metrics for TokenSize", func(t *testing.T) {
		metrics := NewPoolMetrics()
		bp := NewBytePool(TokenSize, metrics)

		bp.Get()
		bp.Get()

		stats := metrics.Stats()
		assert.Equal(t, uint64(2), stats.Gets)
	})

	t.Run("non-standard size allocates new", func(t *testing.T) {
		metrics := NewPoolMetrics()
		bp := NewBytePool(64, metrics)

		slice := bp.Get()
		require.NotNil(t, slice)
		assert.Len(t, slice, 64)

		stats := metrics.Stats()
		assert.Equal(t, uint64(1), stats.Misses)
		assert.Equal(t, uint64(0), stats.Gets)
	})

	t.Run("without metrics no panic", func(t *testing.T) {
		bp := NewBytePool(TokenSize, nil)
		slice := bp.Get()
		require.NotNil(t, slice)
	})
}

func TestBytePool_Put(t *testing.T) {
	t.Run("accepts correct size slice", func(t *testing.T) {
		metrics := NewPoolMetrics()
		bp := NewBytePool(TokenSize, metrics)

		slice := bp.Get()
		for i := range slice {
			slice[i] = byte(i)
		}
		bp.Put(slice)

		stats := metrics.Stats()
		assert.Equal(t, uint64(1), stats.Returns)
	})

	t.Run("nil slice safe", func(t *testing.T) {
		metrics := NewPoolMetrics()
		bp := NewBytePool(TokenSize, metrics)

		bp.Put(nil)

		stats := metrics.Stats()
		assert.Equal(t, uint64(0), stats.Returns)
		assert.Equal(t, uint64(0), stats.Discards)
	})

	t.Run("wrong size slice discarded", func(t *testing.T) {
		metrics := NewPoolMetrics()
		bp := NewBytePool(TokenSize, metrics)

		wrongSize := make([]byte, 16)
		bp.Put(wrongSize)

		stats := metrics.Stats()
		assert.Equal(t, uint64(1), stats.Discards)
		assert.Equal(t, uint64(0), stats.Returns)
	})

	t.Run("without metrics no panic", func(t *testing.T) {
		bp := NewBytePool(TokenSize, nil)
		slice := bp.Get()
		bp.Put(slice)
	})

	t.Run("wrong size without metrics no panic", func(t *testing.T) {
		bp := NewBytePool(TokenSize, nil)
		wrongSize := make([]byte, 16)
		bp.Put(wrongSize)
	})
}

func TestBytePool_Concurrent(t *testing.T) {
	metrics := NewPoolMetrics()
	bp := NewBytePool(TokenSize, metrics)

	const goroutines = 100
	const iterations = 1000

	var wg sync.WaitGroup
	wg.Add(goroutines)

	for g := 0; g < goroutines; g++ {
		go func() {
			defer wg.Done()
			for i := 0; i < iterations; i++ {
				slice := bp.Get()
				for j := range slice {
					slice[j] = byte(j)
				}
				bp.Put(slice)
			}
		}()
	}

	wg.Wait()

	stats := metrics.Stats()
	assert.True(t, stats.Gets > 0)
}

func TestGetBytes(t *testing.T) {
	t.Run("returns valid slice", func(t *testing.T) {
		slice := GetBytes()
		require.NotNil(t, slice)
		assert.Len(t, slice, TokenSize)
	})
}

func TestPutBytes(t *testing.T) {
	t.Run("accepts valid slice", func(t *testing.T) {
		slice := GetBytes()
		for i := range slice {
			slice[i] = byte(i)
		}
		PutBytes(slice)
	})

	t.Run("nil safe", func(t *testing.T) {
		PutBytes(nil)
	})

	t.Run("wrong size silently ignored", func(t *testing.T) {
		wrongSize := make([]byte, 16)
		PutBytes(wrongSize)
	})
}

func TestByteConstants(t *testing.T) {
	assert.Equal(t, 32, TokenSize)
}

// Benchmarks

func BenchmarkBytePoolGetPut(b *testing.B) {
	bp := NewBytePool(TokenSize, nil)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		slice := bp.Get()
		for j := range slice {
			slice[j] = byte(j)
		}
		bp.Put(slice)
	}
}

func BenchmarkBytePoolGetPutParallel(b *testing.B) {
	bp := NewBytePool(TokenSize, nil)

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

func BenchmarkNoPoolBytes(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		slice := make([]byte, TokenSize)
		for j := range slice {
			slice[j] = byte(j)
		}
	}
}

func BenchmarkBytePoolWithMetrics(b *testing.B) {
	metrics := NewPoolMetrics()
	bp := NewBytePool(TokenSize, metrics)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		slice := bp.Get()
		for j := range slice {
			slice[j] = byte(j)
		}
		bp.Put(slice)
	}
}
