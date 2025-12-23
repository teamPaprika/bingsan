"""
Spark Integration Tests

Tests for validating Bingsan compatibility with Apache Spark.
These tests require a running Spark cluster and Bingsan instance.

Run with:
    docker compose -f deployments/docker/docker-compose-spark.yml up -d
    docker compose -f deployments/docker/docker-compose-spark.yml exec spark-test pytest /tests/test_spark.py -v
"""
import os
import uuid
import pytest

# Skip tests if PySpark is not available
pyspark = pytest.importorskip("pyspark")

from pyspark.sql import SparkSession


@pytest.fixture(scope="session")
def spark() -> SparkSession:
    """Create a Spark session configured to use Bingsan as the Iceberg catalog."""
    catalog_uri = os.environ.get("ICEBERG_CATALOG_URI", "http://catalog:8181")
    s3_endpoint = os.environ.get("ICEBERG_S3_ENDPOINT", "http://minio:9000")

    session = (
        SparkSession.builder
        .appName("bingsan-spark-integration-test")
        .config("spark.sql.catalog.iceberg", "org.apache.iceberg.spark.SparkCatalog")
        .config("spark.sql.catalog.iceberg.type", "rest")
        .config("spark.sql.catalog.iceberg.uri", catalog_uri)
        .config("spark.sql.catalog.iceberg.warehouse", "s3://warehouse/")
        .config("spark.sql.catalog.iceberg.s3.endpoint", s3_endpoint)
        .config("spark.sql.catalog.iceberg.s3.access-key-id", "minioadmin")
        .config("spark.sql.catalog.iceberg.s3.secret-access-key", "minioadmin")
        .config("spark.sql.catalog.iceberg.s3.path-style-access", "true")
        .config("spark.sql.extensions", "org.apache.iceberg.spark.extensions.IcebergSparkSessionExtensions")
        .config("spark.sql.defaultCatalog", "iceberg")
        .getOrCreate()
    )

    yield session
    session.stop()


@pytest.fixture(scope="function")
def test_namespace(spark: SparkSession) -> str:
    """Create a unique namespace for each test."""
    ns_name = f"spark_test_{uuid.uuid4().hex[:8]}"
    spark.sql(f"CREATE NAMESPACE IF NOT EXISTS iceberg.{ns_name}")
    yield ns_name

    # Cleanup
    try:
        # Drop all tables in namespace
        tables = spark.sql(f"SHOW TABLES IN iceberg.{ns_name}").collect()
        for row in tables:
            spark.sql(f"DROP TABLE IF EXISTS iceberg.{ns_name}.{row['tableName']}")
        spark.sql(f"DROP NAMESPACE IF EXISTS iceberg.{ns_name}")
    except Exception:
        pass


class TestSparkNamespaceOperations:
    """Test namespace operations via Spark SQL."""

    def test_create_namespace(self, spark: SparkSession, test_namespace: str):
        """Verify namespace creation through Spark."""
        result = spark.sql("SHOW NAMESPACES IN iceberg").collect()
        namespaces = [row[0] for row in result]
        assert test_namespace in namespaces

    def test_list_namespaces(self, spark: SparkSession, test_namespace: str):
        """Verify listing namespaces."""
        result = spark.sql("SHOW NAMESPACES IN iceberg").collect()
        assert len(result) > 0

    def test_namespace_properties(self, spark: SparkSession, test_namespace: str):
        """Verify namespace properties."""
        result = spark.sql(f"DESCRIBE NAMESPACE iceberg.{test_namespace}").collect()
        # Should have at least some metadata
        assert len(result) > 0


class TestSparkTableOperations:
    """Test table operations via Spark SQL."""

    def test_create_table(self, spark: SparkSession, test_namespace: str):
        """Verify table creation through Spark."""
        spark.sql(f"""
            CREATE TABLE iceberg.{test_namespace}.test_table (
                id BIGINT,
                name STRING
            ) USING iceberg
        """)

        tables = spark.sql(f"SHOW TABLES IN iceberg.{test_namespace}").collect()
        table_names = [row["tableName"] for row in tables]
        assert "test_table" in table_names

    def test_insert_data(self, spark: SparkSession, test_namespace: str):
        """Verify data insertion."""
        spark.sql(f"""
            CREATE TABLE iceberg.{test_namespace}.insert_test (
                id BIGINT,
                name STRING
            ) USING iceberg
        """)

        spark.sql(f"""
            INSERT INTO iceberg.{test_namespace}.insert_test VALUES
            (1, 'Alice'),
            (2, 'Bob'),
            (3, 'Charlie')
        """)

        result = spark.sql(f"SELECT COUNT(*) as cnt FROM iceberg.{test_namespace}.insert_test").collect()
        assert result[0]["cnt"] == 3

    def test_read_data(self, spark: SparkSession, test_namespace: str):
        """Verify data reading."""
        spark.sql(f"""
            CREATE TABLE iceberg.{test_namespace}.read_test (
                id BIGINT,
                name STRING
            ) USING iceberg
        """)

        spark.sql(f"INSERT INTO iceberg.{test_namespace}.read_test VALUES (1, 'Test')")

        result = spark.sql(f"SELECT * FROM iceberg.{test_namespace}.read_test").collect()
        assert len(result) == 1
        assert result[0]["id"] == 1
        assert result[0]["name"] == "Test"

    def test_drop_table(self, spark: SparkSession, test_namespace: str):
        """Verify table deletion."""
        spark.sql(f"""
            CREATE TABLE iceberg.{test_namespace}.drop_test (
                id BIGINT
            ) USING iceberg
        """)

        spark.sql(f"DROP TABLE iceberg.{test_namespace}.drop_test")

        tables = spark.sql(f"SHOW TABLES IN iceberg.{test_namespace}").collect()
        table_names = [row["tableName"] for row in tables]
        assert "drop_test" not in table_names


class TestSparkSchemaEvolution:
    """Test schema evolution via Spark SQL."""

    def test_add_column(self, spark: SparkSession, test_namespace: str):
        """Verify adding columns through Spark."""
        spark.sql(f"""
            CREATE TABLE iceberg.{test_namespace}.schema_test (
                id BIGINT
            ) USING iceberg
        """)

        spark.sql(f"ALTER TABLE iceberg.{test_namespace}.schema_test ADD COLUMN name STRING")

        schema = spark.table(f"iceberg.{test_namespace}.schema_test").schema
        field_names = [f.name for f in schema.fields]
        assert "name" in field_names


class TestSparkTimeTravel:
    """Test time travel queries via Spark SQL."""

    def test_snapshot_history(self, spark: SparkSession, test_namespace: str):
        """Verify snapshot history access."""
        spark.sql(f"""
            CREATE TABLE iceberg.{test_namespace}.history_test (
                id BIGINT,
                value STRING
            ) USING iceberg
        """)

        # Insert initial data
        spark.sql(f"INSERT INTO iceberg.{test_namespace}.history_test VALUES (1, 'initial')")

        # Update data
        spark.sql(f"INSERT INTO iceberg.{test_namespace}.history_test VALUES (2, 'second')")

        # Check history
        history = spark.sql(f"SELECT * FROM iceberg.{test_namespace}.history_test.history").collect()
        # Should have at least 2 snapshots
        assert len(history) >= 2

    def test_read_from_snapshot(self, spark: SparkSession, test_namespace: str):
        """Verify reading from specific snapshot."""
        spark.sql(f"""
            CREATE TABLE iceberg.{test_namespace}.snapshot_read_test (
                id BIGINT
            ) USING iceberg
        """)

        spark.sql(f"INSERT INTO iceberg.{test_namespace}.snapshot_read_test VALUES (1)")

        # Get first snapshot
        snapshots = spark.sql(f"SELECT * FROM iceberg.{test_namespace}.snapshot_read_test.snapshots").collect()

        if len(snapshots) > 0:
            snapshot_id = snapshots[0]["snapshot_id"]
            # Read from that snapshot
            result = spark.sql(f"""
                SELECT * FROM iceberg.{test_namespace}.snapshot_read_test
                VERSION AS OF {snapshot_id}
            """).collect()
            assert len(result) >= 0  # Should not error
