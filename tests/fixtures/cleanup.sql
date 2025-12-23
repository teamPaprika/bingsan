-- Cleanup script for test data
-- Run after tests to clean up

-- Delete test tables
DELETE FROM tables WHERE namespace_id IN (SELECT id FROM namespaces WHERE name LIKE 'test_%');

-- Delete test views
DELETE FROM views WHERE namespace_id IN (SELECT id FROM namespaces WHERE name LIKE 'test_%');

-- Delete test namespaces
DELETE FROM namespaces WHERE name LIKE 'test_%';

-- Delete test scan plans
DELETE FROM scan_plans WHERE namespace LIKE 'test_%';
