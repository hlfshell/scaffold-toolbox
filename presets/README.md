# scaffold presets

Preset stacks compose existing scaffold harnesses. They do not duplicate service implementation.

## RAG stack

The RAG stack includes Postgres, Qdrant, and MinIO for local RAG app development, agent tests, embedding tests, and document/object workflows.

```go
rag, err := presets.NewRAGStack("rag-dev",
	presets.WithPostgresSQL("create table documents (id serial primary key, body text);"),
	presets.WithQdrantCollection(qdrant.CollectionConfig{Name: "documents", Size: 1536}),
	presets.WithMinIOBucket("documents"),
)
if err != nil {
	return err
}

err = rag.Create(ctx)
if err != nil {
	return err
}
defer rag.Cleanup(context.WithoutCancel(ctx))
```

Cleanup is delegated to the underlying scaffold stack.
