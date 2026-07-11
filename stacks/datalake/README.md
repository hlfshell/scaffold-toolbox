# scaffold toolbox data lake stack

Local lakehouse stack for scaffold. It wraps the Iceberg toolbox module: MinIO for object storage, Iceberg REST catalog for table metadata, and Trino for SQL queries.

## Install

```bash
go get github.com/hlfshell/scaffold-toolbox/stacks/datalake
```

```go
import "github.com/hlfshell/scaffold-toolbox/stacks/datalake"
```

## Example

```go
stack, err := datalake.NewStack("lake-dev",
	datalake.WithBucket("warehouse"),
)
if err != nil {
	return err
}

if err := stack.Create(ctx); err != nil {
	return err
}

result, err := stack.Query(ctx, "show catalogs")
```
