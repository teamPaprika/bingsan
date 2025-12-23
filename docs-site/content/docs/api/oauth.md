---
title: "OAuth"
weight: 9
---

# OAuth API

Bingsan supports OAuth2 token exchange for client authentication, compatible with the Iceberg REST Catalog specification.

## Token Exchange

Exchange credentials for an access token.

### Request

```http
POST /v1/oauth/tokens
Content-Type: application/x-www-form-urlencoded
```

### Form Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `grant_type` | string | Yes | Must be `client_credentials` |
| `client_id` | string | Yes | Client identifier |
| `client_secret` | string | Yes | Client secret |
| `scope` | string | No | Requested scope (e.g., `catalog`) |

### Response

```json
{
  "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "token_type": "bearer",
  "expires_in": 3600,
  "scope": "catalog"
}
```

### Response Fields

| Field | Type | Description |
|-------|------|-------------|
| `access_token` | string | Bearer token for API requests |
| `token_type` | string | Always `bearer` |
| `expires_in` | integer | Token lifetime in seconds |
| `scope` | string | Granted scope |

### Errors

| Code | Error | Description |
|------|-------|-------------|
| 400 | `invalid_request` | Missing required parameters |
| 401 | `invalid_client` | Invalid client credentials |
| 403 | `access_denied` | Client not authorized |

### Example

```bash
curl -X POST http://localhost:8181/v1/oauth/tokens \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "grant_type=client_credentials" \
  -d "client_id=my-client" \
  -d "client_secret=my-secret" \
  -d "scope=catalog"
```

---

## Using Access Tokens

Include the access token in API requests:

```bash
curl -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..." \
  http://localhost:8181/v1/namespaces
```

---

## Token Refresh

When a token expires, request a new one using the same credentials. Bingsan does not currently support refresh tokens.

---

## Client Configuration

### Spark

```properties
spark.sql.catalog.my_catalog=org.apache.iceberg.spark.SparkCatalog
spark.sql.catalog.my_catalog.type=rest
spark.sql.catalog.my_catalog.uri=http://localhost:8181
spark.sql.catalog.my_catalog.credential=client_id:client_secret
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

---

## Configuration

Enable OAuth in `config.yaml`:

```yaml
auth:
  enabled: true
  token_expiry: 1h
  signing_key: "your-secure-signing-key-change-in-production"

  oauth2:
    enabled: true
    # Optional: Use external OAuth provider
    issuer: ""
    client_id: ""
    client_secret: ""
```

### Environment Variables

```bash
ICEBERG_AUTH_ENABLED=true
ICEBERG_AUTH_TOKEN_EXPIRY=1h
ICEBERG_AUTH_SIGNING_KEY=your-secure-signing-key
ICEBERG_AUTH_OAUTH2_ENABLED=true
```

---

## Security Considerations

### Signing Key

- Use a strong, random signing key (at least 256 bits)
- Store the key securely (e.g., Kubernetes secrets, vault)
- Rotate keys periodically

### Token Expiry

- Default: 1 hour
- Shorter expiry improves security but increases token exchange frequency
- Consider your client's tolerance for re-authentication

### HTTPS

Always use HTTPS in production to protect credentials and tokens in transit.

```yaml
# Behind a TLS-terminating proxy
server:
  host: 0.0.0.0
  port: 8181
```

---

## External OAuth Providers

Bingsan can validate tokens from external OAuth providers (OIDC):

```yaml
auth:
  enabled: true
  oauth2:
    enabled: true
    issuer: "https://your-idp.example.com"
    # Leave client_id/client_secret empty when using external issuer
```

Tokens from the external provider are validated against the issuer's JWKS endpoint.

### Supported Providers

- Auth0
- Okta
- Keycloak
- Azure AD
- Google Identity Platform
- Any OIDC-compliant provider
