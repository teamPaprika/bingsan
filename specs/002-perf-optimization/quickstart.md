# Quickstart: Verifying Performance Optimization

**Feature**: 002-perf-optimization
**Purpose**: How to verify sync.Pool implementation is working correctly

## Prerequisites

- Go 1.21+ installed
- Bingsan repository cloned
- PostgreSQL running (for full integration tests)

## Quick Verification

### 1. Run Baseline Benchmarks (Before Implementation)

```bash
# Record current allocation baseline
go test -bench=BenchmarkTable -benchmem ./tests/benchmark/... | tee baseline.txt

# Example output:
# BenchmarkTableLoad-12    50000    25000 ns/op    4096 B/op    15 allocs/op
```

### 2. Run Optimized Benchmarks (After Implementation)

```bash
# Run pool-enabled benchmarks
go test -bench=BenchmarkPool -benchmem ./tests/benchmark/... | tee optimized.txt

# Compare results
benchstat baseline.txt optimized.txt
```

### 3. Verify Pool Hit Rate

```bash
# Start server and generate load
./bin/iceberg-catalog &
hey -n 10000 -c 100 http://localhost:8080/v1/namespaces/test/tables/test

# Check pool metrics
curl -s http://localhost:8080/metrics | grep bingsan_pool

# Expected output:
# bingsan_pool_hits_total{pool="buffer"} 8500
# bingsan_pool_misses_total{pool="buffer"} 1500
# bingsan_pool_returns_total{pool="buffer"} 10000
```

### 4. Verify No Memory Leaks

```bash
# Run memory stability benchmark
go test -bench=BenchmarkPoolMemoryStability -benchtime=30s ./tests/benchmark/...

# Check for stable heap usage
go test -bench=. -memprofile=mem.prof ./tests/benchmark/...
go tool pprof -text mem.prof | head -20
```

## Success Criteria Checklist

| Metric | Target | How to Measure |
|--------|--------|----------------|
| Allocation reduction | ≥ 30% | `benchstat` comparison |
| No latency regression | p50 unchanged | `ns/op` in benchmark |
| Pool hit rate | ≥ 80% | Prometheus metrics |
| GC pause p99 | < 10ms | `runtime.ReadMemStats` |
| Memory stability | No growth | 24h load test |

## Troubleshooting

### Low Hit Rate (< 80%)

- Check if GC is running too frequently
- Consider increasing `GOGC` environment variable
- Verify buffers are being returned (not leaked)

### Higher Latency After Change

- Check if buffer size is too small (causing reallocation)
- Verify `Reset()` is O(1) not O(n)
- Profile with `pprof` to find bottleneck

### Memory Growth Over Time

- Check for leaked buffers (Get without Put)
- Verify defer pattern is used correctly
- Look for error paths that skip Put()

## Example Integration Test

```go
func TestPoolIntegration(t *testing.T) {
    // Setup server with pool enabled
    cfg := config.Default()
    server := api.NewServer(cfg, nil)

    // Generate load
    for i := 0; i < 10000; i++ {
        req := httptest.NewRequest("GET", "/v1/config", nil)
        resp, _ := server.App().Test(req, -1)
        resp.Body.Close()
    }

    // Check metrics
    metrics := getPoolMetrics()
    hitRate := float64(metrics.Hits) / float64(metrics.Hits + metrics.Misses)

    assert.GreaterOrEqual(t, hitRate, 0.80, "Pool hit rate should be >= 80%")
}
```

## Monitoring in Production

### Grafana Dashboard Queries

```promql
# Pool hit rate
sum(rate(bingsan_pool_hits_total[5m])) /
(sum(rate(bingsan_pool_hits_total[5m])) + sum(rate(bingsan_pool_misses_total[5m])))

# Allocation rate (should decrease after optimization)
rate(go_memstats_alloc_bytes_total[5m])

# GC pause duration
go_gc_duration_seconds{quantile="0.99"}
```

### Alerting Rules

```yaml
# Alert if pool hit rate drops
- alert: PoolHitRateLow
  expr: |
    sum(rate(bingsan_pool_hits_total[5m])) /
    (sum(rate(bingsan_pool_hits_total[5m])) + sum(rate(bingsan_pool_misses_total[5m]))) < 0.7
  for: 5m
  annotations:
    summary: "Pool hit rate below 70%"
```
