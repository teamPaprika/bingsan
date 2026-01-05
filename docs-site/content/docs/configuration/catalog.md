---
title: "Catalog"
weight: 5
---

# Catalog Configuration

Configure catalog behavior and locking.

## Options

```yaml
catalog:
  prefix: ""
  default_warehouse: ""
  lock_timeout: 30s
  lock_retry_interval: 100ms
  max_lock_retries: 100
```

## Reference

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `prefix` | string | `""` | Catalog name prefix for multi-catalog deployments |
| `default_warehouse` | string | `""` | Default warehouse location (overrides `storage.warehouse`) |
| `lock_timeout` | duration | `30s` | Maximum time to hold distributed locks |
| `lock_retry_interval` | duration | `100ms` | Interval between lock acquisition retries |
| `max_lock_retries` | integer | `100` | Maximum lock acquisition attempts |

## Environment Variables

```bash
ICEBERG_CATALOG_PREFIX=my-catalog
ICEBERG_CATALOG_DEFAULT_WAREHOUSE=s3://bucket/warehouse
ICEBERG_CATALOG_LOCK_TIMEOUT=30s
ICEBERG_CATALOG_LOCK_RETRY_INTERVAL=100ms
ICEBERG_CATALOG_MAX_LOCK_RETRIES=100
```

---

## Catalog Prefix

Use a prefix to support multiple logical catalogs on a single Bingsan instance:

```yaml
catalog:
  prefix: "production"
```

Clients can then access the catalog at:

```
GET /v1/production/namespaces
POST /v1/production/namespaces/{namespace}/tables
```

The prefix is included in the API path but not stored in the database.

### Multi-Catalog Deployment

For separate production and staging catalogs:

```yaml
# production instance
catalog:
  prefix: "prod"

# staging instance
catalog:
  prefix: "staging"
```

Or run separate Bingsan instances:

```
http://catalog.example.com/v1/prod/namespaces
http://catalog.example.com/v1/staging/namespaces
```

---

## Default Warehouse

Override the storage warehouse for table creation:

```yaml
storage:
  warehouse: s3://bucket/default-path

catalog:
  default_warehouse: s3://bucket/special-warehouse
```

Tables created without a location will use `default_warehouse` if set, otherwise `storage.warehouse`.

---

## Distributed Locking

Bingsan uses PostgreSQL row-level locking with configurable timeouts for distributed locking, ensuring consistency when multiple instances run against the same database.

{{< hint info >}}
For detailed implementation and advanced tuning, see [Distributed Locking]({{< relref "/docs/performance/locking" >}}).
{{< /hint >}}

### How It Works

1. When a table commit is requested, Bingsan acquires a lock on the table
2. The lock prevents concurrent commits to the same table
3. If another commit is in progress, the request retries
4. After `max_lock_retries`, the request fails

### Lock Configuration

```yaml
catalog:
  lock_timeout: 30s         # How long to hold a lock
  lock_retry_interval: 100ms  # Time between retries
  max_lock_retries: 100       # Maximum retry attempts
```

### Total Wait Time

Maximum wait time = `lock_retry_interval` × `max_lock_retries`

Default: 100ms × 100 = 10 seconds

### High-Contention Settings

For high-contention workloads:

```yaml
catalog:
  lock_timeout: 60s
  lock_retry_interval: 50ms
  max_lock_retries: 300
```

### Low-Latency Settings

For latency-sensitive workloads:

```yaml
catalog:
  lock_timeout: 15s
  lock_retry_interval: 20ms
  max_lock_retries: 50
```

---

## Optimistic Concurrency

In addition to distributed locks, Bingsan uses optimistic concurrency control:

1. Client provides requirements (e.g., `assert-current-schema-id`)
2. Server checks requirements against current state
3. If requirements pass, update is applied
4. If requirements fail, `409 Conflict` is returned

### Example

```json
{
  "requirements": [
    {"type": "assert-current-schema-id", "current-schema-id": 0}
  ],
  "updates": [
    {"action": "set-current-schema", "schema-id": 1}
  ]
}
```

If another client modified the schema between read and write, the commit fails.

---

## Transaction Isolation

Bingsan uses PostgreSQL's SERIALIZABLE isolation for multi-table transactions, ensuring:

- **Atomicity**: All table changes succeed or all fail
- **Consistency**: Requirements are checked atomically
- **Isolation**: Concurrent transactions don't interfere
- **Durability**: Committed changes persist

---

## Best Practices

### Lock Timeout

- Set higher than your slowest expected commit
- Include time for network latency
- Default (30s) is suitable for most workloads

### Retry Configuration

- `lock_retry_interval`: Lower values reduce latency, increase database load
- `max_lock_retries`: Higher values handle more contention, but increase wait time

### Monitoring

Watch for lock contention:

```promql
# Lock wait time
rate(iceberg_db_wait_duration_seconds_total[5m])

# Lock failures
rate(iceberg_commits_total{status="lock_failed"}[5m])
```

### Troubleshooting

**Commits timing out**:
- Increase `max_lock_retries` or `lock_timeout`
- Reduce concurrent commit rate
- Check for long-running transactions

**High latency on commits**:
- Reduce `lock_retry_interval` for faster retries
- Check database connection pool usage
- Verify network latency to PostgreSQL
