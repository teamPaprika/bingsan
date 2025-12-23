// Package pool provides object pooling for performance optimization.
//
// The package implements sync.Pool-based pools for:
//   - BufferPool: Reusable byte buffers for JSON serialization
//   - BytePool: Reusable byte slices for token generation
//
// Pool metrics are exposed via Prometheus for monitoring pool effectiveness.
package pool
