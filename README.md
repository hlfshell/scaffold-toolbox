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
- MongoDB - planned support for document-heavy services and JSON-shaped test data.
- ClickHouse - planned support for analytics workloads, event stores, and columnar query testing.

Caches:

- Redis - a general-purpose cache, queue-adjacent store, or coordination dependency for local apps.
- Memcached - a small cache service for apps that only need simple key/value caching.

Search and vectors:

- Qdrant - local vector search for embedding, retrieval, and RAG development.
- Weaviate - planned support for teams already building against Weaviate's object/schema model.

Object storage and cloud:

- MinIO - S3-compatible storage for files, documents, model artifacts, and test uploads.
- LocalStack - local AWS APIs for services like SQS without reaching out to real cloud accounts.
- AWS - planned higher-level setup for common LocalStack resources such as buckets, queues, and topics.

Data platforms:

- Trino - planned local query engine for testing SQL over object storage and lakehouse-style data.
- Iceberg - planned stack for data lake experiments that need storage, catalog, and query pieces together.

LLM services:

- Ollama - planned local model runtime for offline or laptop-friendly LLM development.
- LiteLLM - planned OpenAI-compatible proxy for testing apps across multiple model providers.

Orchestration:

- Kubernetes - planned local cluster helpers for projects that need to test against Kubernetes directly.
- Argo CD - planned GitOps control plane for local deployment and sync workflows.
- Argo Workflows - planned workflow engine for testing pipeline and job orchestration locally.

Stacks:

- RAG stack - a ready-made Postgres, Qdrant, and MinIO environment for document and retrieval apps.

Each module has its own README with the current status and usage notes.
