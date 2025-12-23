---
title: "Authentication"
weight: 4
---

# Authentication Configuration

Configure authentication and authorization for the catalog API.

## Options

```yaml
auth:
  enabled: false
  token_expiry: 1h
  signing_key: "change-me-in-production"

  oauth2:
    enabled: false
    issuer: ""
    client_id: ""
    client_secret: ""

  api_key:
    enabled: false
```

## Reference

### General Options

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `enabled` | boolean | `false` | Enable authentication |
| `token_expiry` | duration | `1h` | Access token lifetime |
| `signing_key` | string | - | Secret key for signing tokens |

### OAuth2 Options

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `oauth2.enabled` | boolean | `false` | Enable OAuth2 endpoint |
| `oauth2.issuer` | string | `""` | External OAuth issuer URL |
| `oauth2.client_id` | string | `""` | OAuth client ID |
| `oauth2.client_secret` | string | `""` | OAuth client secret |

### API Key Options

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `api_key.enabled` | boolean | `false` | Enable API key authentication |

## Environment Variables

```bash
ICEBERG_AUTH_ENABLED=true
ICEBERG_AUTH_TOKEN_EXPIRY=1h
ICEBERG_AUTH_SIGNING_KEY=your-256-bit-secret-key
ICEBERG_AUTH_OAUTH2_ENABLED=true
ICEBERG_AUTH_OAUTH2_ISSUER=https://your-idp.example.com
ICEBERG_AUTH_OAUTH2_CLIENT_ID=your-client-id
ICEBERG_AUTH_OAUTH2_CLIENT_SECRET=your-client-secret
ICEBERG_AUTH_API_KEY_ENABLED=true
```

---

## Enabling Authentication

### Basic Setup

```yaml
auth:
  enabled: true
  token_expiry: 1h
  signing_key: "your-secure-256-bit-secret-key-here-change-me"
```

> [!CAUTION]
> Always change the `signing_key` in production. Use a cryptographically secure random string of at least 256 bits (32 characters).

Generate a secure key:

```bash
openssl rand -hex 32
```

---

## OAuth2 Token Exchange

Enable the OAuth2 token endpoint for Iceberg clients:

```yaml
auth:
  enabled: true
  signing_key: "your-secure-key"

  oauth2:
    enabled: true
```

Clients can then exchange credentials for tokens:

```bash
curl -X POST http://localhost:8181/v1/oauth/tokens \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "grant_type=client_credentials" \
  -d "client_id=my-client" \
  -d "client_secret=my-secret"
```

Response:

```json
{
  "access_token": "eyJhbGciOiJIUzI1NiIs...",
  "token_type": "bearer",
  "expires_in": 3600
}
```

---

## External OAuth Provider

Use an external OAuth/OIDC provider:

```yaml
auth:
  enabled: true

  oauth2:
    enabled: true
    issuer: "https://your-idp.example.com"
    # client_id/client_secret optional for token validation
```

Bingsan will validate tokens against the issuer's JWKS endpoint.

### Supported Providers

- **Auth0**: `issuer: "https://your-tenant.auth0.com/"`
- **Okta**: `issuer: "https://your-org.okta.com"`
- **Keycloak**: `issuer: "https://keycloak.example.com/realms/your-realm"`
- **Azure AD**: `issuer: "https://login.microsoftonline.com/your-tenant/v2.0"`
- **Google**: `issuer: "https://accounts.google.com"`

---

## API Key Authentication

Enable API key authentication:

```yaml
auth:
  enabled: true

  api_key:
    enabled: true
```

Use API keys in requests:

```bash
curl -H "Authorization: Bearer YOUR_API_KEY" \
  http://localhost:8181/v1/namespaces
```

---

## Client Configuration

### Apache Spark

```properties
spark.sql.catalog.bingsan=org.apache.iceberg.spark.SparkCatalog
spark.sql.catalog.bingsan.type=rest
spark.sql.catalog.bingsan.uri=http://localhost:8181
spark.sql.catalog.bingsan.credential=client_id:client_secret
```

### PyIceberg

```python
from pyiceberg.catalog import load_catalog

catalog = load_catalog(
    "rest",
    uri="http://localhost:8181",
    credential="client_id:client_secret"
)
```

### Trino

```properties
connector.name=iceberg
iceberg.catalog.type=rest
iceberg.rest-catalog.uri=http://localhost:8181
iceberg.rest-catalog.security=OAUTH2
iceberg.rest-catalog.oauth2.client-id=client_id
iceberg.rest-catalog.oauth2.client-secret=client_secret
```

### cURL

```bash
# Get token
TOKEN=$(curl -s -X POST http://localhost:8181/v1/oauth/tokens \
  -d "grant_type=client_credentials" \
  -d "client_id=my-client" \
  -d "client_secret=my-secret" | jq -r '.access_token')

# Use token
curl -H "Authorization: Bearer $TOKEN" \
  http://localhost:8181/v1/namespaces
```

---

## Token Expiry

Configure token lifetime:

```yaml
auth:
  token_expiry: 1h   # 1 hour (default)
  # or
  token_expiry: 30m  # 30 minutes
  # or
  token_expiry: 24h  # 24 hours
```

### Considerations

- **Shorter expiry** (15m-1h): More secure, more token refreshes
- **Longer expiry** (24h+): Fewer refreshes, larger window if token compromised

---

## Endpoints Without Authentication

These endpoints never require authentication:

| Endpoint | Description |
|----------|-------------|
| `GET /health` | Health check |
| `GET /ready` | Readiness check |
| `GET /metrics` | Prometheus metrics |
| `GET /v1/config` | Catalog configuration |
| `POST /v1/oauth/tokens` | Token exchange |

---

## Security Best Practices

### Signing Key

- Use at least 256 bits of entropy
- Store in a secure secret store (Kubernetes Secrets, HashiCorp Vault)
- Rotate periodically

```bash
# Generate secure key
openssl rand -hex 32

# Or using /dev/urandom
head -c 32 /dev/urandom | xxd -p -c 64
```

### HTTPS

Always use HTTPS in production:

```yaml
# Place Bingsan behind a TLS-terminating proxy/load balancer
server:
  host: 0.0.0.0
  port: 8181
```

### Network Security

- Use network policies to restrict access
- Place behind a firewall
- Use VPC endpoints for cloud services

---

## Troubleshooting

### Invalid Token

```
Error: invalid or expired token
```

- Check token hasn't expired
- Verify signing key matches
- Ensure token is properly formatted

### Authentication Required

```
Error: authentication required
```

- Include `Authorization: Bearer <token>` header
- Verify authentication is properly configured

### Invalid Credentials

```
Error: invalid client credentials
```

- Check client_id and client_secret
- Verify credentials are URL-encoded if using form data
