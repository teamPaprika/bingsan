-- Revert namespace-level access control

DROP FUNCTION IF EXISTS check_namespace_permission(TEXT[], TEXT, TEXT, TEXT);
DROP TRIGGER IF EXISTS update_namespace_permissions_updated_at ON namespace_permissions;
DROP TABLE IF EXISTS namespace_permissions;
