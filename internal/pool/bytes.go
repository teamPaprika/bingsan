package pool

import (
	"sync"
)

const (
	// TokenSize is the fixed size for token byte slices.
	TokenSize = 32 // 32 bytes = 64 hex characters
)

// bytePool is the global pool for fixed-size byte slices.
var bytePool = sync.Pool{
	New: func() any {
		return make([]byte, TokenSize)
	},
}

// BytePool provides access to a pool of reusable byte slices.
// Use this for token generation to reduce allocations.
type BytePool struct {
	size    int
	metrics *PoolMetrics
}

// NewBytePool creates a new BytePool for slices of the specified size.
func NewBytePool(size int, metrics *PoolMetrics) *BytePool {
	return &BytePool{
		size:    size,
		metrics: metrics,
	}
}

// Get retrieves a byte slice from the pool.
// The slice content is NOT zeroed - it will be overwritten anyway.
func (p *BytePool) Get() []byte {
	if p.size != TokenSize {
		// For non-standard sizes, allocate new
		if p.metrics != nil {
			p.metrics.RecordMiss("bytes")
		}
		return make([]byte, p.size)
	}

	slice := bytePool.Get().([]byte)

	if p.metrics != nil {
		p.metrics.RecordGet("bytes")
	}

	return slice
}

// Put returns a byte slice to the pool.
// Only slices of the expected size are returned to the pool.
func (p *BytePool) Put(slice []byte) {
	if slice == nil || len(slice) != TokenSize {
		if p.metrics != nil && slice != nil {
			p.metrics.RecordDiscard("bytes")
		}
		return
	}

	bytePool.Put(slice)

	if p.metrics != nil {
		p.metrics.RecordReturn("bytes")
	}
}

// GetBytes is a convenience function that gets a token-sized byte slice.
// For high-performance code, use NewBytePool with metrics.
func GetBytes() []byte {
	return bytePool.Get().([]byte)
}

// PutBytes returns a byte slice to the default pool.
// For high-performance code, use NewBytePool with metrics.
func PutBytes(slice []byte) {
	if slice == nil || len(slice) != TokenSize {
		return
	}
	bytePool.Put(slice)
}
