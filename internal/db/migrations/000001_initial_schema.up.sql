-- Iceberg REST Catalog Initial Schema
-- Supports multi-node deployments with distributed locking

-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- Namespaces (hierarchical, like database.schema)
CREATE TABLE namespaces (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT[] NOT NULL UNIQUE,  -- Hierarchical namespace as array
    properties JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Tables (Iceberg tables)
CREATE TABLE tables (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    namespace_id UUID NOT NULL REFERENCES namespaces(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    metadata_location TEXT NOT NULL,
    metadata JSONB NOT NULL,  -- Full TableMetadata as JSON
    previous_metadata_location TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(namespace_id, name)
);

-- Views (Iceberg views)
CREATE TABLE views (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    namespace_id UUID NOT NULL REFERENCES namespaces(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    metadata_location TEXT NOT NULL,
    metadata JSONB NOT NULL,  -- Full ViewMetadata as JSON
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(namespace_id, name)
);

-- Distributed Locking for multi-node deployments
CREATE TABLE catalog_locks (
    lock_key TEXT PRIMARY KEY,
    owner_id TEXT NOT NULL,
    acquired_at TIMESTAMPTZ DEFAULT NOW(),
    expires_at TIMESTAMPTZ NOT NULL
);

-- Commit Log for optimistic concurrency control
CREATE TABLE commit_log (
    id BIGSERIAL PRIMARY KEY,
    table_id UUID REFERENCES tables(id) ON DELETE CASCADE,
    metadata_location TEXT NOT NULL,
    committed_at TIMESTAMPTZ DEFAULT NOW()
);

-- API Keys for authentication
CREATE TABLE api_keys (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    key_hash TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    scopes TEXT[] DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    expires_at TIMESTAMPTZ,
    last_used_at TIMESTAMPTZ
);

-- OAuth2 Tokens for server-side token management
CREATE TABLE oauth_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    access_token_hash TEXT NOT NULL UNIQUE,
    refresh_token_hash TEXT,
    client_id TEXT NOT NULL,
    scopes TEXT[] DEFAULT '{}',
    issued_at TIMESTAMPTZ DEFAULT NOW(),
    expires_at TIMESTAMPTZ NOT NULL
);

-- Scan Plans (for server-side scan planning)
CREATE TABLE scan_plans (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    table_id UUID NOT NULL REFERENCES tables(id) ON DELETE CASCADE,
    plan_data JSONB NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending', -- pending, running, completed, cancelled
    created_at TIMESTAMPTZ DEFAULT NOW(),
    expires_at TIMESTAMPTZ NOT NULL
);

-- Indexes for performance
CREATE INDEX idx_namespaces_name ON namespaces USING GIN(name);
CREATE INDEX idx_tables_namespace ON tables(namespace_id);
CREATE INDEX idx_tables_name ON tables(name);
CREATE INDEX idx_views_namespace ON views(namespace_id);
CREATE INDEX idx_views_name ON views(name);
CREATE INDEX idx_locks_expires ON catalog_locks(expires_at);
CREATE INDEX idx_commit_log_table ON commit_log(table_id, committed_at DESC);
CREATE INDEX idx_scan_plans_table ON scan_plans(table_id);
CREATE INDEX idx_scan_plans_status ON scan_plans(status) WHERE status IN ('pending', 'running');

-- Trigger to update updated_at automatically
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_namespaces_updated_at
    BEFORE UPDATE ON namespaces
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_tables_updated_at
    BEFORE UPDATE ON tables
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_views_updated_at
    BEFORE UPDATE ON views
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Function to clean up expired locks
CREATE OR REPLACE FUNCTION cleanup_expired_locks()
RETURNS INTEGER AS $$
DECLARE
    deleted_count INTEGER;
BEGIN
    DELETE FROM catalog_locks WHERE expires_at < NOW();
    GET DIAGNOSTICS deleted_count = ROW_COUNT;
    RETURN deleted_count;
END;
$$ LANGUAGE plpgsql;

-- Function to acquire a lock with retry
CREATE OR REPLACE FUNCTION try_acquire_lock(
    p_lock_key TEXT,
    p_owner_id TEXT,
    p_ttl_seconds INTEGER
)
RETURNS BOOLEAN AS $$
DECLARE
    acquired BOOLEAN := FALSE;
BEGIN
    -- Clean up expired locks first
    DELETE FROM catalog_locks WHERE lock_key = p_lock_key AND expires_at < NOW();

    -- Try to insert a new lock
    INSERT INTO catalog_locks (lock_key, owner_id, expires_at)
    VALUES (p_lock_key, p_owner_id, NOW() + (p_ttl_seconds || ' seconds')::INTERVAL)
    ON CONFLICT (lock_key) DO NOTHING;

    -- Check if we got the lock
    SELECT TRUE INTO acquired
    FROM catalog_locks
    WHERE lock_key = p_lock_key AND owner_id = p_owner_id;

    RETURN COALESCE(acquired, FALSE);
END;
$$ LANGUAGE plpgsql;

-- Function to release a lock
CREATE OR REPLACE FUNCTION release_lock(
    p_lock_key TEXT,
    p_owner_id TEXT
)
RETURNS BOOLEAN AS $$
DECLARE
    released BOOLEAN := FALSE;
BEGIN
    DELETE FROM catalog_locks
    WHERE lock_key = p_lock_key AND owner_id = p_owner_id;

    GET DIAGNOSTICS released = ROW_COUNT;
    RETURN released > 0;
END;
$$ LANGUAGE plpgsql;
