---
title: "Scalability"
weight: 3
---

# Scalability

Bingsan is designed to scale horizontally while maintaining consistency.

## Horizontal Scaling

### Stateless Architecture

Each Bingsan instance is stateless:

- All state stored in PostgreSQL
- No inter-node communication required
- Any node can handle any request
- Simple load balancing (round-robin works fine)

```
┌──────────────────────────────────────────┐
│              Load Balancer               │
└────────────────────┬─────────────────────┘
                     │
     ┌───────────────┼───────────────┐
     ▼               ▼               ▼
┌─────────┐   ┌─────────┐   ┌─────────┐
│ Node 1  │   │ Node 2  │   │ Node N  │
└────┬────┘   └────┬────┘   └────┬────┘
     │             │             │
     └─────────────┼─────────────┘
                   ▼
          ┌───────────────┐
          │  PostgreSQL   │
          └───────────────┘
```

### Kubernetes Deployment

Scale with Kubernetes HPA:

```yaml
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: bingsan
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: bingsan
  minReplicas: 3
  maxReplicas: 50
  metrics:
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: 70
```

---

## PostgreSQL Scaling

### Connection Pool Sizing

Total connections across all instances:

```
total_connections = max_open_conns × num_instances
```

Example: 25 connections × 10 instances = 250 connections

Ensure PostgreSQL can handle the load:

```sql
ALTER SYSTEM SET max_connections = 500;
```

### Read Replicas

For read-heavy workloads, use PostgreSQL read replicas:

```yaml
database:
  host: primary.postgres.internal      # Writes
  read_host: replica.postgres.internal # Reads (future feature)
```

### Connection Pooling (PgBouncer)

For many instances, use PgBouncer:

```
┌─────────┐  ┌─────────┐  ┌─────────┐
│ Node 1  │  │ Node 2  │  │ Node N  │
└────┬────┘  └────┬────┘  └────┬────┘
     │            │            │
     └────────────┼────────────┘
                  ▼
          ┌─────────────┐
          │  PgBouncer  │
          └──────┬──────┘
                 ▼
          ┌───────────────┐
          │  PostgreSQL   │
          └───────────────┘
```

Configure PgBouncer for transaction pooling:

```ini
[databases]
iceberg_catalog = host=postgres port=5432 dbname=iceberg_catalog

[pgbouncer]
pool_mode = transaction
max_client_conn = 1000
default_pool_size = 50
```

---

## Performance Characteristics

### Latency

Typical operation latencies:

| Operation | p50 | p99 |
|-----------|-----|-----|
| List namespaces | 2ms | 10ms |
| Get namespace | 1ms | 5ms |
| List tables | 3ms | 15ms |
| Load table | 5ms | 25ms |
| Create table | 20ms | 100ms |
| Commit table | 30ms | 150ms |

### Throughput

Single node capacity (depends on hardware):

- **Reads**: 5,000-10,000 requests/second
- **Writes**: 500-2,000 requests/second

Scale linearly by adding nodes.

### Resource Usage

Per instance:

| Resource | Typical | Peak |
|----------|---------|------|
| Memory | 50-100 MB | 200 MB |
| CPU | 0.2 cores | 1 core |
| Goroutines | 100-500 | 2,000 |

---

## Bottlenecks and Solutions

### PostgreSQL Connections

**Symptom**: `too many connections` errors

**Solutions**:
- Use PgBouncer
- Reduce `max_open_conns` per instance
- Increase PostgreSQL `max_connections`

### Lock Contention

**Symptom**: High commit latency, timeout errors

**Solutions**:
- Increase `lock_timeout`
- Reduce write frequency to same table
- Partition workloads across tables

### Network Latency

**Symptom**: High latency despite low server load

**Solutions**:
- Deploy closer to PostgreSQL
- Use connection pooling
- Enable keep-alives

---

## Capacity Planning

### Estimating Instances

```
instances = (peak_requests_per_second / requests_per_instance) × 1.5
```

Example: 10,000 RPS with 5,000 RPS/instance = 3 instances × 1.5 = 5 instances

### Database Sizing

Metadata size per table: ~10-50 KB (varies with schema complexity)

```
database_size ≈ num_tables × 30 KB + num_namespaces × 1 KB
```

10,000 tables ≈ 300 MB database

### Memory

```
memory_per_instance = base_memory + (concurrent_requests × request_memory)
                   ≈ 50 MB + (500 × 100 KB)
                   ≈ 100 MB
```

---

## High Availability

### Multiple Instances

Run at least 3 instances for HA:

```yaml
spec:
  replicas: 3
  strategy:
    type: RollingUpdate
    rollingUpdate:
      maxUnavailable: 1
      maxSurge: 1
```

### Pod Disruption Budget

```yaml
apiVersion: policy/v1
kind: PodDisruptionBudget
metadata:
  name: bingsan
spec:
  minAvailable: 2
  selector:
    matchLabels:
      app: bingsan
```

### PostgreSQL HA

Use managed PostgreSQL with automatic failover:
- AWS RDS Multi-AZ
- GCP Cloud SQL HA
- Azure Database for PostgreSQL

Or deploy with Patroni/Stolon for self-managed HA.

---

## Multi-Region

### Active-Passive

Single primary region, standby in secondary:

```
Region A (Active)              Region B (Passive)
┌─────────┐                   ┌─────────┐
│ Bingsan │                   │ Bingsan │ (standby)
└────┬────┘                   └────┬────┘
     │                             │
┌────▼────┐                   ┌────▼────┐
│ Primary │  ──replication──▶ │ Replica │
│   DB    │                   │   DB    │
└─────────┘                   └─────────┘
```

### Active-Active (Sharded)

Partition by namespace prefix:

```
analytics.* → Region A
raw.*       → Region B
staging.*   → Region C
```

Each region has its own database and Bingsan cluster.

---

## Monitoring for Scale

### Key Metrics

```promql
# Request rate per instance
sum(rate(iceberg_catalog_http_requests_total[5m])) by (instance)

# Database connection utilization
iceberg_db_connections_in_use / iceberg_db_connections_max

# Lock wait time
rate(iceberg_db_wait_duration_seconds_total[5m])

# Request queue depth
iceberg_catalog_http_requests_in_flight
```

### Scaling Triggers

Auto-scale based on:
- CPU > 70%
- Request latency p99 > 100ms
- Request queue > 100

### Capacity Alerts

```yaml
- alert: BingsanApproachingCapacity
  expr: |
    sum(rate(iceberg_catalog_http_requests_total[5m]))
    / (count(up{job="bingsan"}) * 5000) > 0.8
  labels:
    severity: warning
  annotations:
    summary: "Bingsan cluster approaching capacity"
```
