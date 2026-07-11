# scaffold stacks

Ready-made stacks compose existing scaffold toolbox services into useful local environments. Each stack lives in its own subpackage so applications can import only the environment they need.

## Available stacks

- `github.com/hlfshell/scaffold-toolbox/stacks/rag` - Postgres, Qdrant, and MinIO for retrieval and document apps.
- `github.com/hlfshell/scaffold-toolbox/stacks/workflow` - Argo Workflows on a local k3s cluster with manifest and image helpers.
- `github.com/hlfshell/scaffold-toolbox/stacks/analytics` - ClickHouse and Trino for analytical storage and SQL query testing.
- `github.com/hlfshell/scaffold-toolbox/stacks/datalake` - MinIO, Iceberg REST catalog, and Trino wired into a local lakehouse.

The AWS toolbox is intentionally not represented here as a stack package. It is already a complete MiniStack-backed service with resource, ECS, image, and SDK helpers. App-shaped AWS setups are documented in `aws/examples` so the base AWS API stays the source of truth.
