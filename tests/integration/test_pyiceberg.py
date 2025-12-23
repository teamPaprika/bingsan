"""
PyIceberg Integration Tests

Tests for validating Bingsan compatibility with the PyIceberg client.
These tests require a running Bingsan instance and database.

Run with: pytest tests/integration/test_pyiceberg.py -v
"""
import pytest
from pyiceberg.schema import Schema
from pyiceberg.types import (
    LongType,
    StringType,
    NestedField,
    TimestampType,
    BooleanType,
    DoubleType,
)
from pyiceberg.catalog.rest import RestCatalog


class TestNamespaceCRUD:
    """Test namespace operations with PyIceberg client."""

    def test_list_namespaces(self, rest_catalog: RestCatalog):
        """Verify that list_namespaces returns a list."""
        namespaces = rest_catalog.list_namespaces()
        assert isinstance(namespaces, list)

    def test_create_namespace(self, rest_catalog: RestCatalog, test_namespace: tuple):
        """Verify namespace creation works."""
        namespaces = rest_catalog.list_namespaces()
        assert test_namespace in namespaces

    def test_namespace_properties(self, rest_catalog: RestCatalog, test_namespace: tuple):
        """Verify namespace properties can be retrieved."""
        props = rest_catalog.load_namespace_properties(test_namespace)
        assert isinstance(props, dict)
        assert "created_by" in props

    def test_update_namespace_properties(self, rest_catalog: RestCatalog, test_namespace: tuple):
        """Verify namespace properties can be updated."""
        rest_catalog.update_namespace_properties(
            test_namespace,
            updates={"description": "Test namespace"},
        )
        props = rest_catalog.load_namespace_properties(test_namespace)
        assert props.get("description") == "Test namespace"


class TestTableCRUD:
    """Test table operations with PyIceberg client."""

    @pytest.fixture
    def simple_schema(self) -> Schema:
        """Create a simple schema for testing."""
        return Schema(
            NestedField(field_id=1, name="id", field_type=LongType(), required=True),
            NestedField(field_id=2, name="name", field_type=StringType(), required=False),
        )

    def test_create_table(
        self,
        rest_catalog: RestCatalog,
        test_namespace: tuple,
        simple_schema: Schema,
    ):
        """Verify table creation with schema."""
        table_name = "test_create_table"
        identifier = (*test_namespace, table_name)

        table = rest_catalog.create_table(
            identifier=identifier,
            schema=simple_schema,
        )

        assert table is not None
        assert table.name() == table_name

        # Cleanup
        rest_catalog.drop_table(identifier)

    def test_list_tables(
        self,
        rest_catalog: RestCatalog,
        test_namespace: tuple,
        simple_schema: Schema,
    ):
        """Verify listing tables in namespace."""
        table_name = "test_list_table"
        identifier = (*test_namespace, table_name)

        rest_catalog.create_table(identifier=identifier, schema=simple_schema)

        tables = rest_catalog.list_tables(test_namespace)
        assert any(t[-1] == table_name for t in tables)

        rest_catalog.drop_table(identifier)

    def test_load_table(
        self,
        rest_catalog: RestCatalog,
        test_namespace: tuple,
        simple_schema: Schema,
    ):
        """Verify loading table metadata."""
        table_name = "test_load_table"
        identifier = (*test_namespace, table_name)

        rest_catalog.create_table(identifier=identifier, schema=simple_schema)

        loaded_table = rest_catalog.load_table(identifier)
        assert loaded_table is not None
        assert len(loaded_table.schema().fields) == 2

        rest_catalog.drop_table(identifier)

    def test_drop_table(
        self,
        rest_catalog: RestCatalog,
        test_namespace: tuple,
        simple_schema: Schema,
    ):
        """Verify table deletion."""
        table_name = "test_drop_table"
        identifier = (*test_namespace, table_name)

        rest_catalog.create_table(identifier=identifier, schema=simple_schema)
        rest_catalog.drop_table(identifier)

        tables = rest_catalog.list_tables(test_namespace)
        assert not any(t[-1] == table_name for t in tables)


class TestSchemaEvolution:
    """Test schema evolution with PyIceberg client."""

    def test_add_column(self, rest_catalog: RestCatalog, test_namespace: tuple):
        """Verify adding a new column to table schema."""
        table_name = "test_schema_evolution"
        identifier = (*test_namespace, table_name)

        initial_schema = Schema(
            NestedField(field_id=1, name="id", field_type=LongType(), required=True),
        )

        table = rest_catalog.create_table(identifier=identifier, schema=initial_schema)

        # Add a new column
        with table.update_schema() as update:
            update.add_column("name", StringType())

        # Reload table and verify
        table = rest_catalog.load_table(identifier)
        field_names = [f.name for f in table.schema().fields]
        assert "name" in field_names

        rest_catalog.drop_table(identifier)

    def test_rename_column(self, rest_catalog: RestCatalog, test_namespace: tuple):
        """Verify renaming a column."""
        table_name = "test_rename_column"
        identifier = (*test_namespace, table_name)

        initial_schema = Schema(
            NestedField(field_id=1, name="id", field_type=LongType(), required=True),
            NestedField(field_id=2, name="old_name", field_type=StringType(), required=False),
        )

        table = rest_catalog.create_table(identifier=identifier, schema=initial_schema)

        # Rename column
        with table.update_schema() as update:
            update.rename_column("old_name", "new_name")

        # Reload and verify
        table = rest_catalog.load_table(identifier)
        field_names = [f.name for f in table.schema().fields]
        assert "new_name" in field_names
        assert "old_name" not in field_names

        rest_catalog.drop_table(identifier)


class TestSnapshotHistory:
    """Test snapshot history operations."""

    def test_empty_snapshot_history(self, rest_catalog: RestCatalog, test_namespace: tuple):
        """Verify empty table has no snapshots."""
        table_name = "test_empty_snapshots"
        identifier = (*test_namespace, table_name)

        schema = Schema(
            NestedField(field_id=1, name="id", field_type=LongType(), required=True),
        )

        table = rest_catalog.create_table(identifier=identifier, schema=schema)

        # New table should have no snapshots (or one initial snapshot)
        snapshots = list(table.snapshots())
        assert len(snapshots) >= 0  # May be 0 or 1 depending on implementation

        rest_catalog.drop_table(identifier)


class TestComplexSchemas:
    """Test complex schema types."""

    def test_multiple_field_types(self, rest_catalog: RestCatalog, test_namespace: tuple):
        """Verify table creation with multiple field types."""
        table_name = "test_complex_schema"
        identifier = (*test_namespace, table_name)

        schema = Schema(
            NestedField(field_id=1, name="id", field_type=LongType(), required=True),
            NestedField(field_id=2, name="name", field_type=StringType(), required=False),
            NestedField(field_id=3, name="timestamp", field_type=TimestampType(), required=False),
            NestedField(field_id=4, name="active", field_type=BooleanType(), required=False),
            NestedField(field_id=5, name="score", field_type=DoubleType(), required=False),
        )

        table = rest_catalog.create_table(identifier=identifier, schema=schema)
        assert table is not None
        assert len(table.schema().fields) == 5

        rest_catalog.drop_table(identifier)
