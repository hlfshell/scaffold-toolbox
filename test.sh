#!/usr/bin/env bash
set -euo pipefail

modules=(
  postgres
  mysql
  redis
  memcached
  qdrant
  minio
  stacks/rag
  stacks/workflow
  stacks/analytics
  stacks/datalake
  mongo
  clickhouse
  weaviate
  trino
  iceberg
  aws
  ollama
  litellm
  kubernetes
  argocd
  argo-workflows
)

for module in "${modules[@]}"; do
  echo "==> ${module}"
  (
    cd "${module}"
    go test ./... -count=1 -timeout 30m
  )
done
