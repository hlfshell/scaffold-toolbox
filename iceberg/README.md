# scaffold toolbox iceberg

Local Iceberg lakehouse stack for scaffold. It composes MinIO for object
storage, an Iceberg REST catalog, and Trino configured with an Iceberg
catalog on a shared Docker network.

## Install

```bash
go get github.com/hlfshell/scaffold-toolbox/iceberg
```

```go
import "github.com/hlfshell/scaffold-toolbox/iceberg"
```

## Example

```go
lake, err := iceberg.NewStack("lake", iceberg.WithBucket("warehouse"))
if err != nil {
	return err
}

stack := scaffold.NewStack("app", scaffold.WithServices(lake))
```
