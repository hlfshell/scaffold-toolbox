#!/usr/bin/env bash
set -euo pipefail

export SCAFFOLD_TOOLBOX_KUBERNETES_TESTS=1
export SCAFFOLD_TOOLBOX_ARGO_TESTS=1
export SCAFFOLD_TOOLBOX_LLM_TESTS=1
export SCAFFOLD_AWS_ECS_INTEGRATION=1

go test \
  ./postgres/... \
  ./mysql/... \
  ./redis/... \
  ./memcached/... \
  ./qdrant/... \
  ./minio/... \
  ./stacks/... \
  ./mongo/... \
  ./clickhouse/... \
  ./weaviate/... \
  ./trino/... \
  ./iceberg/... \
  ./aws/... \
  ./ollama/... \
  ./litellm/... \
  ./kubernetes/... \
  ./argocd/... \
  ./argo-workflows/... \
  -count=1 \
  -timeout 30m
