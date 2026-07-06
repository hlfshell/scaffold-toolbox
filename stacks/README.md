# scaffold stacks

Ready-made stacks compose existing scaffold harnesses. They do not duplicate service implementation.

## Install

```bash
go get github.com/hlfshell/scaffold-toolbox/stacks
```

```go
import "github.com/hlfshell/scaffold-toolbox/stacks"
```

## RAG stack

The RAG stack includes Postgres, Qdrant, and MinIO for local RAG app development, agent tests, embedding tests, and document/object workflows.

```go
rag, err := stacks.NewRAGStack("rag-dev",
	stacks.WithPostgresSQL("create table documents (id serial primary key, body text);"),
	stacks.WithQdrantCollection(qdrant.CollectionConfig{Name: "documents", Size: 1536}),
	stacks.WithMinIOBucket("documents"),
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
