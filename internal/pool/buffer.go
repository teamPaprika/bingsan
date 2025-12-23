package pool

import (
	"bytes"
	"sync"
)

const (
	// DefaultBufferSize is the initial capacity for pooled buffers.
	DefaultBufferSize = 4096 // 4KB - typical JSON metadata size

	// MaxBufferSize is the maximum size before buffers are discarded.
	MaxBufferSize = 65536 // 64KB - prevent memory bloat
)

// bufferPool is the global buffer pool for JSON serialization.
var bufferPool = sync.Pool{
	New: func() any {
		buf := new(bytes.Buffer)
		buf.Grow(DefaultBufferSize)
		return buf
	},
}

// BufferPool provides access to a pool of reusable byte buffers.
// Use this for JSON serialization in handlers to reduce allocations.
type BufferPool struct {
	metrics *PoolMetrics
}

// NewBufferPool creates a new BufferPool with optional metrics tracking.
func NewBufferPool(metrics *PoolMetrics) *BufferPool {
	return &BufferPool{metrics: metrics}
}

// Get retrieves a buffer from the pool.
// The buffer is reset and ready for use.
func (p *BufferPool) Get() *bytes.Buffer {
	buf := bufferPool.Get().(*bytes.Buffer)
	buf.Reset()

	if p.metrics != nil {
		// We can't directly detect pool hits vs misses with sync.Pool,
		// but we can track total gets
		p.metrics.RecordGet("buffer")
	}

	return buf
}

// Put returns a buffer to the pool.
// Buffers exceeding MaxBufferSize are discarded to prevent memory bloat.
func (p *BufferPool) Put(buf *bytes.Buffer) {
	if buf == nil {
		return
	}

	// Discard oversized buffers to prevent memory bloat
	if buf.Cap() > MaxBufferSize {
		if p.metrics != nil {
			p.metrics.RecordDiscard("buffer")
		}
		return
	}

	buf.Reset()
	bufferPool.Put(buf)

	if p.metrics != nil {
		p.metrics.RecordReturn("buffer")
	}
}

// GetBuffer is a convenience function that gets a buffer from the default pool.
// For high-performance code, use NewBufferPool with metrics.
func GetBuffer() *bytes.Buffer {
	buf := bufferPool.Get().(*bytes.Buffer)
	buf.Reset()
	return buf
}

// PutBuffer returns a buffer to the default pool.
// For high-performance code, use NewBufferPool with metrics.
func PutBuffer(buf *bytes.Buffer) {
	if buf == nil {
		return
	}
	if buf.Cap() > MaxBufferSize {
		return
	}
	buf.Reset()
	bufferPool.Put(buf)
}
