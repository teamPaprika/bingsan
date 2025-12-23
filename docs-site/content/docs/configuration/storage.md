---
title: "Storage"
weight: 3
---

# Storage Configuration

Configure the storage backend for Iceberg data files.

## Overview

Bingsan supports three storage backends:

- **S3** - Amazon S3 and compatible services (MinIO, R2)
- **GCS** - Google Cloud Storage
- **Local** - Local filesystem (development only)

## Options

```yaml
storage:
  type: s3
  warehouse: s3://bucket/warehouse

  s3:
    endpoint: ""
    region: us-east-1
    access_key_id: ""
    secret_access_key: ""
    bucket: warehouse
    use_path_style: false

  gcs:
    project: ""
    credentials_file: ""
    bucket: ""

  local:
    root_path: /tmp/iceberg/data
```

## Reference

### General Options

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `type` | string | `local` | Storage type: `s3`, `gcs`, or `local` |
| `warehouse` | string | - | Default warehouse location for new tables |

### S3 Options

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `endpoint` | string | `""` | Custom endpoint (for MinIO, R2, etc.) |
| `region` | string | `us-east-1` | AWS region |
| `access_key_id` | string | `""` | AWS access key (or use IAM role) |
| `secret_access_key` | string | `""` | AWS secret key |
| `bucket` | string | `warehouse` | S3 bucket name |
| `use_path_style` | boolean | `false` | Use path-style URLs (for MinIO) |

### GCS Options

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `project` | string | `""` | GCP project ID |
| `credentials_file` | string | `""` | Path to service account JSON |
| `bucket` | string | `""` | GCS bucket name |

### Local Options

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `root_path` | string | `/tmp/iceberg/data` | Root directory for data files |

## Environment Variables

```bash
ICEBERG_STORAGE_TYPE=s3
ICEBERG_STORAGE_WAREHOUSE=s3://bucket/warehouse

# S3
ICEBERG_STORAGE_S3_ENDPOINT=
ICEBERG_STORAGE_S3_REGION=us-east-1
ICEBERG_STORAGE_S3_ACCESS_KEY_ID=AKIA...
ICEBERG_STORAGE_S3_SECRET_ACCESS_KEY=...
ICEBERG_STORAGE_S3_BUCKET=warehouse
ICEBERG_STORAGE_S3_USE_PATH_STYLE=false

# GCS
ICEBERG_STORAGE_GCS_PROJECT=my-project
ICEBERG_STORAGE_GCS_CREDENTIALS_FILE=/path/to/credentials.json
ICEBERG_STORAGE_GCS_BUCKET=my-bucket

# Local
ICEBERG_STORAGE_LOCAL_ROOT_PATH=/tmp/iceberg/data
```

---

## Amazon S3

### Basic Configuration

```yaml
storage:
  type: s3
  warehouse: s3://my-bucket/warehouse

  s3:
    region: us-east-1
    bucket: my-bucket
```

### With IAM Role (Recommended)

When running on EC2, ECS, EKS, or Lambda with an IAM role attached:

```yaml
storage:
  type: s3
  warehouse: s3://my-bucket/warehouse

  s3:
    region: us-east-1
    bucket: my-bucket
    # No credentials needed - uses instance role
```

### With Access Keys

```yaml
storage:
  type: s3
  warehouse: s3://my-bucket/warehouse

  s3:
    region: us-east-1
    bucket: my-bucket
    access_key_id: "AKIAIOSFODNN7EXAMPLE"
    secret_access_key: "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"
```

### Required IAM Permissions

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "s3:GetObject",
        "s3:PutObject",
        "s3:DeleteObject",
        "s3:ListBucket"
      ],
      "Resource": [
        "arn:aws:s3:::my-bucket",
        "arn:aws:s3:::my-bucket/*"
      ]
    }
  ]
}
```

---

## MinIO

MinIO is S3-compatible and works with the S3 backend:

```yaml
storage:
  type: s3
  warehouse: s3://warehouse/data

  s3:
    endpoint: "http://minio:9000"
    region: us-east-1
    access_key_id: "minioadmin"
    secret_access_key: "minioadmin"
    bucket: warehouse
    use_path_style: true
```

Key differences from S3:
- Set `endpoint` to MinIO address
- Enable `use_path_style: true`

---

## Cloudflare R2

R2 is S3-compatible:

```yaml
storage:
  type: s3
  warehouse: s3://my-r2-bucket/warehouse

  s3:
    endpoint: "https://ACCOUNT_ID.r2.cloudflarestorage.com"
    region: auto
    access_key_id: "YOUR_R2_ACCESS_KEY"
    secret_access_key: "YOUR_R2_SECRET_KEY"
    bucket: my-r2-bucket
    use_path_style: false
```

---

## Google Cloud Storage

### Basic Configuration

```yaml
storage:
  type: gcs
  warehouse: gs://my-bucket/warehouse

  gcs:
    project: my-gcp-project
    bucket: my-bucket
```

### With Service Account

```yaml
storage:
  type: gcs
  warehouse: gs://my-bucket/warehouse

  gcs:
    project: my-gcp-project
    bucket: my-bucket
    credentials_file: /path/to/service-account.json
```

### With Default Credentials

On GCE/GKE with attached service account:

```yaml
storage:
  type: gcs
  warehouse: gs://my-bucket/warehouse

  gcs:
    project: my-gcp-project
    bucket: my-bucket
    # No credentials_file - uses default credentials
```

### Required Permissions

The service account needs these roles:
- `roles/storage.objectAdmin` (or custom role with equivalent permissions)

Specific permissions needed:
- `storage.objects.get`
- `storage.objects.create`
- `storage.objects.delete`
- `storage.objects.list`
- `storage.buckets.get`

---

## Local Storage

> [!WARNING]
> Local storage is intended for development and testing only. Do not use in production.

### Configuration

```yaml
storage:
  type: local
  warehouse: file:///data/warehouse

  local:
    root_path: /data/warehouse
```

### Docker Volume

When using Docker, mount a volume:

```yaml
# docker-compose.yml
services:
  bingsan:
    volumes:
      - ./data:/data/warehouse
```

---

## Warehouse Location

The `warehouse` setting defines the default location for new tables.

### Format

- S3: `s3://bucket/path`
- GCS: `gs://bucket/path`
- Local: `file:///absolute/path`

### Per-Table Locations

Tables can override the warehouse location:

```bash
curl -X POST http://localhost:8181/v1/namespaces/analytics/tables \
  -H "Content-Type: application/json" \
  -d '{
    "name": "events",
    "location": "s3://different-bucket/custom-path/events",
    "schema": {...}
  }'
```

---

## Vended Credentials

When authentication is enabled, Bingsan can vend temporary credentials to clients:

```bash
curl http://localhost:8181/v1/namespaces/analytics/tables/events/credentials \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "X-Iceberg-Access-Delegation: vended-credentials"
```

This requires appropriate IAM policies allowing AssumeRole or token generation.

---

## Best Practices

### S3

- Use IAM roles instead of access keys
- Enable server-side encryption
- Use VPC endpoints for improved security and performance
- Enable versioning for data protection

### GCS

- Use Workload Identity on GKE
- Configure lifecycle policies for old metadata files
- Use regional buckets for lower latency

### Multi-Region

For multi-region deployments:

```yaml
storage:
  type: s3
  warehouse: s3://my-bucket/warehouse

  s3:
    region: us-west-2
    bucket: my-bucket-us-west
```

Configure separate Bingsan instances per region pointing to regional buckets.
