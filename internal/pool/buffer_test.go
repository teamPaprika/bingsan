package pool

import (
	"bytes"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewBufferPool(t *testing.T) {
	t.Run("with metrics", func(t *testing.T) {
		metrics := NewPoolMetrics()
		bp := NewBufferPool(metrics)
		require.NotNil(t, bp)
		assert.Same(t, metrics, bp.metrics)
	})

	t.Run("without metrics", func(t *testing.T) {
		bp := NewBufferPool(nil)
		require.NotNil(t, bp)
		assert.Nil(t, bp.metrics)
	})
}

func TestBufferPool_Get(t *testing.T) {
	t.Run("returns valid buffer", func(t *testing.T) {
		metrics := NewPoolMetrics()
		bp := NewBufferPool(metrics)

		buf := bp.Get()
		require.NotNil(t, buf)
		assert.Equal(t, 0, buf.Len(), "buffer should be empty after Get")
	})

	t.Run("buffer is reset", func(t *testing.T) {
		metrics := NewPoolMetrics()
		bp := NewBufferPool(metrics)

		buf := bp.Get()
		buf.WriteString("test data")
		bp.Put(buf)

		buf2 := bp.Get()
		assert.Equal(t, 0, buf2.Len(), "buffer should be reset")
		bp.Put(buf2)
	})

	t.Run("records metrics", func(t *testing.T) {
		metrics := NewPoolMetrics()
		bp := NewBufferPool(metrics)

		bp.Get()
		bp.Get()
		bp.Get()

		stats := metrics.Stats()
		assert.Equal(t, uint64(3), stats.Gets)
	})

	t.Run("without metrics no panic", func(t *testing.T) {
		bp := NewBufferPool(nil)
		buf := bp.Get()
		require.NotNil(t, buf)
	})
}

func TestBufferPool_Put(t *testing.T) {
	t.Run("accepts used buffer", func(t *testing.T) {
		metrics := NewPoolMetrics()
		bp := NewBufferPool(metrics)

		buf := bp.Get()
		buf.WriteString("test data")
		bp.Put(buf)

		stats := metrics.Stats()
		assert.Equal(t, uint64(1), stats.Returns)
	})

	t.Run("nil buffer safe", func(t *testing.T) {
		metrics := NewPoolMetrics()
		bp := NewBufferPool(metrics)

		// Should not panic
		bp.Put(nil)

		stats := metrics.Stats()
		assert.Equal(t, uint64(0), stats.Returns)
	})

	t.Run("oversized buffer discarded", func(t *testing.T) {
		metrics := NewPoolMetrics()
		bp := NewBufferPool(metrics)

		buf := bp.Get()
		oversizedData := make([]byte, MaxBufferSize+1)
		buf.Write(oversizedData)

		bp.Put(buf)

		stats := metrics.Stats()
		assert.Equal(t, uint64(1), stats.Discards)
		assert.Equal(t, uint64(0), stats.Returns)
	})

	t.Run("buffer at max size accepted", func(t *testing.T) {
		metrics := NewPoolMetrics()
		bp := NewBufferPool(metrics)

		buf := bp.Get()
		// Write exactly MaxBufferSize bytes
		buf.Write(make([]byte, MaxBufferSize))

		bp.Put(buf)

		stats := metrics.Stats()
		assert.Equal(t, uint64(0), stats.Discards)
		assert.Equal(t, uint64(1), stats.Returns)
	})

	t.Run("without metrics no panic", func(t *testing.T) {
		bp := NewBufferPool(nil)
		buf := bp.Get()
		buf.WriteString("test")
		bp.Put(buf)
		// No panic expected
	})
}

func TestBufferPool_Concurrent(t *testing.T) {
	metrics := NewPoolMetrics()
	bp := NewBufferPool(metrics)

	const goroutines = 100
	const iterations = 1000

	var wg sync.WaitGroup
	wg.Add(goroutines)

	for g := 0; g < goroutines; g++ {
		go func() {
			defer wg.Done()
			for i := 0; i < iterations; i++ {
				buf := bp.Get()
				buf.WriteString("concurrent test data")
				bp.Put(buf)
			}
		}()
	}

	wg.Wait()

	stats := metrics.Stats()
	expectedGets := uint64(goroutines * iterations)
	assert.Equal(t, expectedGets, stats.Gets)
}

func TestGetBuffer(t *testing.T) {
	t.Run("returns valid buffer", func(t *testing.T) {
		buf := GetBuffer()
		require.NotNil(t, buf)
		assert.Equal(t, 0, buf.Len())
	})

	t.Run("buffer has initial capacity", func(t *testing.T) {
		buf := GetBuffer()
		assert.GreaterOrEqual(t, buf.Cap(), DefaultBufferSize)
		PutBuffer(buf)
	})
}

func TestPutBuffer(t *testing.T) {
	t.Run("accepts valid buffer", func(t *testing.T) {
		buf := GetBuffer()
		buf.WriteString("test")
		PutBuffer(buf)
		// No panic expected
	})

	t.Run("nil safe", func(t *testing.T) {
		PutBuffer(nil)
		// No panic expected
	})

	t.Run("oversized discarded", func(t *testing.T) {
		buf := GetBuffer()
		buf.Write(make([]byte, MaxBufferSize+1))
		PutBuffer(buf)
		// No panic expected, buffer silently discarded
	})
}

func TestBufferPool_ClearsContents(t *testing.T) {
	metrics := NewPoolMetrics()
	bp := NewBufferPool(metrics)

	buf := bp.Get()
	buf.WriteString("sensitive data")

	bp.Put(buf)

	buf2 := bp.Get()
	assert.Equal(t, 0, buf2.Len(), "buffer should be empty")
	assert.NotContains(t, buf2.String(), "sensitive", "previous content should be cleared")
	bp.Put(buf2)
}

func TestBufferConstants(t *testing.T) {
	assert.Equal(t, 4096, DefaultBufferSize)
	assert.Equal(t, 65536, MaxBufferSize)
}

// Benchmarks

func BenchmarkBufferPoolGetPut(b *testing.B) {
	bp := NewBufferPool(nil)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		buf := bp.Get()
		buf.WriteString("test data for benchmark")
		bp.Put(buf)
	}
}

func BenchmarkBufferPoolGetPutParallel(b *testing.B) {
	bp := NewBufferPool(nil)

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

func BenchmarkNoPoolBuffer(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		buf := new(bytes.Buffer)
		buf.Grow(DefaultBufferSize)
		buf.WriteString("test data for benchmark")
	}
}

func BenchmarkBufferPoolWithMetrics(b *testing.B) {
	metrics := NewPoolMetrics()
	bp := NewBufferPool(metrics)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		buf := bp.Get()
		buf.WriteString("test data for benchmark")
		bp.Put(buf)
	}
}
