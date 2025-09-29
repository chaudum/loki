import sys
from pyiceberg.catalog import load_catalog, NoSuchTableError, NoSuchNamespaceError
from pyiceberg.schema import Schema
from pyiceberg.table.sorting import SortOrder, SortField
from pyiceberg.types import NestedField, StringType, TimestampType, MapType
from pyiceberg.partitioning import PartitionSpec, PartitionField
from pyiceberg.transforms import DayTransform, IdentityTransform


def cleanup(uri: str, namespace: str, table_name: str):
    catalog = load_catalog("logslake", uri=uri)
    try:
        catalog.drop_table((namespace, table_name))
        catalog.drop_namespace(namespace)
    except (NoSuchTableError, NoSuchNamespaceError):
        pass
    else:
        print("Successfully cleaned table and namespace")


def setup(uri: str, namespace: str, table_name: str):
    catalog = load_catalog("logslake", uri=uri)

    catalog.create_namespace_if_not_exists(namespace)
    print("Successfully created namespace")

    # Define schema
    schema = Schema(
        NestedField(1, "timestamp", TimestampType(), required=True),
        NestedField(2, "message", StringType(), required=True),
        NestedField(3, "metadata", MapType(key_id=1, key_type=StringType(), value_id=2, value_type=StringType(), value_required=True), required=False),
        NestedField(4, "service_name", StringType(), required=True),
        NestedField(5, "detected_level", StringType(), required=True),
    )

    # Define partition spec
    partition_spec = PartitionSpec(
        # PartitionField(source_id=1, field_id=1001, transform=DayTransform(), name="day"),
        PartitionField(source_id=4, field_id=1004, transform=IdentityTransform(), name="service_name"),
        PartitionField(source_id=5, field_id=1005, transform=IdentityTransform(), name="detected_level"),
    )

    sort_order = SortOrder(
        SortField(source_id=1, transform=IdentityTransform(), direction="asc", null_order="nulls-first"),
    )

    catalog.create_table_if_not_exists(
        identifier=(namespace, table_name),
        schema=schema,
        partition_spec=partition_spec,
        sort_order=sort_order,
        properties={
            "write.format.default": "parquet",
            "write.parquet.compression-codec": "zstd",
            "write.target-file-size-bytes": "134217728"
        }
    )
    print("Successfully created table")


if __name__ == "__main__":
    uri: str = "http://localhost:8181"
    namespace: str = "loki"
    table_name: str = "chunks"

    args = sys.argv[1:]
    if len(args) > 0:
        if args[0] == "drop":
            cleanup(uri, namespace, table_name)
        elif args[0] == "create":
            setup(uri, namespace, table_name)
        else:
            print("invalid command: ", args[0])
    else:
        print("missing argument")

