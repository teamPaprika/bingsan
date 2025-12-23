-- Test fixtures for namespace operations
-- Run these SQL statements to set up test data

-- Clean up any existing test data
DELETE FROM tables WHERE namespace_id IN (SELECT id FROM namespaces WHERE name LIKE 'test_%');
DELETE FROM views WHERE namespace_id IN (SELECT id FROM namespaces WHERE name LIKE 'test_%');
DELETE FROM namespaces WHERE name LIKE 'test_%';

-- Create test namespaces
INSERT INTO namespaces (name, properties, created_at, updated_at)
VALUES
    ('test_ns1', '{"owner": "test", "location": "s3://test/ns1"}', NOW(), NOW()),
    ('test_ns2', '{"owner": "test", "location": "s3://test/ns2"}', NOW(), NOW()),
    ('test_ns3', '{}', NOW(), NOW())
ON CONFLICT (name) DO NOTHING;
