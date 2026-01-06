---
title: "Performance"
weight: 5
bookCollapseSection: true
---

# Performance

Bingsan is optimized for high-throughput metadata operations with minimal memory overhead. This section covers the performance features, tuning options, and benchmarking capabilities.

## Overview

Bingsan implements several performance optimizations:

- **Object Pooling** - Reuses memory buffers to reduce GC pressure
- **Distributed Locking** - PostgreSQL-based locks with configurable timeouts
- **Optimized Serialization** - Uses goccy/go-json for fast JSON encoding

```
┌─────────────────────────────────────────────────────────────┐
│                      Request Flow                           │
└─────────────────────────┬───────────────────────────────────┘
                          │
                          ▼
┌─────────────────────────────────────────────────────────────┐
│                    Object Pool                              │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐         │
│  │ BufferPool  │  │  BytePool   │  │   Metrics   │         │
│  │ (JSON/IO)   │  │  (Tokens)   │  │ (Prometheus)│         │
│  └─────────────┘  └─────────────┘  └─────────────┘         │
└─────────────────────────┬───────────────────────────────────┘
                          │
                          ▼
┌─────────────────────────────────────────────────────────────┐
│                  PostgreSQL Locking                         │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐         │
│  │ lock_timeout│  │   Retries   │  │  Advisory   │         │
│  │  (per-tx)   │  │  (backoff)  │  │   Locks     │         │
│  └─────────────┘  └─────────────┘  └─────────────┘         │
└─────────────────────────────────────────────────────────────┘
```

## Performance Targets

| Metric | Target | Typical |
|--------|--------|---------|
| Table metadata serialization | <50ms | ~10µs |
| Large schema (100+ cols) | <200ms | ~90µs |
| Memory allocation reduction | ≥30% | 19-26% |
| GC pause (p99) | <10ms | <5ms |
| Pool hit rate | ≥80% | ~100% |

## Configuration

Pool and locking settings are configured in `config.yaml`:

```yaml
catalog:
  lock_timeout: 30s
  lock_retry_interval: 100ms
  max_lock_retries: 100
```

Or via environment variables:

```bash
ICEBERG_CATALOG_LOCK_TIMEOUT=30s
ICEBERG_CATALOG_LOCK_RETRY_INTERVAL=100ms
ICEBERG_CATALOG_MAX_LOCK_RETRIES=100
```

## Sections

- [Object Pooling]({{< relref "/docs/performance/pooling" >}}) - Memory buffer reuse for reduced allocations
- [Distributed Locking]({{< relref "/docs/performance/locking" >}}) - PostgreSQL-based locking with retry logic
- [Benchmarking]({{< relref "/docs/performance/benchmarking" >}}) - Load testing with Apache Polaris Tools
- [Metrics]({{< relref "/docs/performance/metrics" >}}) - Prometheus metrics for monitoring
- [Tuning]({{< relref "/docs/performance/tuning" >}}) - Performance tuning guidelines
