---
title: "Database"
weight: 2
---

# Database Configuration

Configure the PostgreSQL connection for metadata storage.

## Options

```yaml
database:
  host: localhost
  port: 5432
  user: iceberg
  password: iceberg
  database: iceberg_catalog
  ssl_mode: disable
  max_open_conns: 25
  max_idle_conns: 5
  conn_max_lifetime: 5m
  conn_max_idle_time: 5m
```

## Reference

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `host` | string | `localhost` | PostgreSQL server hostname |
| `port` | integer | `5432` | PostgreSQL server port |
| `user` | string | `iceberg` | Database user |
| `password` | string | `iceberg` | Database password |
| `database` | string | `iceberg_catalog` | Database name |
| `ssl_mode` | string | `disable` | SSL mode (`disable`, `require`, `verify-ca`, `verify-full`) |
| `max_open_conns` | integer | `25` | Maximum open connections |
| `max_idle_conns` | integer | `5` | Maximum idle connections |
| `conn_max_lifetime` | duration | `5m` | Maximum connection lifetime |
| `conn_max_idle_time` | duration | `5m` | Maximum connection idle time |

## Environment Variables

```bash
ICEBERG_DATABASE_HOST=localhost
ICEBERG_DATABASE_PORT=5432
ICEBERG_DATABASE_USER=iceberg
ICEBERG_DATABASE_PASSWORD=iceberg
ICEBERG_DATABASE_DATABASE=iceberg_catalog
ICEBERG_DATABASE_SSL_MODE=disable
ICEBERG_DATABASE_MAX_OPEN_CONNS=25
ICEBERG_DATABASE_MAX_IDLE_CONNS=5
ICEBERG_DATABASE_CONN_MAX_LIFETIME=5m
ICEBERG_DATABASE_CONN_MAX_IDLE_TIME=5m
```

## Connection URL

Bingsan constructs a connection string from these settings:

```
postgresql://user:password@host:port/database?sslmode=ssl_mode
```

## SSL Modes

| Mode | Description |
|------|-------------|
| `disable` | No SSL (development only) |
| `require` | Use SSL but don't verify certificate |
| `verify-ca` | Verify server certificate against CA |
| `verify-full` | Verify certificate and hostname |

### Production SSL Configuration

```yaml
database:
  host: postgres.example.com
  ssl_mode: verify-full
```

For AWS RDS with SSL:

```yaml
database:
  host: mydb.us-east-1.rds.amazonaws.com
  ssl_mode: require
```

## Connection Pool Tuning

### Default Settings

Suitable for most workloads:

```yaml
database:
  max_open_conns: 25
  max_idle_conns: 5
  conn_max_lifetime: 5m
  conn_max_idle_time: 5m
```

### High-Throughput Settings

For high request rates:

```yaml
database:
  max_open_conns: 100
  max_idle_conns: 25
  conn_max_lifetime: 30m
  conn_max_idle_time: 10m
```

### Resource-Constrained Settings

For limited resources:

```yaml
database:
  max_open_conns: 10
  max_idle_conns: 2
  conn_max_lifetime: 5m
  conn_max_idle_time: 1m
```

## Connection Pool Sizing

### General Formula

```
max_open_conns = (2 * num_cpus) + effective_spindle_count
```

For SSDs, `effective_spindle_count` is typically 1.

### Considerations

- **Too Few Connections**: Requests queue, increasing latency
- **Too Many Connections**: Overloads PostgreSQL, causes memory issues

### PostgreSQL max_connections

Ensure PostgreSQL's `max_connections` is higher than `max_open_conns` multiplied by the number of Bingsan instances:

```sql
SHOW max_connections;  -- Default is 100
```

Increase if needed:

```sql
ALTER SYSTEM SET max_connections = 200;
-- Requires restart
```

## Connection Lifetime

### conn_max_lifetime

Maximum time a connection can be reused. Set lower than PostgreSQL's `idle_in_transaction_session_timeout`:

```yaml
database:
  conn_max_lifetime: 5m
```

### conn_max_idle_time

Maximum time a connection can be idle before being closed:

```yaml
database:
  conn_max_idle_time: 5m
```

## Database Setup

### Create Database

```sql
CREATE DATABASE iceberg_catalog;
CREATE USER iceberg WITH PASSWORD 'your-secure-password';
GRANT ALL PRIVILEGES ON DATABASE iceberg_catalog TO iceberg;
```

### Required Extensions

Bingsan uses these PostgreSQL extensions (auto-created if permissions allow):

```sql
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
```

### Migrations

Bingsan automatically runs database migrations on startup. No manual schema setup is required.

## Multiple Instances

When running multiple Bingsan instances against the same database:

```yaml
database:
  max_open_conns: 25  # Per instance
```

Total connections = `max_open_conns * num_instances`

Ensure PostgreSQL can handle the total:

```sql
ALTER SYSTEM SET max_connections = 500;  -- For 20 instances
```

## Monitoring

Monitor connection pool health via Prometheus metrics:

```promql
iceberg_db_connections_open
iceberg_db_connections_in_use
iceberg_db_connections_idle
iceberg_db_wait_count_total
```

### Connection Pool Saturation Alert

```yaml
- alert: DatabasePoolSaturation
  expr: iceberg_db_connections_in_use / iceberg_db_connections_open > 0.9
  for: 5m
  labels:
    severity: warning
```

## Troubleshooting

### Connection Refused

```
Error: connection refused
```

- Verify PostgreSQL is running
- Check host and port settings
- Ensure network connectivity

### Authentication Failed

```
Error: password authentication failed
```

- Verify user and password
- Check PostgreSQL's `pg_hba.conf` for allowed connections

### Too Many Connections

```
Error: too many connections for role "iceberg"
```

- Reduce `max_open_conns`
- Increase PostgreSQL's `max_connections`
- Check for connection leaks in other applications

### SSL Certificate Error

```
Error: certificate verify failed
```

- For development: use `ssl_mode: disable`
- For production: ensure correct SSL certificates are installed
