---
title: "Request Flow"
weight: 1
---

# Request Flow

Understanding how Bingsan processes requests helps with debugging and performance optimization.

## HTTP Request Lifecycle

```
┌──────────────┐
│   Client     │
└──────┬───────┘
       │ HTTP Request
       ▼
┌──────────────────────────────────────────────┐
│              Fiber HTTP Server               │
├──────────────────────────────────────────────┤
│  1. Request ID Middleware                    │
│  2. Recovery Middleware (panic handling)     │
│  3. Prometheus Metrics Middleware            │
│  4. CORS Middleware                          │
│  5. Logger Middleware                        │
│  6. Auth Middleware (if enabled)             │
├──────────────────────────────────────────────┤
│              Route Handler                   │
├──────────────────────────────────────────────┤
│           Database Operations                │
├──────────────────────────────────────────────┤
│           Event Publishing                   │
└──────────────────────────────────────────────┘
       │
       ▼ HTTP Response
┌──────────────┐
│   Client     │
└──────────────┘
```

## Middleware Stack

### 1. Request ID

Assigns a unique ID to each request for tracing:

```go
c.Locals("requestId", uuid.New().String())
```

Returned in `X-Request-ID` header.

### 2. Recovery

Catches panics and returns 500 Internal Server Error:

```go
recover.New(recover.Config{
    EnableStackTrace: config.Server.Debug,
})
```

### 3. Prometheus Metrics

Records request metrics:

- `iceberg_catalog_http_requests_total` - Counter
- `iceberg_catalog_http_request_duration_seconds` - Histogram

Paths `/health`, `/ready`, `/metrics` are excluded.

### 4. CORS

Allows cross-origin requests:

```go
cors.New(cors.Config{
    AllowOrigins: "*",
    AllowMethods: "GET,POST,PUT,DELETE,HEAD,OPTIONS",
    AllowHeaders: "Origin,Content-Type,Accept,Authorization,X-Iceberg-Access-Delegation",
})
```

### 5. Logger

Logs request details:

```json
{
  "level": "INFO",
  "msg": "request",
  "method": "GET",
  "path": "/v1/namespaces",
  "status": 200,
  "latency": "1.234ms",
  "request_id": "abc-123"
}
```

### 6. Authentication

When enabled, validates bearer tokens:

```go
if config.Auth.Enabled {
    v1.Use(middleware.Auth(config, db))
}
```

Returns 401 if token is invalid or missing.

---

## Table Commit Flow

The most complex operation is a table commit:

```
┌──────────────┐
│   Client     │
└──────┬───────┘
       │ POST /v1/namespaces/{ns}/tables/{table}
       ▼
┌──────────────────────────────────────────────┐
│           CommitTable Handler                │
├──────────────────────────────────────────────┤
│  1. Parse request body                       │
│  2. Validate table identifier                │
│  3. Acquire table lock (PostgreSQL advisory) │
│  4. Begin transaction                        │
│  5. Load current metadata                    │
│  6. Check requirements                       │
│  7. Apply updates                            │
│  8. Write new metadata file (S3/GCS)         │
│  9. Update database                          │
│ 10. Commit transaction                       │
│ 11. Release lock                             │
│ 12. Publish event                            │
└──────────────────────────────────────────────┘
       │
       ▼
┌──────────────┐
│   Client     │
└──────────────┘
```

### Lock Acquisition

```go
// Acquire advisory lock on table
_, err := tx.Exec(ctx, "SELECT pg_advisory_xact_lock($1)", tableID)
```

The lock is held for the duration of the transaction.

### Requirement Checking

Each requirement is validated:

```go
switch req.Type {
case "assert-table-uuid":
    if current.UUID != req.UUID {
        return CommitFailedError("UUID mismatch")
    }
case "assert-current-schema-id":
    if current.SchemaID != req.SchemaID {
        return CommitFailedError("Schema ID mismatch")
    }
// ... other requirements
}
```

If any requirement fails, the entire commit is rejected.

### Update Application

Updates are applied in order:

```go
for _, update := range updates {
    switch update.Action {
    case "add-snapshot":
        metadata.AddSnapshot(update.Snapshot)
    case "set-current-schema":
        metadata.SetCurrentSchemaID(update.SchemaID)
    // ... other actions
    }
}
```

---

## Transaction Commit Flow

Multi-table transactions have additional complexity:

```
┌──────────────┐
│   Client     │
└──────┬───────┘
       │ POST /v1/transactions/commit
       ▼
┌──────────────────────────────────────────────┐
│        CommitTransaction Handler             │
├──────────────────────────────────────────────┤
│  1. Parse all table changes                  │
│  2. Sort tables by ID (deadlock prevention)  │
│  3. Begin SERIALIZABLE transaction           │
│  4. For each table:                          │
│     a. Acquire lock                          │
│     b. Check requirements                    │
│     c. Apply updates                         │
│  5. Write all metadata files                 │
│  6. Update all database records              │
│  7. Commit transaction                       │
│  8. Publish events                           │
└──────────────────────────────────────────────┘
```

Tables are locked in consistent order to prevent deadlocks.

---

## Event Publishing

After successful operations, events are published:

```go
event := events.Event{
    Type:      "table_updated",
    Timestamp: time.Now(),
    Namespace: namespace,
    Table:     table,
}
broker.Publish(event)
```

Events are delivered to all connected WebSocket clients.

---

## Error Handling

### Client Errors (4xx)

- **400 Bad Request**: Invalid JSON, missing fields
- **401 Unauthorized**: Missing or invalid token
- **404 Not Found**: Namespace/table doesn't exist
- **409 Conflict**: Requirement check failed

### Server Errors (5xx)

- **500 Internal Server Error**: Unexpected errors, database failures
- **503 Service Unavailable**: Database unavailable

### Error Response Format

```json
{
  "error": {
    "message": "Requirement not met: current-schema-id is 1, expected 0",
    "type": "CommitFailedException",
    "code": 409
  }
}
```

---

## Performance Considerations

### Connection Pooling

Database connections are pooled:

```go
poolConfig.MaxConns = config.Database.MaxOpenConns
poolConfig.MinConns = config.Database.MaxIdleConns
```

### Goroutine-per-Request

Each request runs in its own goroutine:

- No thread pool limits
- Automatic scheduling
- Efficient memory usage

### JSON Serialization

Uses [goccy/go-json](https://github.com/goccy/go-json) for faster JSON:

```go
fiber.Config{
    JSONEncoder: json.Marshal,
    JSONDecoder: json.Unmarshal,
}
```

---

## Debugging

### Request Tracing

Use the request ID to trace through logs:

```bash
grep "request_id.*abc-123" /var/log/bingsan.log
```

### Slow Request Analysis

Check the request duration in metrics:

```promql
histogram_quantile(0.99,
  rate(iceberg_catalog_http_request_duration_seconds_bucket{path="/v1/namespaces/{namespace}/tables/{table}"}[5m])
)
```

### Database Queries

Enable PostgreSQL query logging for slow query analysis:

```sql
ALTER SYSTEM SET log_min_duration_statement = 100;
```
