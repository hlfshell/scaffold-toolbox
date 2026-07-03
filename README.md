# scaffold toolbox

Premade services and stacks for [scaffold](https://github.com/hlfshell/scaffold).

Each toolbox entry is its own Go module so applications can import only the pieces they need. This repository groups those modules together for development and release, but consumers can depend on individual packages directly.

## Available modules

- `github.com/hlfshell/scaffold-toolbox/postgres` - Postgres service.
- `github.com/hlfshell/scaffold-toolbox/mysql` - MySQL service.
- `github.com/hlfshell/scaffold-toolbox/redis` - Redis service.
- `github.com/hlfshell/scaffold-toolbox/memcached` - Memcached service.
- `github.com/hlfshell/scaffold-toolbox/qdrant` - Qdrant service.
- `github.com/hlfshell/scaffold-toolbox/minio` - MinIO service.
- `github.com/hlfshell/scaffold-toolbox/localstack` - LocalStack service.
- `github.com/hlfshell/scaffold-toolbox/presets` - composed preset stacks.
- `github.com/hlfshell/scaffold-toolbox/mongo` - planned MongoDB service.
- `github.com/hlfshell/scaffold-toolbox/clickhouse` - planned ClickHouse service.
- `github.com/hlfshell/scaffold-toolbox/weaviate` - planned Weaviate service.
- `github.com/hlfshell/scaffold-toolbox/trino` - planned Trino service.
- `github.com/hlfshell/scaffold-toolbox/iceberg` - planned Iceberg service/stack helpers.
- `github.com/hlfshell/scaffold-toolbox/aws` - planned AWS helper stack.
- `github.com/hlfshell/scaffold-toolbox/ollama` - planned Ollama service.
- `github.com/hlfshell/scaffold-toolbox/litellm` - planned LiteLLM service.
- `github.com/hlfshell/scaffold-toolbox/kubernetes` - planned Kubernetes service/stack helpers.
- `github.com/hlfshell/scaffold-toolbox/argocd` - planned Argo CD helpers.
- `github.com/hlfshell/scaffold-toolbox/argo-workflows` - planned Argo Workflows helpers.

Each module has its own README with the current status and usage notes.
