# scaffold toolbox rag stack

Local retrieval app stack for scaffold. It composes Postgres, Qdrant, and MinIO on a shared Docker network.

## Install

```bash
go get github.com/hlfshell/scaffold-toolbox/stacks/rag
```

```go
import "github.com/hlfshell/scaffold-toolbox/stacks/rag"
```

## Example

```go
stack, err := rag.NewStack("rag-dev",
	rag.WithPostgresSQL("create table documents (id serial primary key, body text);"),
	rag.WithQdrantCollection(qdrant.CollectionConfig{Name: "documents", Size: 1536}),
	rag.WithMinIOBucket("documents"),
)
if err != nil {
	return err
}

if err := stack.Create(ctx); err != nil {
	return err
}
defer stack.Cleanup(context.WithoutCancel(ctx))
```
