# scaffold toolbox analytics stack

Local analytics stack for scaffold. It combines ClickHouse for analytical storage with Trino for SQL client, catalog, and federation testing.

## Install

```bash
go get github.com/hlfshell/scaffold-toolbox/stacks/analytics
```

```go
import "github.com/hlfshell/scaffold-toolbox/stacks/analytics"
```

## Example

```go
stack, err := analytics.NewStack("analytics-dev",
	analytics.WithClickHouseSQL("create table events (id UInt64, name String) engine = Memory;"),
	analytics.WithTrinoMemoryCatalog("scratch"),
)
if err != nil {
	return err
}

if err := stack.Create(ctx); err != nil {
	return err
}

err = stack.ExecClickHouse(ctx, "insert into events values (1, 'signup')")
rows, err := stack.QueryTrino(ctx, "select * from scratch.default.example")
```
