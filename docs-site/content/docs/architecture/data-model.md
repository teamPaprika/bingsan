---
title: "Data Model"
weight: 2
---

# Data Model

Bingsan stores all catalog metadata in PostgreSQL. This page describes the database schema.

## Overview

```
┌─────────────────┐     ┌─────────────────┐
│   namespaces    │     │     tables      │
├─────────────────┤     ├─────────────────┤
│ id              │────▶│ namespace_id    │
│ name            │     │ name            │
│ properties      │     │ metadata_loc    │
└─────────────────┘     │ metadata        │
                        └─────────────────┘
                                │
                                ▼
┌─────────────────┐     ┌─────────────────┐
│     views       │     │   scan_plans    │
├─────────────────┤     ├─────────────────┤
│ namespace_id    │     │ table_id        │
│ name            │     │ plan_id         │
│ metadata_loc    │     │ status          │
│ metadata        │     │ tasks           │
└─────────────────┘     └─────────────────┘
```

## Tables

### namespaces

Stores namespace metadata.

| Column | Type | Description |
|--------|------|-------------|
| `id` | BIGSERIAL | Primary key |
| `name` | TEXT[] | Namespace name as array (e.g., `{analytics,events}`) |
| `properties` | JSONB | Namespace properties |
| `created_at` | TIMESTAMPTZ | Creation timestamp |
| `updated_at` | TIMESTAMPTZ | Last update timestamp |

```sql
CREATE TABLE namespaces (
    id BIGSERIAL PRIMARY KEY,
    name TEXT[] NOT NULL UNIQUE,
    properties JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_namespaces_name ON namespaces USING GIN (name);
```

### tables

Stores Iceberg table metadata.

| Column | Type | Description |
|--------|------|-------------|
| `id` | BIGSERIAL | Primary key |
| `namespace_id` | BIGINT | Foreign key to namespaces |
| `name` | TEXT | Table name |
| `table_uuid` | UUID | Iceberg table UUID |
| `metadata_location` | TEXT | Path to current metadata file |
| `metadata` | JSONB | Cached table metadata (optional) |
| `created_at` | TIMESTAMPTZ | Creation timestamp |
| `updated_at` | TIMESTAMPTZ | Last update timestamp |

```sql
CREATE TABLE tables (
    id BIGSERIAL PRIMARY KEY,
    namespace_id BIGINT NOT NULL REFERENCES namespaces(id),
    name TEXT NOT NULL,
    table_uuid UUID NOT NULL,
    metadata_location TEXT NOT NULL,
    metadata JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (namespace_id, name)
);

CREATE INDEX idx_tables_namespace ON tables (namespace_id);
CREATE INDEX idx_tables_uuid ON tables (table_uuid);
```

### views

Stores Iceberg view metadata.

| Column | Type | Description |
|--------|------|-------------|
| `id` | BIGSERIAL | Primary key |
| `namespace_id` | BIGINT | Foreign key to namespaces |
| `name` | TEXT | View name |
| `view_uuid` | UUID | Iceberg view UUID |
| `metadata_location` | TEXT | Path to current metadata file |
| `metadata` | JSONB | Cached view metadata |
| `created_at` | TIMESTAMPTZ | Creation timestamp |
| `updated_at` | TIMESTAMPTZ | Last update timestamp |

```sql
CREATE TABLE views (
    id BIGSERIAL PRIMARY KEY,
    namespace_id BIGINT NOT NULL REFERENCES namespaces(id),
    name TEXT NOT NULL,
    view_uuid UUID NOT NULL,
    metadata_location TEXT NOT NULL,
    metadata JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (namespace_id, name)
);
```

### scan_plans

Stores scan plan state for server-side planning.

| Column | Type | Description |
|--------|------|-------------|
| `id` | BIGSERIAL | Primary key |
| `plan_id` | UUID | External plan identifier |
| `table_id` | BIGINT | Foreign key to tables |
| `status` | TEXT | Plan status |
| `request` | JSONB | Original plan request |
| `tasks` | JSONB | Computed scan tasks |
| `statistics` | JSONB | Planning statistics |
| `created_at` | TIMESTAMPTZ | Creation timestamp |
| `completed_at` | TIMESTAMPTZ | Completion timestamp |

```sql
CREATE TABLE scan_plans (
    id BIGSERIAL PRIMARY KEY,
    plan_id UUID NOT NULL UNIQUE,
    table_id BIGINT NOT NULL REFERENCES tables(id),
    status TEXT NOT NULL DEFAULT 'submitted',
    request JSONB NOT NULL,
    tasks JSONB,
    statistics JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ
);

CREATE INDEX idx_scan_plans_table ON scan_plans (table_id);
CREATE INDEX idx_scan_plans_status ON scan_plans (status);
```

---

## Metadata Storage Strategy

### Database vs Object Storage

Bingsan uses a hybrid approach:

**PostgreSQL stores:**
- Namespace metadata
- Table/view registry (name, UUID, location)
- Scan plan state
- Cached metadata (optional)

**Object storage (S3/GCS) stores:**
- Full Iceberg metadata JSON files
- Manifest lists
- Manifests
- Data files

### Metadata Caching

Table metadata can be cached in PostgreSQL for faster reads:

```json
{
  "format-version": 2,
  "table-uuid": "...",
  "location": "s3://bucket/table",
  "schemas": [...],
  "current-schema-id": 0,
  ...
}
```

This avoids reading from object storage for every request.

---

## Locking Model

### Advisory Locks

PostgreSQL advisory locks ensure consistency:

```sql
-- Namespace-level lock (for CRUD)
SELECT pg_advisory_lock(hashtext('ns:' || $1));

-- Table-level lock (for commits)
SELECT pg_advisory_xact_lock($1);  -- table ID
```

### Lock Hierarchy

```
Namespace Lock
    └── Table Lock (for commits within namespace)
```

Locks are held for the minimum necessary duration.

---

## Migrations

Bingsan uses [golang-migrate](https://github.com/golang-migrate/migrate) for schema migrations.

### Migration Files

Located in `internal/db/migrations/`:

```
001_create_namespaces.up.sql
001_create_namespaces.down.sql
002_create_tables.up.sql
002_create_tables.down.sql
...
```

### Automatic Migrations

Migrations run automatically on startup:

```go
m, _ := migrate.New(
    "file://migrations",
    databaseURL,
)
m.Up()
```

### Manual Migrations

```bash
# Check current version
migrate -database "postgres://..." -path migrations version

# Apply migrations
migrate -database "postgres://..." -path migrations up

# Rollback one migration
migrate -database "postgres://..." -path migrations down 1
```

---

## Indexes

### Query Patterns

Indexes are optimized for common queries:

| Query | Index Used |
|-------|------------|
| List namespaces | `idx_namespaces_name` (GIN) |
| Get namespace by name | `namespaces_name_key` (UNIQUE) |
| List tables in namespace | `idx_tables_namespace` |
| Get table by name | `tables_namespace_id_name_key` (UNIQUE) |
| Find table by UUID | `idx_tables_uuid` |

### Performance Considerations

- GIN index on `namespaces.name` supports prefix queries
- Composite unique constraints double as indexes
- UUID index enables fast lookup by Iceberg table UUID

---

## Data Lifecycle

### Namespace

1. **Create**: Insert into `namespaces`
2. **Update**: Update `properties`, set `updated_at`
3. **Delete**: Delete from `namespaces` (must be empty)

### Table

1. **Create**: Insert into `tables`, write metadata to storage
2. **Commit**: Update `metadata_location`, optional metadata cache
3. **Drop**: Delete from `tables`, optionally purge storage

### View

Similar to tables, but stores view-specific metadata.

### Scan Plan

1. **Submit**: Insert into `scan_plans` with status `submitted`
2. **Planning**: Update status to `planning`, compute tasks
3. **Complete**: Update status to `complete`, store tasks
4. **Cleanup**: Delete expired plans (background job)

---

## Backup and Recovery

### PostgreSQL Backup

```bash
# Full backup
pg_dump -h localhost -U iceberg iceberg_catalog > backup.sql

# Restore
psql -h localhost -U iceberg iceberg_catalog < backup.sql
```

### Point-in-Time Recovery

Enable WAL archiving for PITR:

```sql
ALTER SYSTEM SET wal_level = replica;
ALTER SYSTEM SET archive_mode = on;
ALTER SYSTEM SET archive_command = 'cp %p /archive/%f';
```

### Metadata Consistency

Since full Iceberg metadata is in object storage, the database can be rebuilt from metadata files if needed.
