---
title: "Object Pooling"
weight: 1
---

# Object Pooling

Bingsan uses `sync.Pool` from Go's standard library to reduce memory allocation pressure in hot paths. This significantly reduces garbage collection overhead and improves response latency consistency.

## Overview

Two types of pools are implemented:

| Pool | Purpose | Default Size | Max Size |
|------|---------|--------------|----------|
| **BufferPool** | JSON serialization buffers | 4 KB | 64 KB |
| **BytePool** | OAuth token generation | 32 bytes | 32 bytes |

## How It Works

### BufferPool

The `BufferPool` provides reusable `bytes.Buffer` instances for JSON serialization:

```
Request 1 ──► Get buffer ──► Serialize JSON ──► Return buffer ──► Pool
                  │                                  ▲
                  └──────────────────────────────────┘
                              Reused

Request 2 ──► Get buffer (same one!) ──► Serialize JSON ──► Return buffer
```

**Key characteristics:**

- Initial capacity: 4 KB (typical JSON metadata size)
- Maximum size: 64 KB (oversized buffers are discarded)
- Thread-safe via `sync.Pool`
- Automatic reset on get

### BytePool

The `BytePool` provides fixed-size byte slices for token generation:

- Fixed size: 32 bytes (64 hex characters when encoded)
- Used for OAuth access token generation
- Contents not zeroed on reuse (overwritten anyway)

## Usage Patterns

### In API Handlers

Buffers are acquired and released within request handlers:

```go
func (h *Handler) GetTable(ctx *fiber.Ctx) error {
    // Get buffer from pool
    buf := pool.GetBuffer()
    defer pool.PutBuffer(buf)  // Always return!

    // Use buffer for serialization
    encoder := json.NewEncoder(buf)
    if err := encoder.Encode(table); err != nil {
        return err
    }

    return ctx.Send(buf.Bytes())
}
```

### With Metrics

For observability, use the metrics-enabled pool:

```go
metrics := pool.NewPoolMetrics()
bufferPool := pool.NewBufferPool(metrics)

buf := bufferPool.Get()
defer bufferPool.Put(buf)

// Check stats
stats := metrics.Stats()
fmt.Printf("Hit rate: %.2f%%\n", stats.HitRate()*100)
```

## Configuration

Pool behavior is configured via constants (compile-time):

| Constant | Value | Description |
|----------|-------|-------------|
| `DefaultBufferSize` | 4096 | Initial buffer capacity in bytes |
| `MaxBufferSize` | 65536 | Maximum buffer size before discard |
| `TokenSize` | 32 | Fixed size for token byte slices |

### Tuning via Custom Build

To customize pool sizes, modify `internal/pool/buffer.go`:

```go
const (
    DefaultBufferSize = 8192  // Increase for larger schemas
    MaxBufferSize = 131072    // Increase for very large metadata
)
```

## Memory Management

### Buffer Lifecycle

```
┌──────────────────────────────────────────────────────────────────┐
│                        Buffer Lifecycle                           │
└──────────────────────────────────────────────────────────────────┘

1. Get() called
   │
   ▼
┌─────────────────────┐
│ Pool has buffer?    │──No──► Create new buffer (4KB)
└─────────────────────┘              │
   │ Yes                             │
   ▼                                 ▼
┌─────────────────────┐        ┌─────────────────────┐
│ Return from pool    │        │ Grow to initial cap │
└─────────────────────┘        └─────────────────────┘
         │                            │
         └──────────────┬─────────────┘
                        │
                        ▼
              ┌─────────────────────┐
              │ Reset buffer        │
              │ (clear contents)    │
              └─────────────────────┘
                        │
                        ▼
              ┌─────────────────────┐
              │ Use for operation   │
              │ (may grow buffer)   │
              └─────────────────────┘
                        │
                        ▼
              ┌─────────────────────┐
              │ Put() called        │
              └─────────────────────┘
                        │
                        ▼
              ┌─────────────────────┐
              │ Size > 64KB?        │──Yes──► Discard (GC reclaims)
              └─────────────────────┘
                   │ No
                   ▼
              ┌─────────────────────┐
              │ Return to pool      │
              └─────────────────────┘
```

### Why Discard Oversized Buffers

Large buffers are discarded to prevent memory bloat:

1. **Occasional large responses** don't permanently increase pool memory
2. **Memory stays bounded** even with variable workloads
3. **GC can reclaim** memory from large temporary allocations

## Best Practices

### Always Use `defer`

```go
buf := pool.GetBuffer()
defer pool.PutBuffer(buf)  // Guaranteed return
```

### Don't Hold References

```go
// Wrong: Reference escapes
data := buf.Bytes()
pool.PutBuffer(buf)
return data  // data is now invalid!

// Correct: Copy if needed
data := make([]byte, buf.Len())
copy(data, buf.Bytes())
pool.PutBuffer(buf)
return data
```

### Don't Use After Put

```go
pool.PutBuffer(buf)
buf.Write([]byte("data"))  // Wrong: buf may be reused!
```

## Metrics

Pool performance is exposed via Prometheus:

| Metric | Type | Description |
|--------|------|-------------|
| `bingsan_pool_gets_total` | Counter | Total Get() operations |
| `bingsan_pool_returns_total` | Counter | Total Put() operations |
| `bingsan_pool_discards_total` | Counter | Oversized items discarded |
| `bingsan_pool_misses_total` | Counter | New allocations (pool empty) |

### Example Queries

**Pool utilization rate:**
```promql
rate(bingsan_pool_returns_total{pool="buffer"}[5m])
/ rate(bingsan_pool_gets_total{pool="buffer"}[5m])
```

**Discard rate (should be low):**
```promql
rate(bingsan_pool_discards_total{pool="buffer"}[5m])
```

## Benchmarks

Run pool benchmarks:

```bash
go test -bench=BenchmarkPool -benchmem ./tests/benchmark/...
```

Expected results:

| Benchmark | Time | Allocs |
|-----------|------|--------|
| BufferPool.Get/Put | ~50ns | 0 |
| BufferPool.Concurrent | ~100ns | 0 |
| BytePool.Get/Put | ~30ns | 0 |

## Troubleshooting

### High Discard Rate

If `bingsan_pool_discards_total` is increasing rapidly:

1. **Cause**: Many large responses exceeding 64KB
2. **Impact**: Reduced pool effectiveness
3. **Solution**: Consider increasing `MaxBufferSize` for schemas with 100+ columns

### Pool Not Reducing Allocations

If memory allocations aren't decreasing:

1. **Check**: Handlers use `pool.GetBuffer()` / `pool.PutBuffer()`
2. **Check**: All code paths call `Put()` (including error paths)
3. **Check**: `defer` is used to ensure returns

### Memory Growing Over Time

If heap size keeps increasing:

1. **Check**: `MaxBufferSize` isn't too high
2. **Check**: No buffer references escaping
3. **Profile**: Use `go tool pprof` to identify leaks
