# Research: sync.Pool Best Practices for Performance Optimization

**Feature**: 002-perf-optimization
**Date**: 2025-12-24
**Purpose**: Resolve technical unknowns and establish best practices for sync.Pool implementation

## Research Topics

### 1. sync.Pool Behavior and Lifecycle

**Decision**: Use `sync.Pool` with `New` function for automatic allocation on miss

**Rationale**:
- `sync.Pool` automatically clears objects during GC (not leaked, but recycled)
- Objects may be removed between GC cycles - pool is a cache, not a guarantee
- `New` function provides fallback allocation when pool is empty
- Thread-safe by design, no additional locking needed

**Alternatives Considered**:
- Custom ring buffer: Rejected - more complex, sync.Pool is battle-tested
- Channel-based pool: Rejected - higher overhead, less flexible sizing
- Pre-allocated fixed array: Rejected - doesn't scale with load

**Key Pattern**:
```go
var bufferPool = sync.Pool{
    New: func() interface{} {
        return new(bytes.Buffer)
    },
}

func GetBuffer() *bytes.Buffer {
    return bufferPool.Get().(*bytes.Buffer)
}

func PutBuffer(buf *bytes.Buffer) {
    buf.Reset()  // CRITICAL: Reset before returning
    bufferPool.Put(buf)
}
```

---

### 2. Buffer Sizing Strategy

**Decision**: Use 4KB initial buffer size, allow growth, reset on return

**Rationale**:
- Average Iceberg table metadata JSON is 1-3KB
- Large schemas (100+ columns) can reach 10-20KB
- Go slices grow efficiently (doubling strategy)
- Reset() is O(1), doesn't deallocate backing array

**Alternatives Considered**:
- Fixed 1KB buffers: Rejected - frequent reallocation for larger metadata
- Fixed 64KB buffers: Rejected - wasteful for typical small responses
- Size-bucketed pools: Considered but deferred - adds complexity, measure first

**Measurements from Codebase**:
- `json.Marshal(metadata)` in table.go called ~5 times per table operation
- Typical metadata size: 2-5KB JSON
- Large table schemas: up to 50KB JSON

---

### 3. When NOT to Use sync.Pool

**Decision**: Only pool high-frequency, short-lived allocations

**Do Pool**:
- JSON serialization buffers (per-request, discarded immediately)
- Random byte slices for token generation (short-lived)
- Temporary structs for request/response building

**Don't Pool**:
- Database connection objects (already pooled by pgx)
- Fiber context (already pooled by framework)
- Long-lived objects (defeats purpose)
- Objects with cleanup requirements (finalizers)

**Rationale**: Pooling low-frequency allocations adds overhead without benefit. GC handles small, infrequent allocations efficiently.

---

### 4. Pool Metrics and Observability

**Decision**: Expose hit/miss counters via Prometheus

**Metrics to Track**:
```text
bingsan_pool_hits_total{pool="buffer"}    - Pool hit count
bingsan_pool_misses_total{pool="buffer"}  - Pool miss (new allocation)
bingsan_pool_returns_total{pool="buffer"} - Items returned to pool
```

**Rationale**:
- Hit rate indicates pool effectiveness
- High miss rate suggests pool size insufficient or GC too aggressive
- Return count helps detect leaks (hits + misses should ≈ returns)

**Alternatives Considered**:
- Tracking pool size: Not possible - sync.Pool doesn't expose size
- Per-item timing: Rejected - too much overhead for the benefit

---

### 5. Testing Strategy

**Decision**: Benchmark-driven development with allocation tracking

**Test Types**:
1. **Baseline benchmarks**: Measure current allocations before changes
2. **Pool benchmarks**: Measure allocations with pooling
3. **Concurrent benchmarks**: Verify thread-safety under load
4. **Memory stability**: Long-running test to detect leaks

**Benchmark Pattern**:
```go
func BenchmarkTableLoad(b *testing.B) {
    b.ReportAllocs()  // Track allocations
    for i := 0; i < b.N; i++ {
        // ... operation ...
    }
}
```

**Success Criteria**:
- `allocs/op` reduced by ≥30%
- `ns/op` unchanged or improved
- No goroutine leaks after 1M iterations

---

### 6. Integration with goccy/go-json

**Decision**: Pool `bytes.Buffer` used with json encoder, not json output directly

**Rationale**:
- goccy/go-json internally optimizes encoding
- We can pool the buffer passed to `json.Marshal` for reuse
- Pooling the JSON output bytes directly risks data corruption if reused before copy

**Pattern**:
```go
buf := bufferPool.Get().(*bytes.Buffer)
defer func() {
    buf.Reset()
    bufferPool.Put(buf)
}()

encoder := json.NewEncoder(buf)
encoder.Encode(metadata)
// Copy buf.Bytes() to response
```

---

## Summary of Decisions

| Topic | Decision | Confidence |
|-------|----------|------------|
| Pool implementation | `sync.Pool` with `New` function | High |
| Buffer size | 4KB initial, allow growth | Medium |
| What to pool | JSON buffers, byte slices | High |
| Metrics | Prometheus counters (hit/miss/return) | High |
| Testing | Benchmark-driven with `-benchmem` | High |
| JSON integration | Pool buffers, not JSON output | High |

## Open Questions (for implementation phase)

1. Should we implement size-bucketed pools if large variance in buffer sizes observed?
2. Should pool metrics be opt-in (flag) or always-on?
3. Need to measure actual hit rate in production before tuning

## References

- [Go sync.Pool documentation](https://pkg.go.dev/sync#Pool)
- [Effective Go - Concurrency](https://go.dev/doc/effective_go#concurrency)
- [High Performance Go - sync.Pool](https://dave.cheney.net/2016/01/18/cgo-is-not-go)
- [goccy/go-json performance](https://github.com/goccy/go-json#benchmarks)
