-- Namespace-level Access Control
-- Supports principal-based permissions on namespaces

CREATE TABLE namespace_permissions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    namespace_id UUID NOT NULL REFERENCES namespaces(id) ON DELETE CASCADE,
    principal_id TEXT NOT NULL,      -- API key ID or OAuth client ID
    principal_type TEXT NOT NULL,    -- 'api_key' or 'oauth2'
    permissions TEXT[] NOT NULL DEFAULT '{}',  -- ['read', 'write', 'manage']
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(namespace_id, principal_id, principal_type)
);

-- Indexes for efficient lookups
CREATE INDEX idx_namespace_perms_principal
    ON namespace_permissions(principal_id, principal_type);
CREATE INDEX idx_namespace_perms_namespace
    ON namespace_permissions(namespace_id);

-- Trigger to update updated_at
CREATE TRIGGER update_namespace_permissions_updated_at
    BEFORE UPDATE ON namespace_permissions
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Function to check if a principal has a permission on a namespace
CREATE OR REPLACE FUNCTION check_namespace_permission(
    p_namespace_name TEXT[],
    p_principal_id TEXT,
    p_principal_type TEXT,
    p_permission TEXT
)
RETURNS BOOLEAN AS $$
DECLARE
    has_permission BOOLEAN := FALSE;
BEGIN
    SELECT TRUE INTO has_permission
    FROM namespace_permissions np
    JOIN namespaces n ON n.id = np.namespace_id
    WHERE n.name = p_namespace_name
      AND np.principal_id = p_principal_id
      AND np.principal_type = p_principal_type
      AND p_permission = ANY(np.permissions);

    RETURN COALESCE(has_permission, FALSE);
END;
$$ LANGUAGE plpgsql;
