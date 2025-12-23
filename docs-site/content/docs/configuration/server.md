---
title: "Server"
weight: 1
---

# Server Configuration

Configure the HTTP server settings.

## Options

```yaml
server:
  host: 0.0.0.0
  port: 8181
  debug: false
  version: "0.1.0"
  read_timeout: 30s
  write_timeout: 30s
  idle_timeout: 120s
```

## Reference

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `host` | string | `0.0.0.0` | IP address to bind to |
| `port` | integer | `8181` | Port to listen on |
| `debug` | boolean | `false` | Enable debug mode (verbose logging, stack traces) |
| `version` | string | `0.1.0` | Server version reported in responses |
| `read_timeout` | duration | `30s` | Maximum time to read request |
| `write_timeout` | duration | `30s` | Maximum time to write response |
| `idle_timeout` | duration | `120s` | Maximum time for idle connections |

## Environment Variables

```bash
ICEBERG_SERVER_HOST=0.0.0.0
ICEBERG_SERVER_PORT=8181
ICEBERG_SERVER_DEBUG=false
ICEBERG_SERVER_READ_TIMEOUT=30s
ICEBERG_SERVER_WRITE_TIMEOUT=30s
ICEBERG_SERVER_IDLE_TIMEOUT=120s
```

## Binding Address

### All Interfaces

Bind to all network interfaces (default):

```yaml
server:
  host: 0.0.0.0
```

### Localhost Only

Bind only to localhost (more secure for development):

```yaml
server:
  host: 127.0.0.1
```

### Specific Interface

Bind to a specific network interface:

```yaml
server:
  host: 192.168.1.100
```

## Timeouts

### Read Timeout

Maximum duration for reading the entire request, including the body.

```yaml
server:
  read_timeout: 30s
```

Increase for large request bodies:

```yaml
server:
  read_timeout: 5m  # For large metadata commits
```

### Write Timeout

Maximum duration before timing out writes of the response.

```yaml
server:
  write_timeout: 30s
```

Increase for large responses:

```yaml
server:
  write_timeout: 5m  # For large table metadata
```

### Idle Timeout

Maximum time to wait for the next request when keep-alives are enabled.

```yaml
server:
  idle_timeout: 120s
```

## Debug Mode

Enable debug mode for troubleshooting:

```yaml
server:
  debug: true
```

In debug mode:
- Stack traces are included in error responses
- More verbose logging is enabled
- Performance may be reduced

> [!WARNING]
> Do not enable debug mode in production.

## Port Selection

### Standard Port

The Iceberg REST Catalog standard port is `8181`:

```yaml
server:
  port: 8181
```

### Alternative Ports

Choose an alternative port to avoid conflicts:

```yaml
server:
  port: 8080  # Common HTTP alternative
  # or
  port: 9181  # Shifted port
```

### Privileged Ports

To use ports below 1024 (like 80 or 443), the process must run as root or have the `CAP_NET_BIND_SERVICE` capability.

## Production Recommendations

```yaml
server:
  host: 0.0.0.0
  port: 8181
  debug: false
  read_timeout: 60s
  write_timeout: 60s
  idle_timeout: 300s
```

For high-throughput environments:

```yaml
server:
  host: 0.0.0.0
  port: 8181
  debug: false
  read_timeout: 30s
  write_timeout: 30s
  idle_timeout: 60s  # Lower to free connections faster
```
