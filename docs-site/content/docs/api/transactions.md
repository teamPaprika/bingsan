---
title: "Transactions"
weight: 6
---

# Transactions API

The transactions API enables atomic multi-table commits, ensuring all-or-nothing semantics across multiple table updates.

## Overview

Multi-table transactions are useful for:

- Maintaining referential integrity across tables
- Coordinated schema changes
- Atomic data pipelines that write to multiple tables
- ETL workflows with multiple output tables

## Commit Transaction

Atomically commit updates to multiple tables.

### Request

```http
POST /v1/transactions/commit
```

### Request Body

```json
{
  "table-changes": [
    {
      "identifier": {
        "namespace": ["analytics"],
        "name": "events"
      },
      "requirements": [
        {"type": "assert-table-uuid", "uuid": "550e8400-e29b-41d4-a716-446655440000"},
        {"type": "assert-current-schema-id", "current-schema-id": 0}
      ],
      "updates": [
        {
          "action": "add-snapshot",
          "snapshot": {
            "snapshot-id": 123456790,
            "parent-snapshot-id": 123456789,
            "timestamp-ms": 1705398600000,
            "summary": {
              "operation": "append",
              "added-data-files": "5",
              "added-records": "10000"
            },
            "manifest-list": "s3://bucket/metadata/snap-123456790.avro",
            "schema-id": 0
          }
        },
        {"action": "set-snapshot-ref", "ref-name": "main", "type": "branch", "snapshot-id": 123456790}
      ]
    },
    {
      "identifier": {
        "namespace": ["analytics"],
        "name": "event_counts"
      },
      "requirements": [
        {"type": "assert-table-uuid", "uuid": "660e8400-e29b-41d4-a716-446655440001"}
      ],
      "updates": [
        {
          "action": "add-snapshot",
          "snapshot": {
            "snapshot-id": 987654322,
            "parent-snapshot-id": 987654321,
            "timestamp-ms": 1705398600000,
            "summary": {
              "operation": "overwrite",
              "added-data-files": "1",
              "added-records": "100"
            },
            "manifest-list": "s3://bucket/metadata/snap-987654322.avro",
            "schema-id": 0
          }
        },
        {"action": "set-snapshot-ref", "ref-name": "main", "type": "branch", "snapshot-id": 987654322}
      ]
    }
  ]
}
```

### Request Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `table-changes` | array | Yes | List of table updates to commit atomically |

Each table change contains:

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `identifier` | object | Yes | Table identifier (namespace + name) |
| `requirements` | array | Yes | Preconditions that must be met |
| `updates` | array | Yes | Updates to apply |

### Requirements

All requirements from the [Tables API]({{< relref "/docs/api/tables" >}}) are supported:

| Type | Fields | Description |
|------|--------|-------------|
| `assert-create` | - | Table must not exist |
| `assert-table-uuid` | `uuid` | UUID must match |
| `assert-ref-snapshot-id` | `ref`, `snapshot-id` | Ref must point to snapshot |
| `assert-last-assigned-field-id` | `last-assigned-field-id` | Last field ID must match |
| `assert-current-schema-id` | `current-schema-id` | Schema ID must match |
| `assert-last-assigned-partition-id` | `last-assigned-partition-id` | Partition ID must match |
| `assert-default-spec-id` | `default-spec-id` | Default spec must match |
| `assert-default-sort-order-id` | `default-sort-order-id` | Sort order must match |

### Updates

All update actions from the [Tables API]({{< relref "/docs/api/tables" >}}) are supported:

| Action | Description |
|--------|-------------|
| `assign-uuid` | Assign table UUID |
| `upgrade-format-version` | Upgrade format version |
| `add-schema` | Add new schema |
| `set-current-schema` | Set current schema |
| `add-partition-spec` | Add partition spec |
| `set-default-spec` | Set default partition spec |
| `add-sort-order` | Add sort order |
| `set-default-sort-order` | Set default sort order |
| `add-snapshot` | Add snapshot |
| `set-snapshot-ref` | Set snapshot reference |
| `remove-snapshots` | Remove snapshots |
| `remove-snapshot-ref` | Remove snapshot reference |
| `set-location` | Set table location |
| `set-properties` | Update properties |
| `remove-properties` | Remove properties |

### Response (Success)

```json
{
  "commit-id": "txn-770e8400-e29b-41d4-a716-446655440002",
  "results": [
    {
      "identifier": {"namespace": ["analytics"], "name": "events"},
      "metadata-location": "s3://bucket/events/metadata/00002.metadata.json"
    },
    {
      "identifier": {"namespace": ["analytics"], "name": "event_counts"},
      "metadata-location": "s3://bucket/event_counts/metadata/00003.metadata.json"
    }
  ]
}
```

### Response (Partial Failure)

If any requirement fails, the entire transaction is rolled back:

```json
{
  "error": {
    "message": "Transaction failed: requirement not met for table analytics.events",
    "type": "CommitFailedException",
    "code": 409,
    "failed-requirements": [
      {
        "identifier": {"namespace": ["analytics"], "name": "events"},
        "requirement": {"type": "assert-current-schema-id", "current-schema-id": 0},
        "actual": {"current-schema-id": 1}
      }
    ]
  }
}
```

### Errors

| Code | Error | Description |
|------|-------|-------------|
| 400 | `BadRequestException` | Invalid request format |
| 404 | `NoSuchTableException` | One or more tables not found |
| 409 | `CommitFailedException` | One or more requirements not met |
| 500 | `ServerError` | Internal transaction error |

### Example

```bash
curl -X POST http://localhost:8181/v1/transactions/commit \
  -H "Content-Type: application/json" \
  -d '{
    "table-changes": [
      {
        "identifier": {"namespace": ["analytics"], "name": "events"},
        "requirements": [
          {"type": "assert-table-uuid", "uuid": "550e8400-e29b-41d4-a716-446655440000"}
        ],
        "updates": [
          {"action": "set-properties", "updates": {"last-modified-by": "etl-pipeline"}}
        ]
      },
      {
        "identifier": {"namespace": ["analytics"], "name": "aggregates"},
        "requirements": [
          {"type": "assert-table-uuid", "uuid": "660e8400-e29b-41d4-a716-446655440001"}
        ],
        "updates": [
          {"action": "set-properties", "updates": {"last-modified-by": "etl-pipeline"}}
        ]
      }
    ]
  }'
```

---

## Transaction Semantics

### Atomicity

All table changes in a transaction either succeed together or fail together. If any requirement check fails, no changes are applied to any table.

### Isolation

Transactions use optimistic concurrency control:

1. Requirements are checked for all tables
2. If all pass, updates are applied
3. If any concurrent modification occurred, the transaction fails

### Ordering

Within a transaction:
- Requirements are checked in order
- Updates are applied in order
- All updates complete before the transaction commits

### Limitations

- Maximum 100 tables per transaction
- Maximum 1000 updates per table
- Transaction timeout: 30 seconds (configurable)

---

## Use Cases

### Coordinated Data Pipeline

Write to a fact table and update a summary table atomically:

```json
{
  "table-changes": [
    {
      "identifier": {"namespace": ["warehouse"], "name": "sales_facts"},
      "requirements": [{"type": "assert-table-uuid", "uuid": "..."}],
      "updates": [
        {"action": "add-snapshot", "snapshot": {...}},
        {"action": "set-snapshot-ref", "ref-name": "main", "type": "branch", "snapshot-id": 123}
      ]
    },
    {
      "identifier": {"namespace": ["warehouse"], "name": "daily_sales_summary"},
      "requirements": [{"type": "assert-table-uuid", "uuid": "..."}],
      "updates": [
        {"action": "add-snapshot", "snapshot": {...}},
        {"action": "set-snapshot-ref", "ref-name": "main", "type": "branch", "snapshot-id": 456}
      ]
    }
  ]
}
```

### Schema Migration

Update schemas across related tables:

```json
{
  "table-changes": [
    {
      "identifier": {"namespace": ["app"], "name": "users"},
      "requirements": [{"type": "assert-current-schema-id", "current-schema-id": 0}],
      "updates": [
        {"action": "add-schema", "schema": {...}},
        {"action": "set-current-schema", "schema-id": 1}
      ]
    },
    {
      "identifier": {"namespace": ["app"], "name": "user_profiles"},
      "requirements": [{"type": "assert-current-schema-id", "current-schema-id": 0}],
      "updates": [
        {"action": "add-schema", "schema": {...}},
        {"action": "set-current-schema", "schema-id": 1}
      ]
    }
  ]
}
```

### Bulk Property Update

Update properties across multiple tables:

```json
{
  "table-changes": [
    {
      "identifier": {"namespace": ["team_a"], "name": "table1"},
      "requirements": [],
      "updates": [{"action": "set-properties", "updates": {"owner": "new-team"}}]
    },
    {
      "identifier": {"namespace": ["team_a"], "name": "table2"},
      "requirements": [],
      "updates": [{"action": "set-properties", "updates": {"owner": "new-team"}}]
    }
  ]
}
```
