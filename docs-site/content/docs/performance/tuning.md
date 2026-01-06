---
title: "Tuning"
weight: 5
---

# Performance Tuning

This guide covers how to tune Bingsan for optimal performance based on your workload characteristics.

## Quick Reference

| Workload | Lock Timeout | Retry Interval | Max Retries | Buffer Size |
|----------|--------------|----------------|-------------|-------------|
| Low latency | 5s | 50ms | 20 | 4KB |
| High throughput | 30s | 100ms | 100 | 4KB |
| Large schemas | 30s | 100ms | 100 | 8-16KB |
| Batch processing | 120s | 1s | 60 | 4KB |

---

## Workload Profiles

### Low Latency

Prioritize fast responses over throughput:

```yaml
catalog:
  lock_timeout: 5s
  lock_retry_interval: 50ms
  max_lock_retries: 20

server:
  read_timeout: 10s
  write_timeout: 10s

database:
  max_open_conns: 50
  max_idle_conns: 25
```

**Characteristics:**
- Fails fast on lock contention
- More aggressive retries
- Higher connection pool

### High Throughput

Maximize requests per second:

```yaml
catalog:
  lock_timeout: 30s
  lock_retry_interval: 100ms
  max_lock_retries: 100

server:
  read_timeout: 60s
  write_timeout: 60s
  idle_timeout: 300s

database:
  max_open_conns: 100
  max_idle_conns: 50
  conn_max_lifetime: 30m
```

**Characteristics:**
- Patient lock acquisition
- Long-lived connections
- Higher resource usage

### Large Schemas

For tables with 100+ columns:

```yaml
catalog:
  lock_timeout: 30s
  lock_retry_interval: 100ms
  max_lock_retries: 100

# Compile-time constants in internal/pool/buffer.go:
# DefaultBufferSize = 8192   (8KB)
# MaxBufferSize = 131072     (128KB)
```

**Characteristics:**
- Larger initial buffers
- Higher max buffer size
- Reduced buffer discards

### Batch Processing

For bulk operations:

```yaml
catalog:
  lock_timeout: 120s
  lock_retry_interval: 1s
  max_lock_retries: 60

server:
  read_timeout: 300s
  write_timeout: 300s

database:
  max_open_conns: 25
  conn_max_lifetime: 60m
```

**Characteristics:**
- Very patient operations
- Conservative resources
- Long timeouts

---

## Tuning by Symptom

### High Latency

**Symptoms:**
- p99 latency > 100ms
- Slow table loads
- Client timeouts

**Diagnosis:**
```promql
# Check lock wait time
rate(iceberg_db_wait_duration_seconds_total[5m])

# Check pool discard rate
rate(bingsan_pool_discards_total[5m])

# Check connection saturation
iceberg_db_connections_in_use / iceberg_db_connections_max
```

**Solutions:**

1. **Lock contention** - Reduce `lock_timeout`, increase `max_lock_retries`
2. **Pool discards** - Increase `MaxBufferSize` for large schemas
3. **Connection pool** - Increase `max_open_conns`

### High Memory Usage

**Symptoms:**
- Memory growth over time
- OOM kills
- High GC pressure

**Diagnosis:**
```bash
# Heap profile
curl http://localhost:8181/debug/pprof/heap > heap.prof
go tool pprof heap.prof

# Check pool stats
curl http://localhost:8181/metrics | grep bingsan_pool
```

**Solutions:**

1. **Buffer leaks** - Check all code paths return buffers
2. **Large buffers** - Reduce `MaxBufferSize`
3. **Connection bloat** - Reduce `max_open_conns`

### Lock Timeout Errors

**Symptoms:**
- `ErrLockTimeout` errors
- 409 Conflict responses
- Failed commits

**Diagnosis:**
```sql
-- Check active locks
SELECT * FROM pg_locks WHERE NOT granted;

-- Check blocking queries
SELECT * FROM pg_stat_activity
WHERE wait_event_type = 'Lock';
```

**Solutions:**

1. **High contention** - Increase `max_lock_retries`
2. **Slow transactions** - Keep transactions short
3. **Deadlocks** - Bingsan handles these automatically

### Connection Exhaustion

**Symptoms:**
- `too many connections` errors
- Connection timeouts
- Slow query start

**Diagnosis:**
```promql
# Connection utilization
iceberg_db_connections_in_use / iceberg_db_connections_max > 0.9
```

**Solutions:**

1. **Increase pool** - Raise `max_open_conns`
2. **Add PgBouncer** - Connection multiplexing
3. **Reduce instances** - Fewer Bingsan replicas

---

## Database Tuning

### PostgreSQL Settings

```sql
-- Increase max connections
ALTER SYSTEM SET max_connections = 500;

-- Lock timeout (server-wide default)
ALTER SYSTEM SET lock_timeout = '30s';

-- Statement timeout
ALTER SYSTEM SET statement_timeout = '60s';

-- Effective cache size (75% of RAM)
ALTER SYSTEM SET effective_cache_size = '12GB';

-- Shared buffers (25% of RAM)
ALTER SYSTEM SET shared_buffers = '4GB';

-- Work memory
ALTER SYSTEM SET work_mem = '256MB';
```

### Index Optimization

Ensure indexes exist for common queries:

```sql
-- Verify indexes
\di iceberg_*

-- Analyze tables
ANALYZE VERBOSE;

-- Check index usage
SELECT schemaname, tablename, indexname, idx_scan
FROM pg_stat_user_indexes
ORDER BY idx_scan DESC;
```

### Connection Pooling with PgBouncer

For many Bingsan instances:

```ini
# pgbouncer.ini
[databases]
iceberg_catalog = host=postgres port=5432 dbname=iceberg_catalog

[pgbouncer]
pool_mode = transaction
max_client_conn = 1000
default_pool_size = 50
reserve_pool_size = 10
reserve_pool_timeout = 3
```

---

## Resource Sizing

### Memory

```
memory_per_instance = base + (concurrent_requests × request_memory)
                   ≈ 50MB + (500 × 100KB)
                   ≈ 100MB typical
                   ≈ 200MB peak
```

### CPU

```
cpu_per_instance ≈ 0.2 cores idle
                 ≈ 1 core under load
```

### Instances

```
instances = (peak_rps / rps_per_instance) × 1.5
          = (10000 / 5000) × 1.5
          = 3 instances minimum
```

---

## Profiling

### CPU Profile

```bash
# Start profiling
curl http://localhost:8181/debug/pprof/profile?seconds=30 > cpu.prof

# Analyze
go tool pprof -http=:8080 cpu.prof
```

### Memory Profile

```bash
# Heap snapshot
curl http://localhost:8181/debug/pprof/heap > heap.prof

# Analyze
go tool pprof -http=:8080 heap.prof
```

### Trace

```bash
# Capture trace
curl http://localhost:8181/debug/pprof/trace?seconds=5 > trace.out

# Analyze
go tool trace trace.out
```

### Goroutine Analysis

```bash
# Goroutine dump
curl http://localhost:8181/debug/pprof/goroutine > goroutine.prof

# Analyze
go tool pprof goroutine.prof
```

---

## Benchmark-Driven Tuning

### Step 1: Establish Baseline

```bash
go test -bench=. -benchmem ./tests/benchmark/... | tee baseline.txt
```

### Step 2: Identify Bottlenecks

```bash
go test -bench=BenchmarkTable -cpuprofile=cpu.prof ./tests/benchmark/...
go tool pprof -top cpu.prof
```

### Step 3: Make Changes

Adjust configuration or code based on profile results.

### Step 4: Measure Impact

```bash
go test -bench=. -benchmem ./tests/benchmark/... | tee optimized.txt
benchstat baseline.txt optimized.txt
```

### Step 5: Validate in Production

```bash
# Load test
cd benchmarks
make read-benchmark

# Monitor metrics
watch -n 1 'curl -s localhost:8181/metrics | grep bingsan_pool'
```

---

## Checklist

### Pre-Production

- [ ] Set appropriate `lock_timeout` for your workload
- [ ] Configure `max_open_conns` based on expected load
- [ ] Enable Prometheus metrics collection
- [ ] Set up alerting for pool health
- [ ] Run load tests with realistic data

### Production Monitoring

- [ ] Pool hit rate > 80%
- [ ] Pool discard rate < 1%
- [ ] Lock timeout rate < 1%
- [ ] Connection utilization < 90%
- [ ] GC pause p99 < 10ms

### Troubleshooting Resources

- [Object Pooling]({{< relref "/docs/performance/pooling" >}}) - Buffer pool details
- [Distributed Locking]({{< relref "/docs/performance/locking" >}}) - Lock configuration
- [Metrics]({{< relref "/docs/performance/metrics" >}}) - Monitoring setup
- [Benchmarking]({{< relref "/docs/performance/benchmarking" >}}) - Load testing
