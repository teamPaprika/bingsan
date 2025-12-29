# Data Model: Object Pool Configuration

**Feature**: 002-perf-optimization
**Date**: 2025-12-24

## Overview

This feature introduces object pooling for performance optimization. The "data model" consists of pool configurations and metrics rather than database entities.

## Pool Types

### BufferPool

**Purpose**: Reusable byte buffers for JSON serialization

| Attribute | Type | Description |
|-----------|------|-------------|
| InitialSize | int | Initial buffer capacity (4096 bytes) |
| MaxSize | int | Maximum buffer size before discard (64KB) |

**Lifecycle**:
1. Get: Retrieve buffer from pool (or allocate new if empty)
2. Use: Write JSON data to buffer
3. Reset: Clear buffer contents (keep capacity)
4. Put: Return buffer to pool

**Constraints**:
- Buffers exceeding MaxSize are discarded (not returned to pool)
- Reset MUST be called before Put to prevent data leakage

---

### BytePool

**Purpose**: Reusable byte slices for token generation

| Attribute | Type | Description |
|-----------|------|-------------|
| SliceSize | int | Fixed slice size (32 bytes for tokens) |

**Lifecycle**:
1. Get: Retrieve slice from pool (or allocate new)
2. Use: Fill with random bytes
3. Put: Return to pool (no reset needed - will be overwritten)

**Constraints**:
- Fixed size slices only (no growth)
- Contents are overwritten on next use

---

## Pool Metrics

### PoolMetrics

**Purpose**: Prometheus metrics for pool utilization

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| pool_hits_total | Counter | pool | Items retrieved from pool |
| pool_misses_total | Counter | pool | New allocations (pool empty) |
| pool_returns_total | Counter | pool | Items returned to pool |
| pool_discards_total | Counter | pool | Items discarded (too large) |

**Relationships**:
- `hits + misses = total_gets` (requests served)
- `returns - discards = effective_returns` (items recycled)
- `hit_rate = hits / (hits + misses)` (pool effectiveness)

---

## State Transitions

### Buffer Lifecycle

```text
┌─────────┐    Get()     ┌─────────┐   Write()   ┌─────────┐
│  Pool   │ ──────────►  │  Active │ ──────────► │  Dirty  │
└─────────┘              └─────────┘             └─────────┘
     ▲                                                │
     │                                                │
     │           Reset() + Put()                      │
     └────────────────────────────────────────────────┘
```

### Metrics State

```text
Get() called:
  └─► Pool not empty → increment hits_total
  └─► Pool empty → increment misses_total

Put() called:
  └─► Buffer size ≤ MaxSize → increment returns_total
  └─► Buffer size > MaxSize → increment discards_total
```

---

## Configuration

### Default Values

| Pool | Setting | Default | Rationale |
|------|---------|---------|-----------|
| BufferPool | InitialSize | 4096 | Typical JSON metadata size |
| BufferPool | MaxSize | 65536 | Prevent memory bloat |
| BytePool | SliceSize | 32 | Token generation size |

### Environment Overrides (optional)

| Variable | Description |
|----------|-------------|
| BINGSAN_POOL_BUFFER_INITIAL | Override initial buffer size |
| BINGSAN_POOL_BUFFER_MAX | Override max buffer size |

---

## No Database Changes

This feature does not introduce any database schema changes. All pool state is in-memory and ephemeral (cleared on GC).
