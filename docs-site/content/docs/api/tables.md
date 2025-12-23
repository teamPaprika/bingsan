---
title: "Tables"
weight: 3
---

# Tables API

Tables are the primary data containers in Iceberg, holding schema, partition specs, and metadata.

## List Tables

List all tables in a namespace.

### Request

```http
GET /v1/namespaces/{namespace}/tables
GET /v1/namespaces/{namespace}/tables?pageToken={token}&pageSize={size}
```

### Query Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `pageToken` | string | No | Pagination token |
| `pageSize` | integer | No | Maximum results (default: 100) |

### Response

```json
{
  "identifiers": [
    {"namespace": ["analytics"], "name": "user_events"},
    {"namespace": ["analytics"], "name": "page_views"}
  ],
  "next-page-token": null
}
```

### Example

```bash
curl http://localhost:8181/v1/namespaces/analytics/tables
```

---

## Create Table

Create a new Iceberg table.

### Request

```http
POST /v1/namespaces/{namespace}/tables
```

### Request Body

```json
{
  "name": "user_events",
  "location": "s3://bucket/warehouse/analytics/user_events",
  "schema": {
    "type": "struct",
    "schema-id": 0,
    "fields": [
      {"id": 1, "name": "event_id", "required": true, "type": "string"},
      {"id": 2, "name": "user_id", "required": true, "type": "long"},
      {"id": 3, "name": "event_type", "required": true, "type": "string"},
      {"id": 4, "name": "event_time", "required": true, "type": "timestamptz"},
      {"id": 5, "name": "properties", "required": false, "type": {
        "type": "map",
        "key-id": 6,
        "key": "string",
        "value-id": 7,
        "value": "string"
      }}
    ]
  },
  "partition-spec": {
    "spec-id": 0,
    "fields": [
      {"source-id": 4, "field-id": 1000, "name": "event_day", "transform": "day"}
    ]
  },
  "write-order": {
    "order-id": 0,
    "fields": [
      {"source-id": 4, "direction": "asc", "null-order": "nulls-last"}
    ]
  },
  "stage-create": false,
  "properties": {
    "format-version": "2",
    "write.parquet.compression-codec": "zstd"
  }
}
```

### Request Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | Yes | Table name |
| `location` | string | No | Custom table location (uses warehouse default if omitted) |
| `schema` | object | Yes | Iceberg schema definition |
| `partition-spec` | object | No | Partition specification |
| `write-order` | object | No | Sort order for writes |
| `stage-create` | boolean | No | If true, creates staged table (no data files) |
| `properties` | object | No | Table properties |

### Response

Returns the full table metadata:

```json
{
  "metadata-location": "s3://bucket/warehouse/analytics/user_events/metadata/00000-uuid.metadata.json",
  "metadata": {
    "format-version": 2,
    "table-uuid": "550e8400-e29b-41d4-a716-446655440000",
    "location": "s3://bucket/warehouse/analytics/user_events",
    "last-updated-ms": 1705312200000,
    "schema": { ... },
    "current-schema-id": 0,
    "schemas": [ ... ],
    "partition-spec": [ ... ],
    "default-spec-id": 0,
    "partition-specs": [ ... ],
    "last-partition-id": 1000,
    "properties": { ... },
    "current-snapshot-id": -1,
    "snapshots": [],
    "sort-orders": [ ... ],
    "default-sort-order-id": 0
  }
}
```

### Example

```bash
curl -X POST http://localhost:8181/v1/namespaces/analytics/tables \
  -H "Content-Type: application/json" \
  -d '{
    "name": "events",
    "schema": {
      "type": "struct",
      "schema-id": 0,
      "fields": [
        {"id": 1, "name": "id", "required": true, "type": "long"},
        {"id": 2, "name": "data", "required": false, "type": "string"}
      ]
    }
  }'
```

---

## Load Table

Load a table's current metadata.

### Request

```http
GET /v1/namespaces/{namespace}/tables/{table}
GET /v1/namespaces/{namespace}/tables/{table}?snapshots={snapshots}
```

### Query Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `snapshots` | string | `all` to include all snapshots, `refs` for referenced only |

### Response

```json
{
  "metadata-location": "s3://bucket/warehouse/analytics/events/metadata/00001-uuid.metadata.json",
  "metadata": {
    "format-version": 2,
    "table-uuid": "550e8400-e29b-41d4-a716-446655440000",
    "location": "s3://bucket/warehouse/analytics/events",
    "last-updated-ms": 1705312200000,
    "schema": {
      "type": "struct",
      "schema-id": 0,
      "fields": [
        {"id": 1, "name": "id", "required": true, "type": "long"},
        {"id": 2, "name": "data", "required": false, "type": "string"}
      ]
    },
    "current-schema-id": 0,
    "partition-spec": [],
    "default-spec-id": 0,
    "properties": {},
    "current-snapshot-id": 123456789,
    "snapshots": [
      {
        "snapshot-id": 123456789,
        "timestamp-ms": 1705312200000,
        "summary": {
          "operation": "append",
          "added-data-files": "10",
          "added-records": "1000"
        },
        "manifest-list": "s3://bucket/.../snap-123456789-uuid.avro"
      }
    ],
    "snapshot-log": [ ... ],
    "metadata-log": [ ... ]
  },
  "config": {
    "client.factory": "org.apache.iceberg.aws.glue.GlueCatalog"
  }
}
```

### Example

```bash
curl http://localhost:8181/v1/namespaces/analytics/tables/events
```

---

## Check Table Exists

Check if a table exists.

### Request

```http
HEAD /v1/namespaces/{namespace}/tables/{table}
```

### Response

- **200 OK**: Table exists
- **404 Not Found**: Table does not exist

### Example

```bash
curl -I http://localhost:8181/v1/namespaces/analytics/tables/events
```

---

## Commit Table Updates

Commit updates to a table (schema changes, new snapshots, property updates).

### Request

```http
POST /v1/namespaces/{namespace}/tables/{table}
```

### Request Body

```json
{
  "identifier": {
    "namespace": ["analytics"],
    "name": "events"
  },
  "requirements": [
    {"type": "assert-current-schema-id", "current-schema-id": 0},
    {"type": "assert-table-uuid", "uuid": "550e8400-e29b-41d4-a716-446655440000"}
  ],
  "updates": [
    {
      "action": "add-schema",
      "schema": {
        "type": "struct",
        "schema-id": 1,
        "fields": [
          {"id": 1, "name": "id", "required": true, "type": "long"},
          {"id": 2, "name": "data", "required": false, "type": "string"},
          {"id": 3, "name": "created_at", "required": true, "type": "timestamptz"}
        ]
      }
    },
    {"action": "set-current-schema", "schema-id": 1},
    {"action": "set-properties", "updates": {"owner": "new-team"}}
  ]
}
```

### Requirements

| Type | Fields | Description |
|------|--------|-------------|
| `assert-create` | - | Assert table doesn't exist |
| `assert-table-uuid` | `uuid` | Assert table UUID matches |
| `assert-ref-snapshot-id` | `ref`, `snapshot-id` | Assert ref points to snapshot |
| `assert-last-assigned-field-id` | `last-assigned-field-id` | Assert last field ID |
| `assert-current-schema-id` | `current-schema-id` | Assert current schema ID |
| `assert-last-assigned-partition-id` | `last-assigned-partition-id` | Assert last partition ID |
| `assert-default-spec-id` | `default-spec-id` | Assert default partition spec |
| `assert-default-sort-order-id` | `default-sort-order-id` | Assert default sort order |

### Update Actions

| Action | Fields | Description |
|--------|--------|-------------|
| `assign-uuid` | `uuid` | Assign table UUID |
| `upgrade-format-version` | `format-version` | Upgrade format version |
| `add-schema` | `schema` | Add new schema |
| `set-current-schema` | `schema-id` | Set current schema |
| `add-partition-spec` | `spec` | Add partition spec |
| `set-default-spec` | `spec-id` | Set default partition spec |
| `add-sort-order` | `sort-order` | Add sort order |
| `set-default-sort-order` | `sort-order-id` | Set default sort order |
| `add-snapshot` | `snapshot` | Add snapshot |
| `set-snapshot-ref` | `ref-name`, `type`, `snapshot-id` | Set snapshot reference |
| `remove-snapshots` | `snapshot-ids` | Remove snapshots |
| `remove-snapshot-ref` | `ref-name` | Remove snapshot reference |
| `set-location` | `location` | Set table location |
| `set-properties` | `updates` | Update properties |
| `remove-properties` | `removals` | Remove properties |

### Response

Returns updated metadata (same format as Load Table).

### Errors

| Code | Error | Description |
|------|-------|-------------|
| 404 | `NoSuchTableException` | Table not found |
| 409 | `CommitFailedException` | Requirements not met |

---

## Drop Table

Delete a table.

### Request

```http
DELETE /v1/namespaces/{namespace}/tables/{table}
DELETE /v1/namespaces/{namespace}/tables/{table}?purgeRequested={true|false}
```

### Query Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `purgeRequested` | boolean | If true, delete data files (default: false) |

### Response

- **204 No Content**: Table deleted

### Example

```bash
# Drop table, keep data files
curl -X DELETE http://localhost:8181/v1/namespaces/analytics/tables/events

# Drop table and purge data
curl -X DELETE "http://localhost:8181/v1/namespaces/analytics/tables/events?purgeRequested=true"
```

---

## Register Table

Register an existing table from a metadata file.

### Request

```http
POST /v1/namespaces/{namespace}/register
```

### Request Body

```json
{
  "name": "imported_table",
  "metadata-location": "s3://bucket/path/to/metadata.json"
}
```

### Example

```bash
curl -X POST http://localhost:8181/v1/namespaces/analytics/register \
  -H "Content-Type: application/json" \
  -d '{
    "name": "imported",
    "metadata-location": "s3://bucket/tables/imported/metadata/00000.metadata.json"
  }'
```

---

## Rename Table

Rename a table or move it to a different namespace.

### Request

```http
POST /v1/tables/rename
```

### Request Body

```json
{
  "source": {
    "namespace": ["analytics"],
    "name": "old_name"
  },
  "destination": {
    "namespace": ["analytics"],
    "name": "new_name"
  }
}
```

### Example

```bash
curl -X POST http://localhost:8181/v1/tables/rename \
  -H "Content-Type: application/json" \
  -d '{
    "source": {"namespace": ["analytics"], "name": "events"},
    "destination": {"namespace": ["analytics"], "name": "user_events"}
  }'
```

---

## Report Metrics

Report table-level metrics (read/write statistics).

### Request

```http
POST /v1/namespaces/{namespace}/tables/{table}/metrics
```

### Request Body

```json
{
  "report-type": "scan",
  "table-name": "analytics.events",
  "snapshot-id": 123456789,
  "filter": "event_time > '2024-01-01'",
  "schema-id": 0,
  "projected-field-ids": [1, 2, 3],
  "projected-field-names": ["id", "data", "created_at"],
  "metrics": {
    "total-planning-duration": 150,
    "total-data-manifests": 10,
    "total-delete-manifests": 2,
    "scanned-data-manifests": 5,
    "skipped-data-manifests": 5,
    "total-file-size-in-bytes": 1073741824,
    "total-data-files": 100,
    "total-delete-files": 10
  }
}
```

### Response

- **204 No Content**: Metrics accepted

---

## Load Credentials

Get temporary credentials for accessing table data.

### Request

```http
GET /v1/namespaces/{namespace}/tables/{table}/credentials
```

### Headers

| Header | Description |
|--------|-------------|
| `X-Iceberg-Access-Delegation` | Credential type (e.g., `vended-credentials`) |

### Response

```json
{
  "config": {
    "s3.access-key-id": "ASIA...",
    "s3.secret-access-key": "...",
    "s3.session-token": "...",
    "s3.region": "us-east-1"
  }
}
```

> [!NOTE]
> Credentials are vended based on storage configuration. Requires appropriate IAM roles and policies.
