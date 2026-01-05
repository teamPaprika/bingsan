-- Initialize separate databases for benchmark comparison
-- This script runs on PostgreSQL startup

-- Create database and user for Bingsan
CREATE USER bingsan WITH PASSWORD 'bingsan';
CREATE DATABASE bingsan_catalog OWNER bingsan;
GRANT ALL PRIVILEGES ON DATABASE bingsan_catalog TO bingsan;

-- Create database and user for Lakekeeper
CREATE USER lakekeeper WITH PASSWORD 'lakekeeper';
CREATE DATABASE lakekeeper_catalog OWNER lakekeeper;
GRANT ALL PRIVILEGES ON DATABASE lakekeeper_catalog TO lakekeeper;
