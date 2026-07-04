# scaffold toolbox

<p align="center">
  <img src="scaffold.png" alt="scaffold logo" width="240">
</p>

Premade services and stacks for [scaffold](https://github.com/hlfshell/scaffold).

Each toolbox entry is its own Go module so applications can import only the pieces they need. This repository groups those modules together for development and release, but consumers can depend on individual packages directly.

## Available modules

Databases:

- Postgres - relational database service with SQL preload helpers.
- MySQL - relational database service with SQL preload helpers.
- MongoDB - planned document database service with document preload helpers.
- ClickHouse - planned analytical database service with SQL preload helpers.

Caches:

- Redis - key/value cache service with key and seed-function helpers.
- Memcached - cache service with item preload helpers.

Search and vectors:

- Qdrant - vector database service with collection and point preload helpers.
- Weaviate - planned vector database service with schema/object preload helpers.

Object storage and cloud:

- MinIO - S3-compatible object storage service with bucket and object preload helpers.
- LocalStack - local AWS service emulator with AWS SDK configuration helpers.
- AWS - planned helper stack built around LocalStack for common AWS resources.

Data platforms:

- Trino - planned distributed query engine service with catalog/schema helpers.
- Iceberg - planned local data lake stack for object storage, catalog, and query workflows.

LLM services:

- Ollama - planned local model server service with model preload helpers.
- LiteLLM - planned OpenAI-compatible proxy service with provider configuration helpers.

Orchestration:

- Kubernetes - planned local Kubernetes service/stack helpers with kubeconfig and manifest support.
- Argo CD - planned GitOps helper stack for local Kubernetes workflows.
- Argo Workflows - planned workflow helper stack for local Kubernetes workflows.

Presets:

- RAG stack - composed Postgres, Qdrant, and MinIO stack for local retrieval-augmented generation workflows.

Each module has its own README with the current status and usage notes.
