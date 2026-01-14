package integration

import (
	"bytes"
	"encoding/json"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/teamPaprika/bingsan/internal/pool"
)

// TestPoolIntegration tests the pool package under realistic conditions.
func TestPoolIntegration(t *testing.T) {
	t.Run("buffer pool reuse", func(t *testing.T) {
		bp := pool.NewBufferPool(nil)

		// Get and use a buffer
		buf1 := bp.Get()
		buf1.WriteString("test data")
		require.Equal(t, "test data", buf1.String())

		// Return buffer
		bp.Put(buf1)

		// Get another buffer - might be the same one
		buf2 := bp.Get()
		assert.Empty(t, buf2.String(), "buffer should be reset")

		// Clean up
		bp.Put(buf2)
	})

	t.Run("concurrent buffer operations", func(t *testing.T) {
		bp := pool.NewBufferPool(nil)
		var wg sync.WaitGroup

		// Simulate 100 concurrent serialization operations
		for i := 0; i < 100; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()

				for j := 0; j < 100; j++ {
					buf := bp.Get()

					// Simulate JSON serialization
					data := map[string]any{
						"id":   id,
						"iter": j,
						"data": "test payload for concurrent operations",
					}

					encoder := json.NewEncoder(buf)
					err := encoder.Encode(data)
					assert.NoError(t, err)
					assert.NotEmpty(t, buf.Bytes())

					bp.Put(buf)
				}
			}(i)
		}

		wg.Wait()
	})

	t.Run("byte pool for tokens", func(t *testing.T) {
		bytePool := pool.NewBytePool(pool.TokenSize, nil)

		// Get and use byte slice
		slice := bytePool.Get()
		require.Len(t, slice, pool.TokenSize)

		// Fill with data
		for i := range slice {
			slice[i] = byte(i)
		}

		// Return and get again
		bytePool.Put(slice)
		slice2 := bytePool.Get()
		require.Len(t, slice2, pool.TokenSize)

		// Clean up
		bytePool.Put(slice2)
	})

	t.Run("metrics tracking", func(t *testing.T) {
		metrics := pool.NewPoolMetrics()
		bp := pool.NewBufferPool(metrics)

		// Perform operations
		for i := 0; i < 10; i++ {
			buf := bp.Get()
			buf.WriteString("test")
			bp.Put(buf)
		}

		stats := metrics.Stats()
		assert.Equal(t, uint64(10), stats.Gets)
		assert.Equal(t, uint64(10), stats.Returns)
		assert.Equal(t, uint64(0), stats.Discards)
	})

	t.Run("oversized buffer discard", func(t *testing.T) {
		metrics := pool.NewPoolMetrics()
		bp := pool.NewBufferPool(metrics)

		// Get a buffer and grow it beyond max size
		buf := bp.Get()

		// Write enough data to exceed MaxBufferSize (64KB)
		largeData := bytes.Repeat([]byte("x"), pool.MaxBufferSize+1)
		buf.Write(largeData)

		// Return - should be discarded
		bp.Put(buf)

		stats := metrics.Stats()
		assert.Equal(t, uint64(1), stats.Discards)
	})

	t.Run("realistic workload simulation", func(t *testing.T) {
		if testing.Short() {
			t.Skip("skipping realistic workload test in short mode")
		}

		bp := pool.NewBufferPool(nil)
		bytePool := pool.NewBytePool(pool.TokenSize, nil)

		// Simulate typical API handler workload
		metadata := map[string]any{
			"format-version":        2,
			"table-uuid":            "550e8400-e29b-41d4-a716-446655440000",
			"location":              "s3://warehouse/default/test_table",
			"last-updated-ms":       time.Now().UnixMilli(),
			"properties":            map[string]string{"owner": "test"},
			"schemas":               []any{map[string]any{"type": "struct", "schema-id": 0}},
			"current-schema-id":     0,
			"partition-specs":       []any{},
			"default-spec-id":       0,
			"sort-orders":           []any{},
			"default-sort-order-id": 0,
		}

		var wg sync.WaitGroup
		requestCount := 1000
		workerCount := 10

		for w := 0; w < workerCount; w++ {
			wg.Add(1)
			go func() {
				defer wg.Done()

				for i := 0; i < requestCount/workerCount; i++ {
					// Simulate table metadata serialization
					buf := bp.Get()
					encoder := json.NewEncoder(buf)
					_ = encoder.Encode(metadata) //nolint:errcheck // test code
					_ = buf.Bytes()              // simulate sending response
					bp.Put(buf)

					// Simulate token generation (occasionally)
					if i%10 == 0 {
						slice := bytePool.Get()
						_ = slice // simulate using the token
						bytePool.Put(slice)
					}
				}
			}()
		}

		wg.Wait()
	})
}

// TestPoolMemoryEfficiency verifies memory efficiency under load.
func TestPoolMemoryEfficiency(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping memory efficiency test in short mode")
	}

	bp := pool.NewBufferPool(nil)

	metadata := map[string]any{
		"format-version":  2,
		"table-uuid":      "550e8400-e29b-41d4-a716-446655440000",
		"location":        "s3://warehouse/default/test_table",
		"last-updated-ms": time.Now().UnixMilli(),
		"properties":      map[string]string{"owner": "test"},
		"schemas":         []any{map[string]any{"type": "struct", "schema-id": 0}},
	}

	// Record initial memory
	var m1 runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&m1)

	// Perform many serializations
	for i := 0; i < 10000; i++ {
		buf := bp.Get()
		encoder := json.NewEncoder(buf)
		_ = encoder.Encode(metadata) //nolint:errcheck // test code
		_ = buf.Bytes()
		bp.Put(buf)
	}

	// Record final memory
	var m2 runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&m2)

	// Memory growth should be minimal due to pooling
	heapGrowth := int64(m2.HeapAlloc) - int64(m1.HeapAlloc)
	t.Logf("Heap growth after 10000 operations: %d bytes", heapGrowth)

	// Allow some growth but not linear with operation count
	// Without pooling, each operation would allocate ~10KB
	// With pooling, total growth should be much less
	assert.Less(t, heapGrowth, int64(1024*1024), "heap growth should be under 1MB")
}

// TestPoolConcurrencySafety verifies thread safety under high contention.
func TestPoolConcurrencySafety(t *testing.T) {
	bp := pool.NewBufferPool(nil)
	bytePool := pool.NewBytePool(pool.TokenSize, nil)

	var wg sync.WaitGroup
	goroutines := 100
	iterations := 1000

	// Track any panics or data races
	panicCount := int64(0)

	for g := 0; g < goroutines; g++ {
		wg.Add(2)

		// Buffer pool goroutine
		go func() {
			defer wg.Done()
			defer func() {
				if r := recover(); r != nil {
					panicCount++
				}
			}()

			for i := 0; i < iterations; i++ {
				buf := bp.Get()
				buf.WriteString("concurrent test data")
				bp.Put(buf)
			}
		}()

		// Byte pool goroutine
		go func() {
			defer wg.Done()
			defer func() {
				if r := recover(); r != nil {
					panicCount++
				}
			}()

			for i := 0; i < iterations; i++ {
				slice := bytePool.Get()
				for j := range slice {
					slice[j] = byte(j)
				}
				bytePool.Put(slice)
			}
		}()
	}

	wg.Wait()
	assert.Equal(t, int64(0), panicCount, "no panics should occur")
}
