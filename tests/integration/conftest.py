"""
PyIceberg integration test fixtures.

This module provides pytest fixtures for testing Bingsan with PyIceberg client.
"""
import os
import uuid
from typing import Generator

import pytest
from pyiceberg.catalog.rest import RestCatalog


def get_env(key: str, default: str) -> str:
    """Get environment variable with default."""
    return os.environ.get(key, default)


@pytest.fixture(scope="session")
def catalog_uri() -> str:
    """Get the catalog URI from environment or use default."""
    return get_env("ICEBERG_CATALOG_URI", "http://localhost:8181")


@pytest.fixture(scope="session")
def s3_endpoint() -> str:
    """Get the S3 endpoint from environment or use default."""
    return get_env("ICEBERG_S3_ENDPOINT", "http://localhost:9002")


@pytest.fixture(scope="session")
def s3_access_key() -> str:
    """Get the S3 access key from environment or use default."""
    return get_env("ICEBERG_S3_ACCESS_KEY", "minioadmin")


@pytest.fixture(scope="session")
def s3_secret_key() -> str:
    """Get the S3 secret key from environment or use default."""
    return get_env("ICEBERG_S3_SECRET_KEY", "minioadmin")


@pytest.fixture(scope="session")
def rest_catalog(
    catalog_uri: str,
    s3_endpoint: str,
    s3_access_key: str,
    s3_secret_key: str,
) -> RestCatalog:
    """
    Create a REST catalog connection to Bingsan.

    This fixture is session-scoped to reuse the catalog connection
    across all tests in the session.
    """
    catalog = RestCatalog(
        name="bingsan-test",
        **{
            "uri": catalog_uri,
            "warehouse": "s3://warehouse/",
            "s3.endpoint": s3_endpoint,
            "s3.access-key-id": s3_access_key,
            "s3.secret-access-key": s3_secret_key,
            "s3.path-style-access": "true",
        }
    )
    return catalog


@pytest.fixture(scope="function")
def test_namespace(rest_catalog: RestCatalog) -> Generator[tuple[str, ...], None, None]:
    """
    Create a temporary namespace for each test.

    The namespace is automatically cleaned up after the test completes.
    """
    ns_name = f"test_ns_{uuid.uuid4().hex[:8]}"
    ns = (ns_name,)

    # Create namespace
    rest_catalog.create_namespace(ns, properties={"created_by": "pytest"})

    yield ns

    # Cleanup: drop all tables in namespace first
    try:
        tables = rest_catalog.list_tables(ns)
        for table_id in tables:
            try:
                rest_catalog.drop_table(table_id)
            except Exception:
                pass  # Ignore errors during cleanup

        # Drop the namespace
        rest_catalog.drop_namespace(ns)
    except Exception:
        pass  # Ignore errors during cleanup


@pytest.fixture(scope="function")
def test_namespace_hierarchical(rest_catalog: RestCatalog) -> Generator[tuple[str, ...], None, None]:
    """
    Create a hierarchical namespace for testing nested namespaces.
    """
    parent = f"test_parent_{uuid.uuid4().hex[:8]}"
    child = f"child_{uuid.uuid4().hex[:8]}"
    ns = (parent, child)

    # Create parent namespace first
    rest_catalog.create_namespace((parent,))
    # Create child namespace
    rest_catalog.create_namespace(ns, properties={"created_by": "pytest"})

    yield ns

    # Cleanup
    try:
        tables = rest_catalog.list_tables(ns)
        for table_id in tables:
            try:
                rest_catalog.drop_table(table_id)
            except Exception:
                pass

        rest_catalog.drop_namespace(ns)
        rest_catalog.drop_namespace((parent,))
    except Exception:
        pass
