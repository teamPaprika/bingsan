---
title: "Distributed Locking"
weight: 2
---

# Distributed Locking

Bingsan uses PostgreSQL row-level locking with configurable timeouts and retry logic to handle concurrent modifications safely across multiple instances.

## Overview

When multiple Bingsan instances modify the same resource simultaneously, locking ensures:

- **Consistency**: Only one operation modifies a resource at a time
- **Isolation**: Operations don't see partial states
- **Automatic Recovery**: Failed locks are retried with backoff

```
Instance A ──┐                    ┌── Instance B
             │                    │
             ▼                    ▼
      ┌──────────────────────────────────────┐
      │         PostgreSQL Lock              │
      │  ┌──────────────────────────────┐    │
      │  │ SELECT ... FOR UPDATE        │    │
      │  │ SET LOCAL lock_timeout       │    │
      │  └──────────────────────────────┘    │
      │                                      │
      │  Instance A: Acquired ✓              │
      │  Instance B: Waiting... (timeout)    │
      │              Retry after interval    │
      └──────────────────────────────────────┘
```

## Configuration

Configure locking via `config.yaml`:

```yaml
catalog:
  # Maximum time to wait for a lock
  lock_timeout: 30s

  # Time between retry attempts
  lock_retry_interval: 100ms

  # Maximum number of retries before failing
  max_lock_retries: 100
```

### Environment Variables

```bash
ICEBERG_CATALOG_LOCK_TIMEOUT=30s
ICEBERG_CATALOG_LOCK_RETRY_INTERVAL=100ms
ICEBERG_CATALOG_MAX_LOCK_RETRIES=100
```

## How It Works

### Lock Acquisition Flow

```
┌─────────────────────────────────────────────────────────────────┐
│                    Lock Acquisition Flow                         │
└─────────────────────────────────────────────────────────────────┘

1. Begin Transaction
   │
   ▼
2. SET LOCAL lock_timeout = '30s'
   │
   ▼
3. Execute operation (SELECT ... FOR UPDATE)
   │
   ├──Success──► 4. Commit Transaction ──► Done ✓
   │
   └──Lock Timeout (55P03)
         │
         ▼
   5. Rollback Transaction
         │
         ▼
   6. Wait retry_interval (100ms)
         │
         ▼
   7. Retry count < max_retries?
         │
         ├──Yes──► Go to Step 1
         │
         └──No──► Return ErrLockTimeout ✗
```

### PostgreSQL Lock Timeout

Each transaction sets `lock_timeout` locally:

```sql
BEGIN;
SET LOCAL lock_timeout = '30000ms';
SELECT * FROM tables WHERE id = $1 FOR UPDATE;
-- ... perform update ...
COMMIT;
```

If the lock isn't acquired within `lock_timeout`, PostgreSQL returns error code `55P03`.

### Retry Logic

When a lock timeout occurs:

1. **Transaction is rolled back** (no partial state)
2. **Wait for retry interval** (prevents thundering herd)
3. **Retry up to max_retries times**
4. **Return error if all retries exhausted**

## Configuration Options

| Option | Default | Description |
|--------|---------|-------------|
| `lock_timeout` | 30s | Max time to wait for a single lock attempt |
| `lock_retry_interval` | 100ms | Wait time between retry attempts |
| `max_lock_retries` | 100 | Maximum retry attempts before failing |

### Total Wait Time

Maximum time before failure:

```
max_wait = lock_timeout + (max_lock_retries × lock_retry_interval)
         = 30s + (100 × 100ms)
         = 30s + 10s
         = 40s
```

## Use Cases

### Table Commits

When committing table changes:

```go
err := db.WithLock(ctx, db.LockConfig{
    Timeout:       30 * time.Second,
    RetryInterval: 100 * time.Millisecond,
    MaxRetries:    100,
}, func(tx pgx.Tx) error {
    // Load current table state
    table, err := loadTable(ctx, tx, tableID)
    if err != nil {
        return err
    }

    // Validate requirements
    if err := validateRequirements(table, req); err != nil {
        return err
    }

    // Apply updates
    return updateTable(ctx, tx, tableID, updates)
})
```

### Namespace Operations

When modifying namespaces:

```go
err := db.WithLock(ctx, lockConfig, func(tx pgx.Tx) error {
    return createNamespace(ctx, tx, namespace)
})
```

## Tuning Guidelines

### High Contention Workloads

If many clients modify the same tables:

```yaml
catalog:
  lock_timeout: 5s         # Fail faster
  lock_retry_interval: 50ms # Retry more frequently
  max_lock_retries: 200     # More retry attempts
```

### Low Contention Workloads

If tables are rarely modified concurrently:

```yaml
catalog:
  lock_timeout: 60s        # Wait longer
  lock_retry_interval: 500ms # Less aggressive retries
  max_lock_retries: 10      # Fewer retries
```

### Batch Processing

For batch jobs that can wait:

```yaml
catalog:
  lock_timeout: 120s       # Very patient
  lock_retry_interval: 1s  # Conservative retries
  max_lock_retries: 60     # 2 minute total wait
```

## Error Handling

### ErrLockTimeout

Returned when all retries are exhausted:

```go
if errors.Is(err, db.ErrLockTimeout) {
    // Lock could not be acquired
    return fiber.NewError(
        fiber.StatusConflict,
        "table is being modified by another operation",
    )
}
```

### Context Cancellation

If the context is cancelled during retry:

```go
ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
defer cancel()

err := db.WithLock(ctx, lockConfig, func(tx pgx.Tx) error {
    // ...
})

if errors.Is(err, context.DeadlineExceeded) {
    // Request timed out before lock acquired
}
```

### Serialization Failures

PostgreSQL serialization errors (40001) are also detected:

```go
if db.IsSerializationError(err) {
    // Concurrent modification detected
    // Client should retry
}
```

## Monitoring

### Lock Wait Metrics

Monitor lock contention via PostgreSQL:

```sql
-- Active locks
SELECT * FROM pg_locks WHERE NOT granted;

-- Lock wait statistics
SELECT * FROM pg_stat_activity
WHERE wait_event_type = 'Lock';
```

### Application Metrics

Track lock retries in your application:

```promql
# Lock timeout errors
rate(iceberg_handler_errors_total{error="lock_timeout"}[5m])

# Lock retry rate
rate(iceberg_lock_retries_total[5m])
```

## Best Practices

### Keep Transactions Short

```go
// Good: Minimal work inside lock
err := db.WithLock(ctx, cfg, func(tx pgx.Tx) error {
    return tx.Exec(ctx, "UPDATE tables SET ...")
})

// Bad: External calls inside lock
err := db.WithLock(ctx, cfg, func(tx pgx.Tx) error {
    callExternalService()  // May be slow!
    return tx.Exec(ctx, "UPDATE tables SET ...")
})
```

### Use Appropriate Timeouts

```go
// For quick operations
cfg := db.LockConfig{
    Timeout:       5 * time.Second,
    RetryInterval: 50 * time.Millisecond,
    MaxRetries:    20,
}

// For batch operations
cfg := db.LockConfig{
    Timeout:       60 * time.Second,
    RetryInterval: time.Second,
    MaxRetries:    10,
}
```

### Handle Errors Gracefully

```go
err := db.WithLock(ctx, cfg, fn)
switch {
case errors.Is(err, db.ErrLockTimeout):
    return fiber.NewError(409, "resource busy")
case errors.Is(err, context.Canceled):
    return fiber.NewError(499, "request cancelled")
default:
    return err
}
```

## Troubleshooting

### Frequent Lock Timeouts

**Symptoms**: Many `ErrLockTimeout` errors

**Causes**:
- High write contention on same tables
- Long-running transactions holding locks
- Database performance issues

**Solutions**:
1. Increase `max_lock_retries`
2. Decrease `lock_timeout` (fail faster, retry sooner)
3. Check for slow queries holding locks
4. Partition workloads across different tables

### Slow Lock Acquisition

**Symptoms**: High latency on writes

**Causes**:
- PostgreSQL connection issues
- Lock contention
- Slow disk I/O

**Solutions**:
1. Check PostgreSQL performance
2. Monitor `pg_stat_activity` for blocked queries
3. Consider read replicas for read-heavy workloads

### Deadlocks

**Symptoms**: Transactions waiting indefinitely

**Causes**:
- Multiple tables locked in different orders

**Solutions**:
1. Bingsan acquires locks in consistent order
2. PostgreSQL detects and breaks deadlocks (40P01)
3. Retry logic handles broken deadlocks automatically
