# Performance Optimization Guide

This document describes the performance optimizations implemented in Bingsan and how to monitor and tune them.

## Overview

Bingsan uses `sync.Pool` from Go's standard library to reduce memory allocation pressure in hot paths. The primary optimization targets are:

1. **JSON Serialization Buffers** - Reused for table/view metadata responses
2. **Token Generation Buffers** - Reused for OAuth token generation

## Pool Configuration

Pool settings can be configured via environment variables:

```bash
# Buffer pool initial size (default: 4KB)
export BINGSAN_POOL_BUFFER_INITIAL_SIZE=4096

# Maximum buffer size before discard (default: 64KB)
export BINGSAN_POOL_BUFFER_MAX_SIZE=65536
```

Or in `config.yaml`:

```yaml
pool:
  buffer_initial_size: 4096   # 4KB - typical JSON metadata size
  buffer_max_size: 65536      # 64KB - discard oversized buffers
```

## Prometheus Metrics

Pool performance is exposed via Prometheus metrics at `/metrics`:

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `bingsan_pool_gets_total` | Counter | pool | Total Get operations |
| `bingsan_pool_returns_total` | Counter | pool | Total Put operations |
| `bingsan_pool_discards_total` | Counter | pool | Oversized buffers discarded |
| `bingsan_pool_misses_total` | Counter | pool | Pool misses (new allocation) |

### Example Queries

**Pool hit rate** (estimated):
```promql
rate(bingsan_pool_returns_total{pool="buffer"}[5m])
/ rate(bingsan_pool_gets_total{pool="buffer"}[5m])
```

**Discard rate** (should be low):
```promql
rate(bingsan_pool_discards_total{pool="buffer"}[5m])
```

## Grafana Dashboard

Import the following queries to monitor pool health:

### Pool Utilization Panel
```promql
# Pool return rate (should be near 100%)
100 * rate(bingsan_pool_returns_total[5m]) / rate(bingsan_pool_gets_total[5m])
```

### Pool Discard Rate Panel
```promql
# Discard rate (should be near 0)
rate(bingsan_pool_discards_total[5m])
```

## Alerting Rules

### Low Pool Hit Rate
```yaml
groups:
  - name: bingsan_pool
    rules:
      - alert: LowPoolHitRate
        expr: |
          (rate(bingsan_pool_returns_total{pool="buffer"}[5m])
           / rate(bingsan_pool_gets_total{pool="buffer"}[5m])) < 0.8
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "Pool hit rate below 80%"
          description: "Buffer pool hit rate is {{ $value | humanizePercentage }}. Consider increasing pool size or checking for memory pressure."
```

### High Discard Rate
```yaml
      - alert: HighPoolDiscardRate
        expr: rate(bingsan_pool_discards_total{pool="buffer"}[5m]) > 10
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "High pool discard rate"
          description: "Discarding {{ $value }} buffers/sec. Large responses may be impacting pool efficiency."
```

## Benchmarks

Run benchmarks to verify performance:

```bash
# Run all pool benchmarks
go test -bench=BenchmarkPool -benchmem ./tests/benchmark/...

# Compare with baseline
go test -bench=BenchmarkBaseline -benchmem ./tests/benchmark/... | tee baseline.txt
go test -bench=BenchmarkPool -benchmem ./tests/benchmark/... | tee optimized.txt
benchstat baseline.txt optimized.txt
```

### Expected Results

| Benchmark | Target | Typical Result |
|-----------|--------|----------------|
| Table Metadata Serialization | <50ms avg | ~10µs |
| Large Schema (100+ cols) | <200ms | ~90µs |
| JSON Serialization | <1ms | ~10µs |
| Concurrent (100 goroutines) | No degradation | ~3µs/op |

### Memory Allocation Targets

| Metric | Target | Typical Result |
|--------|--------|----------------|
| Allocation reduction | ≥30% | 19-26% |
| Concurrent pool reuse | 100% | 100% |
| Pool hit rate | ≥80% | 100% |

## Tuning Guidelines

### When to Increase `buffer_initial_size`:
- If most responses are larger than 4KB
- If you see frequent buffer growth in profiles
- Typical value: 8192 (8KB) for larger metadata

### When to Increase `buffer_max_size`:
- If you have many tables with large schemas (100+ columns)
- If discard rate is high
- Typical value: 131072 (128KB) for large schemas

### When to Decrease `buffer_max_size`:
- If memory usage is a concern
- If most operations use small buffers
- Typical value: 32768 (32KB) for memory-constrained environments

## Troubleshooting

### High Memory Usage
1. Check discard rate - if low, buffers may be too large
2. Reduce `buffer_max_size` to force more discards
3. Monitor heap profile with `go tool pprof`

### High Latency
1. Check pool hit rate - if low, pool may be undersized
2. Check GC pause times - pooling should reduce GC pressure
3. Profile with `go test -bench=. -cpuprofile=cpu.prof`

### Pool Not Being Used
1. Verify handlers use `pool.GetBuffer()` / `pool.PutBuffer()`
2. Check for `defer` patterns to ensure buffers are returned
3. Verify metrics are being collected at `/metrics`
