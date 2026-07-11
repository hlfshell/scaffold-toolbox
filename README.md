# scaffold toolbox

<p align="center">
  <img src="scaffold.png" alt="scaffold logo" width="320">
</p>

Premade services and stacks for [scaffold](https://github.com/hlfshell/scaffold).

Each toolbox entry is its own Go module so applications can import only the pieces they need. This repository groups those modules together for development and release, but consumers can depend on individual packages directly.

## Available modules

Databases:

- Postgres - the default choice for app and API tests that need a real relational database.
- MySQL - useful when production compatibility matters more than using Postgres locally.
- MongoDB - document database service with official Go client access and document preload helpers.
- ClickHouse - analytics database service with HTTP/native endpoints and SQL preload helpers.

Caches:

- Redis - a general-purpose cache, queue-adjacent store, or coordination dependency for local apps.
- Memcached - a small cache service for apps that only need simple key/value caching.

Search and vectors:

- Qdrant - local vector search for embedding, retrieval, and RAG development.
- Weaviate - vector database service with schema and object preload helpers.

Object storage and cloud:

- MinIO - S3-compatible storage for files, documents, model artifacts, and test uploads.
- AWS - [MiniStack](https://ministack.org/)-backed local AWS stack with setup helpers for common AWS resources, SDK config, ECS-style containers, and local image workflows.

Data platforms:

- Trino - local SQL query engine with generated catalog files and an HTTP query helper.
- Iceberg - local lakehouse stack composed from MinIO, Iceberg REST catalog, and Trino.

LLM services:

- Ollama - local model runtime with endpoint helpers and optional model pulls.
- LiteLLM - OpenAI-compatible proxy for testing apps across multiple model providers.

Orchestration:

- Kubernetes - Docker-backed k3s quickstart with host kubeconfig, manifest loading, status, and kubectl passthrough.
- Argo CD - k3s-backed quickstart that installs Argo CD and application manifests.
- Argo Workflows - k3s-backed quickstart that installs Argo Workflows and workflow manifests.

Stacks:

- RAG stack - a ready-made Postgres, Qdrant, and MinIO environment for document and retrieval apps.

Each module has its own README with the current status and usage notes.

## Testing

Run the toolbox test script from the repository root:

```bash
./test.sh
```
